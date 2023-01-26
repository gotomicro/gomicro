package mygrpc

import (
	"context"

	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

func NewGRPCClient(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	grpclog.SetLoggerV2(zapgrpc.NewLogger(grpcLogger))
	opts = append(opts, grpc.WithChainUnaryInterceptor(debugUnaryClientInterceptor("grpcClient", target)))
	return grpc.DialContext(ctx, target, opts...)
}
