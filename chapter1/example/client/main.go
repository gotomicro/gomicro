package main

import (
	"context"
	"log"

	"gomicro/chapter1/example/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	cc, err := grpc.DialContext(context.Background(), "127.0.0.1:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Panicln("连接错误: " + err.Error())
	}
	client := helloworld.NewGoMicroClient(cc)
	resp, err := client.SayHello(context.Background(), &helloworld.HelloReq{
		Name: "i am client",
	})
	if err != nil {
		log.Println("请求错误：" + err.Error())
		return
	}
	log.Println(resp)

}
