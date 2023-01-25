package mygrpc

import (
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

type App struct {
	s       *grpc.Server
	sigChan chan os.Signal
}

func NewApp() *App {
	var app App
	app.sigChan = make(chan os.Signal, 1)
	app.s = grpc.NewServer()
	return &app
}

func (app *App) Start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	app.hookSignals()
	if err := app.s.Serve(lis); err != nil {
		return err
	}
	return nil
}

func (app *App) gracefulStop() {
	app.s.GracefulStop()
}

func (app *App) stop() {
	app.s.Stop()
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
			time.Sleep(time.Second)
			switch sig {
			case syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGSTOP, syscall.SIGUSR1:
				app.gracefulStop()
			case syscall.SIGINT, syscall.SIGKILL, syscall.SIGUSR2, syscall.SIGTERM:
				app.stop()
			}
			time.Sleep(time.Second)
		}
	}()
}
