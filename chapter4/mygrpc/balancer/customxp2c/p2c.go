package customxp2c

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"gomicro/chapter4/mygrpc/balancer/customxp2c/ewma"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

const (
	forcePick = time.Second * 3
	// Name is balancer name
	Name = "customx_p2c"
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

	ewmaNodes := make([]*ewma.Node, 0, len(info.ReadySCs))
	for conn, _ := range info.ReadySCs {
		ewmaNodes = append(ewmaNodes, ewma.Build(conn))
	}
	p.ewmaNodes = ewmaNodes
	return p

}

// balancerPicker is a grpc picker.
type balancerPicker struct {
	nodes     atomic.Value
	lock      sync.Mutex
	r         *rand.Rand
	picked    int64
	ewmaNodes []*ewma.Node
}

// Pick instances.
func (p *balancerPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.ewmaNodes) == 0 {
		return balancer.PickResult{}, ErrNoAvailable
	}

	if len(p.ewmaNodes) == 1 {
		calc := p.ewmaNodes[0].Calc()
		return balancer.PickResult{
			SubConn: p.ewmaNodes[0].GetSubConn(),
			Done: func(di balancer.DoneInfo) {
				calc(info.Ctx, di)
			},
		}, nil
	}

	var lowLoadNode, highLoadNode *ewma.Node
	var chooseNode *ewma.Node
	nodeA, nodeB := p.prePick(p.ewmaNodes)
	// meta.Load is the weight set by the service publisher in discovery
	if nodeB.Load() > nodeA.Load() {
		lowLoadNode, highLoadNode = nodeB, nodeA
	} else {
		lowLoadNode, highLoadNode = nodeA, nodeB
	}
	chooseNode = lowLoadNode
	// If the failed node has never been selected once during forceGap, it is forced to be selected once
	// Take advantage of forced opportunities to trigger updates of success rate and delay
	if highLoadNode.PickElapsed() > forcePick && atomic.CompareAndSwapInt64(&p.picked, 0, 1) {
		chooseNode = highLoadNode
		atomic.StoreInt64(&p.picked, 0)
	}
	calc := chooseNode.Calc()
	return balancer.PickResult{
		SubConn: chooseNode.GetSubConn(),
		Done: func(di balancer.DoneInfo) {
			calc(info.Ctx, di)
		},
	}, nil
}

// choose two distinct nodes.
func (p *balancerPicker) prePick(nodes []*ewma.Node) (nodeA *ewma.Node, nodeB *ewma.Node) {
	p.lock.Lock()
	a := p.r.Intn(len(nodes))
	b := p.r.Intn(len(nodes) - 1)
	p.lock.Unlock()
	if b >= a {
		b++
	}
	nodeA, nodeB = nodes[a], nodes[b]
	return
}

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(Name, new(p2cPickerBuilder), base.Config{HealthCheck: true})
}
