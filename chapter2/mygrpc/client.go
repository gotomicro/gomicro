package mygrpc

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

type ClientComponent struct {
	IsFailFast bool
}

func (c *ClientComponent) NewGRPCClient(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	grpclog.SetLoggerV2(zapgrpc.NewLogger(grpcLogger))
	opts = append(opts, grpc.WithChainUnaryInterceptor(debugUnaryClientInterceptor()))
	// 开启fail fast
	if c.IsFailFast {
		ctx, _ = context.WithTimeout(ctx, time.Second)
		opts = append(opts, grpc.WithBlock())
		opts = append(opts, grpc.FailOnNonTempDialError(true))
	}
	conn, err = grpc.DialContext(ctx, target, opts...)
	if err != nil {
		LoggerPanic("连接错误", zap.String("component", "grpcClient"), zap.String("target", target), zap.Error(err))
	}
	return
}
