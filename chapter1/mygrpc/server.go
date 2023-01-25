package mygrpc

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
)

type App struct {
	*grpc.Server
	sigChan chan os.Signal
}

func NewApp() *App {
	var app App
	app.sigChan = make(chan os.Signal, 1)
	app.Server = grpc.NewServer()
	return &app
}

func (app *App) Start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	log.Println("服务启动监听：" + address)
	app.hookSignals()
	if err := app.Serve(lis); err != nil {
		return err
	}
	return nil
}

func (app *App) gracefulStop() {
	app.GracefulStop()
	log.Println("服务优雅退出")
}

func (app *App) stop() {
	app.Stop()
	log.Println("服务强制退出")
}

func (app *App) hookSignals() {
	signal.Notify(
		app.sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGSTOP,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGKILL,
	)

	go func() {
		var sig os.Signal
		for {
			sig = <-app.sigChan
			switch sig {
			case syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGSTOP, syscall.SIGUSR1:
				app.gracefulStop()
			case syscall.SIGINT, syscall.SIGKILL, syscall.SIGUSR2, syscall.SIGTERM:
				app.stop()
			}
		}
	}()
}
