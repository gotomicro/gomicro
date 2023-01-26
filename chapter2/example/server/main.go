package main

import (
	"context"
	"log"

	"gomicro/chapter2/example/helloworld"
	"gomicro/chapter2/mygrpc"
)

func main() {
	app := mygrpc.NewApp()
	helloworld.RegisterGoMicroServer(app, &GoMicro{})
	err := app.Start("127.0.0.1:9001")
	if err != nil {
		log.Fatalln(err.Error())
	}
}

type GoMicro struct {
	helloworld.UnsafeGoMicroServer
}

// SayHello ...
func (GoMicro) SayHello(ctx context.Context, request *helloworld.HelloReq) (*helloworld.HelloRes, error) {
	log.Println("服务端收到信息：" + request.GetName())
	return &helloworld.HelloRes{
		Message: "我来自服务端",
	}, nil
}
