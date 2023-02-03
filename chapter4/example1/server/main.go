package main

import (
	"context"
	"log"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"gomicro/chapter4/mygrpc"
	"gomicro/config"
	"gomicro/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func main() {
	cc, err := clientv3.New(clientv3.Config{
		Endpoints:        []string{"127.0.0.1:2379"}, // etcd节点ip
		AutoSyncInterval: mygrpc.Duration("60s"),     // 自动同步etcd的member节点
		DialTimeout:      mygrpc.Duration("1s"),      // 拨号超时时间
	})
	if err != nil {
		mygrpc.DefaultLogger.Panic("创建etcd失败", zap.Error(err))
	}

	app := mygrpc.NewApp(
		mygrpc.WithAddress(config.ServerAddr),       // 设置服务Address
		mygrpc.WithRegistry(mygrpc.NewRegistry(cc)), // 设置服务注册中心
		mygrpc.WithServerName("micro"),              // 设置服务名称
	)
	helloworld.RegisterGoMicroServer(app, &GoMicro{})
	err = app.Start(config.ServerAddr)
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
