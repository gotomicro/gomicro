package main

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gomicro/chapter4/mygrpc"
	_ "gomicro/chapter4/mygrpc/balancer/gozerop2c"
	"gomicro/chapter4/mygrpc/resolver/k8s"
	"gomicro/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	clientSet, err := kubernetes.NewForConfig(&rest.Config{
		Host:        "https://192.168.64.227:5443",
		BearerToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6IjY4VVdpUXE1blFOdlNGOTVodGJRbEh2OTRZQ0ZxTVBWZ0pXRnNnc0FLczQifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzA2OTU4NDA5LCJpYXQiOjE2NzU0MjI0MDksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJkZWZhdWx0IiwicG9kIjp7Im5hbWUiOiJtbnMtYmUtNjdjOTk0Zjc4Ny12NnNjcyIsInVpZCI6ImNiNWE0ZmZiLTc0YWUtNDU2ZS1iNDAzLWI3ZDBiMzUzNDQ4YSJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoiZGVmYXVsdCIsInVpZCI6ImVlMmY4MjQzLTA4OWMtNDVlYi05N2Q4LTY3MDYzZWQwMzYyMSJ9LCJ3YXJuYWZ0ZXIiOjE2NzU0MjYwMTZ9LCJuYmYiOjE2NzU0MjI0MDksInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDpkZWZhdWx0OmRlZmF1bHQifQ.Q-44mRc00imgYnnTnJnx86L0ZjbTiP9TFpfJzQFOul97NimXxxjVUK-ei3O_9a3FtimQWQ2qKiDtP9CqYh3VjyFpyeo0oVh7XKRfvdRKZMN_WvVeDItcmohsi9oUBe2Bmewbw8iwMib6vwJid1J7aY640bclQoFDzjRDW1fWEKAGdMGYWiqx3QVvj9lld3nl3-bloq_ppIdSpngo_t0ASFUrKwPn0YZkosEHlZOyBt2UZLXSEf6TOwE7qt3cyJd-sjijn6M29N9ka8j6p6lvpUJgEwiS91y_ufRKCdDp-HfIeqeLOOJrFD1jpJ0LWdIMEEmNPe3mcg-8amvo_dviPA",
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	})
	if err != nil {
		mygrpc.DefaultLogger.Panic("创建k8s失败", zap.Error(err))
	}
	resolver.Register(k8s.NewResolveBuilder(clientSet))
	clientGrpc := &mygrpc.ClientComponent{
		BalancerName: "round_robin",
	}
	cc, _ := clientGrpc.NewGRPCClient(context.Background(), "k8s:///test-p2cserver.default:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := helloworld.NewGoMicroClient(cc)
	// 设置请求头信息
	headers := metadata.Pairs("clientName", "microClient")
	ctx := metadata.NewOutgoingContext(context.Background(), headers)
	for {
		forRequest(ctx, client)
		//break
		time.Sleep(50 * time.Millisecond)
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
	resp2, err2 := client.SayList(ctx, &helloworld.ListReq{
		Msg: "hello",
	})
	if err2 != nil {
		mygrpc.DefaultLogger.Info("请求错误：" + err.Error())
		return
	}
	mygrpc.DefaultLogger.Info("客户端收到信息：" + resp2.GetMsg())
}
