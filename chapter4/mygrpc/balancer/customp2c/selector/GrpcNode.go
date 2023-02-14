package selector

import (
	"google.golang.org/grpc/balancer"
)

type GrpcNode struct {
	SubConn balancer.SubConn
}
