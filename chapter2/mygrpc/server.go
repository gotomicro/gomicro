package mygrpc

import (
	"net"

	"go.uber.org/zap"
	"go.uber.org/zap/zapgrpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

type App struct {
	*grpc.Server
	logger *zap.Logger
}

func NewApp() *App {
	var app App
	// grpc框架日志，因为官方grpc日志是单例，所以这里要处理下
	grpclog.SetLoggerV2(zapgrpc.NewLogger(grpcLogger))
	app.Server = grpc.NewServer(grpc.ChainUnaryInterceptor(defaultUnaryServerInterceptor("grpcServer")))
	return &app
}

func (app *App) Start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	DefaultLogger.Info("服务启动监听：" + address)
	if err := app.Serve(lis); err != nil {
		return err
	}
	return nil
}
