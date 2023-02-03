package main

import (
	"context"
	"time"

	"go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"gomicro/chapter4/mygrpc"
	"gomicro/chapter4/mygrpc/resolver/etcdv3"
	"gomicro/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
)

func main() {
	clientCon, err := clientv3.New(clientv3.Config{
		Endpoints:        []string{"127.0.0.1:2379"},
		AutoSyncInterval: 0,
		DialTimeout:      mygrpc.Duration("1s"),
	})
	if err != nil {
		mygrpc.DefaultLogger.Panic("创建etcd失败", zap.Error(err))
	}
	resolver.Register(etcdv3.newResolver(clientCon))
	clientGrpc := &mygrpc.ClientComponent{
		BalancerName: "round_robin",
	}
	cc, _ := clientGrpc.NewGRPCClient(context.Background(), "etcd:///micro", grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := helloworld.NewGoMicroClient(cc)
	// 设置请求头信息
	headers := metadata.Pairs("clientName", "microClient")
	ctx := metadata.NewOutgoingContext(context.Background(), headers)
	for {
		forRequest(ctx, client)
		time.Sleep(1 * time.Second)
	}

}

func forRequest(ctx context.Context, client helloworld.GoMicroClient) {
	resp, err := client.SayHello(ctx, &helloworld.HelloReq{
		Msg: "hello",
	})
	if err != nil {
		mygrpc.DefaultLogger.Info("请求错误：" + err.Error())
		return
	}
	mygrpc.DefaultLogger.Info("客户端收到信息：" + resp.GetMsg())

}
