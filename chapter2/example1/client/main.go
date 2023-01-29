package main

import (
	"context"

	"go.uber.org/zap"
	"gomicro/chapter2/mygrpc"
	"gomicro/config"
	"gomicro/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	mygrpc.DefaultLogger.Info("客户端收到信息", zap.Any("1", 1), zap.Any("2", 1))

	cc, err := mygrpc.NewGRPCClient(context.Background(), "passthrough:///"+config.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		mygrpc.DefaultLogger.Panic("连接错误: " + err.Error())
	}
	client := helloworld.NewGoMicroClient(cc)
	// 设置请求头信息
	headers := metadata.New(nil)
	headers.Set("clientName", "microClient")
	ctx := metadata.NewOutgoingContext(context.Background(), headers)
	resp, err := client.SayHello(ctx, &helloworld.HelloReq{
		Msg: "123",
	})
	if err != nil {
		mygrpc.DefaultLogger.Info("请求错误：" + err.Error())
		return
	}
	mygrpc.DefaultLogger.Info("客户端收到信息：" + resp.GetMsg())
}
