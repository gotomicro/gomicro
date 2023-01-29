package main

import (
	"context"
	"log"

	"gomicro/chapter2/mygrpc"
	"gomicro/config"
	"gomicro/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
	helloworld.UnimplementedGoMicroServer
}

// SayHello ...
func (GoMicro) SayHello(ctx context.Context, request *helloworld.HelloReq) (*helloworld.HelloRes, error) {
	log.Println("服务端收到信息：" + request.GetMsg())
	headers := metadata.Pairs("serverName", "microServer")
	grpc.SendHeader(ctx, headers)
	tailers := metadata.Pairs("tailName", "microServerTail")
	grpc.SetTrailer(ctx, tailers)
	if request.Msg == "触发一个错误" {
		return nil, status.New(codes.Internal, "系统错误").Err()
	}
	return &helloworld.HelloRes{
		Msg: "我来自服务端",
	}, nil
}
