package mygrpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"runtime"
	"time"

	"github.com/gotomicro/ego/core/util/xstring"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func defaultUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	componentName := "grpcServer"
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (res any, err error) {
		var beg = time.Now()
		// 此处必须使用defer来recover handler内部可能出现的panic
		defer func() {
			stack := make([]byte, 4096)
			cost := time.Since(beg)
			if rec := recover(); rec != nil {
				switch recType := rec.(type) {
				case error:
					err = recType
				default:
					err = fmt.Errorf("%v", rec)
				}

				stack = stack[:runtime.Stack(stack, true)]
			}

			var reqMap = map[string]interface{}{
				"payload": xstring.JSON(req),
			}
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				reqMap["metadata"] = md
			}

			var resMap = map[string]interface{}{
				"payload": xstring.JSON(res),
			}
			statusInfo, _ := status.FromError(err)
			if err != nil {
				log.Println("grpc.request", MakeReqAndResError(fileWithLineNum(), componentName, getPeerAddr(ctx), cost, info.FullMethod, fmt.Sprintf("%v", reqMap), statusInfo.String(), string(stack)))
			} else {
				log.Println("grpc.request", MakeReqAndResInfo(fileWithLineNum(), componentName, getPeerAddr(ctx), cost, info.FullMethod, fmt.Sprintf("%v", reqMap), resMap, statusInfo.String()))
			}
		}()
		return handler(ctx, req)
	}
}

// getPeerAddr 获取对端ip
func getPeerAddr(ctx context.Context) string {
	// 从grpc里取对端ip
	pr, ok2 := peer.FromContext(ctx)
	if !ok2 {
		return ""
	}
	if pr.Addr == net.Addr(nil) {
		return ""
	}
	return pr.Addr.String()
}
