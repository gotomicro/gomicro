package main

import (
	"context"
	"log"

	"gomicro/chapter2/mygrpc"
	"gomicro/config"
	"gomicro/helloworld"
)

func main() {
	app := mygrpc.NewApp()
	helloworld.RegisterGoMicroServer(app, &GoMicro{})
	err := app.Start(config.ServerAddr)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

type GoMicro struct {
	helloworld.UnsafeGoMicroServer
}

// SayHello ...
func (GoMicro) SayHello(ctx context.Context, request *helloworld.HelloReq) (*helloworld.HelloRes, error) {
	log.Println("服务端收到信息：" + request.GetMsg())
	if request.Msg == "panic" {
		panic("i am panic")
	}
	return &helloworld.HelloRes{
		Msg: "我来自服务端",
	}, nil
}
