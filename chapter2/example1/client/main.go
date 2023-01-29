package main

import (
	"context"

	"gomicro/chapter2/mygrpc"
	"gomicro/config"
	"gomicro/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	clientGrpc := &mygrpc.ClientComponent{}
	cc, _ := clientGrpc.NewGRPCClient(context.Background(), "passthrough:///"+config.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := helloworld.NewGoMicroClient(cc)
	// 设置请求头信息
	headers := metadata.Pairs("clientName", "microClient")
	ctx := metadata.NewOutgoingContext(context.Background(), headers)
	resp, err := client.SayHello(ctx, &helloworld.HelloReq{
		Msg: "触发一个错误",
	})
	if err != nil {
		mygrpc.DefaultLogger.Info("请求错误：" + err.Error())
		return
	}
	mygrpc.DefaultLogger.Info("客户端收到信息：" + resp.GetMsg())
}
