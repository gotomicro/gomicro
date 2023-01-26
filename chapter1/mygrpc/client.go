package mygrpc

import (
	"context"

	"google.golang.org/grpc"
)

func NewGRPCClient(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	return grpc.DialContext(ctx, target, opts...)
}
