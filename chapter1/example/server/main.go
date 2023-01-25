package main

import (
	"context"
	"log"

	"gomicro/chapter1/example/helloworld"
	"gomicro/chapter1/mygrpc"
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
	return &helloworld.HelloRes{
		Message: "Hello Go Micro Service",
	}, nil
}
