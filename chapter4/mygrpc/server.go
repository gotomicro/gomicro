package mygrpc

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapgrpc"
	"gomicro/chapter4/mygrpc/governor"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

type App struct {
	*grpc.Server
	governor *governor.Component
	sigChan  chan os.Signal
	logger   *zap.Logger
	opts     ServerOption
}

func NewApp(opts ...ServerOptions) *App {
	var app App
	app.logger = DefaultLogger
	app.sigChan = make(chan os.Signal, 1)
	for _, apply := range opts {
		apply(&app.opts)
	}
	app.governor = governor.NewComponent("0.0.0.0:9003", DefaultLogger)
	// grpc框架日志，因为官方grpc日志是单例，所以这里要处理下
	grpclog.SetLoggerV2(zapgrpc.NewLogger(grpcLogger))
	app.Server = grpc.NewServer(grpc.ChainUnaryInterceptor(defaultUnaryServerInterceptor()))
	return &app
}

func (app *App) Start() error {
	go app.governor.Start()
	lis, err := net.Listen("tcp", app.opts.address)
	if err != nil {
		return err
	}
	//registry
	if app.opts.registry != nil {
		err := app.opts.registry.Register(
			context.Background(),
			app.opts.serverName,
			app.opts.address,
		)
		if err != nil {
			return err
		}
	} else {
		app.logger.Info("registry is nil")
	}
	app.hookSignals()

	DefaultLogger.Info("服务启动监听：" + app.opts.address)
	if err := app.Serve(lis); err != nil {
		return err
	}
	return nil
}

//Stop stop tht server
func (app *App) gracefulStop() {
	if app.opts.registry != nil {
		_ = app.opts.registry.Unregister(
			context.TODO(),
			app.opts.serverName,
			app.opts.address,
		)
		app.opts.registry.Close()
	}
	app.logger.Info("Receive Signal gracefulStop")
	app.GracefulStop()
	app.governor.GracefulStop(context.Background())
}

func (app *App) stop() {
	if app.opts.registry != nil {
		_ = app.opts.registry.Unregister(
			context.TODO(),
			app.opts.serverName,
			app.opts.address,
		)
		app.opts.registry.Close()
	}
	app.logger.Info("Receive Signal stop")

	app.Stop()
	app.governor.Stop()

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
				app.gracefulStop() // graceful stop
			case syscall.SIGINT, syscall.SIGKILL, syscall.SIGUSR2, syscall.SIGTERM:
				app.stop() // terminalte now
			}
		}
	}()
}
