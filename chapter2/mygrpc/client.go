package mygrpc

import (
	"context"
	"time"

	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

func NewGRPCClient(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	grpclog.SetLoggerV2(zapgrpc.NewLogger(grpcLogger))
	ctx, _ = context.WithTimeout(ctx, time.Second)
	opts = append(opts, grpc.WithChainUnaryInterceptor(debugUnaryClientInterceptor()))
	opts = append(opts, grpc.WithBlock())
	opts = append(opts, grpc.FailOnNonTempDialError(true))
	return grpc.DialContext(ctx, target, opts...)
}
