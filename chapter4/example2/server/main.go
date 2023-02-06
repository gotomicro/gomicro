package main

import (
	"context"
	"log"

	"gomicro/chapter4/mygrpc"
	"gomicro/config"
	"gomicro/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func main() {
	app := mygrpc.NewApp(
		mygrpc.WithAddress(config.K8sServerAddr), // 设置服务Address
		mygrpc.WithServerName("micro"),           // 设置服务名称
	)
	helloworld.RegisterGoMicroServer(app, &GoMicro{})
	err := app.Start()
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
	var j []int
	var k int
	for i := 0; i < 100; i++ {
		k++
		j = append(j, k)
	}
	return &helloworld.HelloRes{
		Msg: "我来自服务端",
	}, nil
}

// SayList ...
func (GoMicro) SayList(ctx context.Context, request *helloworld.ListReq) (*helloworld.ListRes, error) {
	log.Println("服务端收到信息：" + request.GetMsg())
	headers := metadata.Pairs("serverName", "microServer")
	grpc.SendHeader(ctx, headers)
	tailers := metadata.Pairs("tailName", "microServerTail")
	grpc.SetTrailer(ctx, tailers)
	if request.Msg == "触发一个错误" {
		return nil, status.New(codes.Internal, "系统错误").Err()
	}
	var j []int
	var k int
	for i := 0; i < 100000; i++ {
		k++
		j = append(j, k)
	}
	return &helloworld.ListRes{
		Msg: "我来自服务端List",
	}, nil
}
