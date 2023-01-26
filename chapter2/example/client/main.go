package main

import (
	"context"

	"gomicro/chapter2/example/helloworld"
	"gomicro/chapter2/mygrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	//cc, err := mygrpc.NewGRPCClient(context.Background(), "127.0.0.1:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	cc, err := mygrpc.NewGRPCClient(context.Background(), "passthrough:///127.0.0.1:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		mygrpc.DefaultLogger.Panic("连接错误: " + err.Error())
	}
	client := helloworld.NewGoMicroClient(cc)
	resp, err := client.SayHello(context.Background(), &helloworld.HelloReq{
		Msg: "123",
	})
	if err != nil {
		mygrpc.DefaultLogger.Info("请求错误：" + err.Error())
		return
	}
	mygrpc.DefaultLogger.Info("客户端收到信息：" + resp.GetMsg())
}
