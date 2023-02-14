package kratosp2c

import (
	"math/rand"
	"sync/atomic"
	"time"

	"gomicro/chapter4/mygrpc/balancer/kratosp2c/ewma"
	"gomicro/chapter4/mygrpc/balancer/kratosp2c/selector"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/metadata"
)

const (
	forcePick = time.Second * 3
	// Name is balancer name
	Name = "kratos_p2c"
)

func init() {
	balancer.Register(newBuilder())
}

type grpcNode struct {
	selector.Node
	subConn balancer.SubConn
}

type p2cPickerBuilder struct{}

func (b *p2cPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	readySCs := info.ReadySCs
	if len(readySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	nodes := make([]selector.Node, 0, len(info.ReadySCs))
	for conn, info := range info.ReadySCs {
		ins, _ := info.Address.Attributes.Value("rawServiceInstance").(*selector.ServiceInstance)
		nodes = append(nodes, &grpcNode{
			Node:    selector.NewNode("grpc", info.Address.Addr, ins),
			subConn: conn,
		})
	}

	p := &balancerPicker{
		Balancer:    &Balancer{r: rand.New(rand.NewSource(time.Now().UnixNano()))},
		NodeBuilder: &ewma.Builder{},
	}
	p.Apply(nodes)
	return p

}

// balancerPicker is a grpc picker.
type balancerPicker struct {
	nodes       atomic.Value
	Balancer    selector.Balancer
	NodeBuilder selector.WeightedNodeBuilder
}

func (p *balancerPicker) Apply(nodes []selector.Node) {
	weightedNodes := make([]selector.WeightedNode, 0, len(nodes))
	for _, n := range nodes {
		weightedNodes = append(weightedNodes, p.NodeBuilder.Build(n))
	}
	// TODO: Do not delete unchanged nodes
	p.nodes.Store(weightedNodes)
}

// Pick pick instances.
func (p *balancerPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	var (
		selected   selector.Node
		done       selector.DoneFunc
		err        error
		candidates []selector.WeightedNode
	)

	nodes, ok := p.nodes.Load().([]selector.WeightedNode)
	if !ok {
		return balancer.PickResult{}, selector.ErrNoAvailable
	}
	candidates = nodes

	if len(candidates) == 0 {
		return balancer.PickResult{}, selector.ErrNoAvailable
	}

	wn, done, err := p.Balancer.Pick(info.Ctx, candidates)
	if err != nil {
		return balancer.PickResult{}, err
	}
	selected = wn.Raw()
	return balancer.PickResult{
		SubConn: selected.(*grpcNode).subConn,
		Done: func(di balancer.DoneInfo) {
			done(info.Ctx, selector.DoneInfo{
				Err:           di.Err,
				BytesSent:     di.BytesSent,
				BytesReceived: di.BytesReceived,
				ReplyMD:       Trailer(di.Trailer),
			})
		},
	}, nil
}

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(Name, new(p2cPickerBuilder), base.Config{HealthCheck: true})
}

// Trailer is a grpc trailer MD.
type Trailer metadata.MD

// Get get a grpc trailer value.
func (t Trailer) Get(k string) string {
	v := metadata.MD(t).Get(k)
	if len(v) > 0 {
		return v[0]
	}
	return ""
}
