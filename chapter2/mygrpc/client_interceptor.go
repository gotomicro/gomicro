package mygrpc

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// debugUnaryClientInterceptor returns grpc unary request request and response details interceptor
func debugUnaryClientInterceptor(componentName string, target string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var p peer.Peer
		beg := time.Now()
		err := invoker(ctx, method, req, reply, cc, append(opts, grpc.Peer(&p))...)
		cost := time.Since(beg)

		statusInfo, _ := status.FromError(err)
		if err != nil {
			log.Println("grpc.response", MakeReqAndResError(fileWithLineNum(), componentName, target, cost, method+" | "+fmt.Sprintf("%v", req), statusInfo.String(), ""))
		} else {
			log.Println("grpc.response", MakeReqAndResInfo(fileWithLineNum(), componentName, target, cost, method+" | "+fmt.Sprintf("%v", req), reply, statusInfo.String()))
		}
		return err
	}
}
