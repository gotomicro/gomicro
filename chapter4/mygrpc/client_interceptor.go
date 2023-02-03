package mygrpc

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gotomicro/ego/core/util/xstring"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// debugUnaryClientInterceptor returns grpc unary request request and response details interceptor
func debugUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	componentName := "grpcClient"
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		beg := time.Now()
		// 获取对端信息
		var p peer.Peer
		// 响应的头信息
		var resHeader metadata.MD
		// 响应的尾信息
		var resTrailer metadata.MD
		// 请求的头信息
		reqHeader, _ := metadata.FromOutgoingContext(ctx)
		opts = append(opts, grpc.Header(&resHeader))
		opts = append(opts, grpc.Trailer(&resTrailer))
		opts = append(opts, grpc.Peer(&p))
		err := invoker(ctx, method, req, reply, cc, opts...)
		// 将err信息转换为grpc的status信息
		statusInfo, _ := status.FromError(err)
		// 请求
		var reqMap = map[string]any{
			"payload":  xstring.JSON(req),
			"metadata": reqHeader,
		}
		var resMap = map[string]any{
			"payload": xstring.JSON(reply),
			"metadata": map[string]any{
				"header":  resHeader,
				"trailer": resTrailer,
			},
		}
		// 记录此次调用grpc的耗时
		cost := time.Since(beg)
		var addr string
		if p.Addr != nil {
			addr = p.Addr.String()
		} else {
			addr = cc.Target()
		}
		if err != nil {
			log.Println("grpc.response", MakeReqAndResError(fileWithLineNum(), componentName, addr, cost, method, fmt.Sprintf("%v", reqMap), fmt.Sprintf("%v", resMap), statusInfo.String(), ""))
		} else {
			log.Println("grpc.response", MakeReqAndResInfo(fileWithLineNum(), componentName, addr, cost, method, fmt.Sprintf("%v", reqMap), resMap, statusInfo.String()))
		}
		return err
	}
}
