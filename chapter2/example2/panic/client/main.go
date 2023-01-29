package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"gomicro/chapter2/mygrpc"
	"gomicro/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	router := gin.Default()
	clientGrpc := &mygrpc.ClientComponent{
		IsFailFast: true,
	}
	cc, _ := clientGrpc.NewGRPCClient(context.Background(), "passthrough:///127.0.0.1:9100", grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := helloworld.NewGoMicroClient(cc)
	router.GET("/grpc", func(c *gin.Context) {
		resp, err := client.SayHello(context.Background(), &helloworld.HelloReq{
			Msg: "我来自客户端",
		})
		if err != nil {
			mygrpc.DefaultLogger.Info("请求错误：" + err.Error())
			return
		}
		mygrpc.DefaultLogger.Info("客户端收到信息：" + resp.GetMsg())
		c.String(200, "%s", resp.GetMsg())
	})
	err := router.Run(":9002")
	if err != nil {
		mygrpc.DefaultLogger.Panic("启动错误: " + err.Error())
	}
}
