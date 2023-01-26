package main

import (
	"context"
	"log"

	"gomicro/chapter1/example/helloworld"
	"gomicro/chapter1/mygrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cc, err := mygrpc.NewGRPCClient(context.Background(), "127.0.0.1:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Panicln("连接错误: " + err.Error())
	}
	client := helloworld.NewGoMicroClient(cc)
	resp, err := client.SayHello(context.Background(), &helloworld.HelloReq{
		Name: "我来自客户端",
	})
	if err != nil {
		log.Println("请求错误：" + err.Error())
		return
	}
	log.Println("客户端收到信息：" + resp.GetMessage())
}
