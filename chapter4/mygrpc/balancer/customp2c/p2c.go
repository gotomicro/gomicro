package kratosp2c

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"gomicro/chapter4/mygrpc/balancer/customp2c/ewma"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

const (
	forcePick = time.Second * 3
	// Name is balancer name
	Name = "custom_p2c"
)

// ErrNoAvailable is no available node.
var ErrNoAvailable = fmt.Errorf("no_available_node")

func init() {
	balancer.Register(newBuilder())
}

type p2cPickerBuilder struct{}

func (b *p2cPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	readySCs := info.ReadySCs
	if len(readySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	p := &balancerPicker{
		r: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	weightedNodes := make([]*ewma.Node, 0, len(info.ReadySCs))
	for conn, _ := range info.ReadySCs {
		weightedNodes = append(weightedNodes, ewma.Build(conn))
	}
	p.weightedNodes = weightedNodes
	return p

}

// balancerPicker is a grpc picker.
type balancerPicker struct {
	nodes         atomic.Value
	mu            sync.Mutex
	r             *rand.Rand
	picked        int64
	weightedNodes []*ewma.Node
}

// Pick instances.
func (p *balancerPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	var (
		done func(ctx context.Context, di balancer.DoneInfo)
	)
	if len(p.weightedNodes) == 0 {
		return balancer.PickResult{}, ErrNoAvailable
	}

	if len(p.weightedNodes) == 1 {
		done = p.weightedNodes[0].Pick()
		return balancer.PickResult{
			SubConn: p.weightedNodes[0].GetSubConn(),
			Done: func(di balancer.DoneInfo) {
				done(info.Ctx, di)
			},
		}, nil
	}

	var pc, upc *ewma.Node
	nodeA, nodeB := p.prePick(p.weightedNodes)
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

	return balancer.PickResult{
		SubConn: pc.GetSubConn(),
		Done: func(di balancer.DoneInfo) {
			done(info.Ctx, di)
		},
	}, nil
}

// choose two distinct nodes.
func (p *balancerPicker) prePick(nodes []*ewma.Node) (nodeA *ewma.Node, nodeB *ewma.Node) {
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
