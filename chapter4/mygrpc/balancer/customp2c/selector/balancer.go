package selector

import (
	"context"
	"fmt"
	"time"
)

// Balancer is balancer interface
type Balancer interface {
	Pick(ctx context.Context, nodes []WeightedNode) (selected WeightedNode, done DoneFunc, err error)
}

// BalancerBuilder build balancer
type BalancerBuilder interface {
	Build() Balancer
}

// WeightedNode calculates scheduling weight in real time
type WeightedNode interface {
	//Node

	// Raw returns the original node
	Raw() *GrpcNode

	// Weight is the runtime calculated weight
	Weight() float64

	// Pick the node
	Pick() DoneFunc

	// PickElapsed is time elapsed since the latest pick
	PickElapsed() time.Duration
}

// WeightedNodeBuilder is WeightedNode Builder
type WeightedNodeBuilder interface {
	Build(*GrpcNode) WeightedNode
}

// DoneFunc is callback function when RPC invoke done.
type DoneFunc func(ctx context.Context, di DoneInfo)

// DoneInfo is callback info when RPC invoke done.
type DoneInfo struct {
	// Response Error
	Err error
	// Response Metadata
	ReplyMD ReplyMD

	// BytesSent indicates if any bytes have been sent to the server.
	BytesSent bool
	// BytesReceived indicates if any byte has been received from the server.
	BytesReceived bool
}

// ReplyMD is Reply Metadata.
type ReplyMD interface {
	Get(key string) string
}

// ErrNoAvailable is no available node.
var ErrNoAvailable = fmt.Errorf("no_available_node")
