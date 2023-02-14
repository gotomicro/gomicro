package kratosp2c

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"gomicro/chapter4/mygrpc/balancer/customp2c/ewma"
	"gomicro/chapter4/mygrpc/balancer/customp2c/selector"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/metadata"
)

const (
	forcePick = time.Second * 3
	// Name is balancer name
	Name = "custom_p2c"
)

func init() {
	balancer.Register(newBuilder())
}

type p2cPickerBuilder struct{}

func (b *p2cPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	readySCs := info.ReadySCs
	if len(readySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	nodes := make([]*selector.GrpcNode, 0, len(info.ReadySCs))
	for conn, _ := range info.ReadySCs {
		nodes = append(nodes, &selector.GrpcNode{
			SubConn: conn,
		})
	}

	p := &balancerPicker{
		r:           rand.New(rand.NewSource(time.Now().UnixNano())),
		NodeBuilder: &ewma.Builder{},
	}
	p.Apply(nodes)
	return p

}

// balancerPicker is a grpc picker.
type balancerPicker struct {
	nodes       atomic.Value
	NodeBuilder *ewma.Builder
	mu          sync.Mutex
	r           *rand.Rand
	picked      int64
}

func (p *balancerPicker) Apply(nodes []*selector.GrpcNode) {
	weightedNodes := make([]selector.WeightedNode, 0, len(nodes))
	for _, n := range nodes {
		weightedNodes = append(weightedNodes, p.NodeBuilder.Build(n))
	}
	// TODO: Do not delete unchanged nodes
	p.nodes.Store(weightedNodes)
}

// Pick instances.
func (p *balancerPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	var (
		selected   *selector.GrpcNode
		done       selector.DoneFunc
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

	if len(candidates) == 1 {
		done = nodes[0].Pick()
		selected = nodes[0].Raw()
		return balancer.PickResult{
			SubConn: selected.SubConn,
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

	var pc, upc selector.WeightedNode
	nodeA, nodeB := p.prePick(nodes)
	// meta.Weight is the weight set by the service publisher in discovery
	if nodeB.Weight() > nodeA.Weight() {
		pc, upc = nodeB, nodeA
	} else {
		pc, upc = nodeA, nodeB
	}

	// If the failed node has never been selected once during forceGap, it is forced to be selected once
	// Take advantage of forced opportunities to trigger updates of success rate and delay
	if upc.PickElapsed() > forcePick && atomic.CompareAndSwapInt64(&p.picked, 0, 1) {
		pc = upc
		atomic.StoreInt64(&p.picked, 0)
	}
	done = pc.Pick()
	selected = pc.Raw()

	return balancer.PickResult{
		SubConn: selected.SubConn,
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

// choose two distinct nodes.
func (p *balancerPicker) prePick(nodes []selector.WeightedNode) (nodeA selector.WeightedNode, nodeB selector.WeightedNode) {
	p.mu.Lock()
	a := p.r.Intn(len(nodes))
	b := p.r.Intn(len(nodes) - 1)
	p.mu.Unlock()
	if b >= a {
		b = b + 1
	}
	nodeA, nodeB = nodes[a], nodes[b]
	return
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
