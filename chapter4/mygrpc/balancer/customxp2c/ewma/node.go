package ewma

import (
	"context"
	"math"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
)

var (
	initSuccess uint64 = 1000
	decayTime          = int64(time.Second * 10) // default value from finagle
	penalty            = int64(math.MaxInt32)
)

type Node struct {
	ewma     uint64 // 用来保存 ewma 值
	inflight int64  // 用在保存当前节点正在处理的请求总数
	success  uint64 // 用来标识一段时间内此连接的健康状态
	requests int64  // 用来保存请求总数
	stamp    int64
	last     int64 // 用来保存上一次请求耗时, 用于计算 ewma 值
	pick     int64 // 保存上一次被选中的时间点
	addr     resolver.Address
	conn     balancer.SubConn
}

// Build create a weighted node.
func Build(n balancer.SubConn) *Node {
	s := &Node{
		conn:     n,
		ewma:     0,
		success:  initSuccess,
		inflight: 1,
	}
	return s
}

func (n *Node) GetSubConn() balancer.SubConn {
	return n.conn
}

// Load is node effective weight.
func (n *Node) Load() (weight float64) {
	weight = float64(n.health()*uint64(time.Second)) / float64(n.load())
	return
}

func (n *Node) PickElapsed() time.Duration {
	return time.Duration(time.Now().UnixNano() - atomic.LoadInt64(&n.last))
}

func (n *Node) health() uint64 {
	return atomic.LoadUint64(&n.success)
}

// load = ewma * inflight;
func (c *Node) load() int64 {
	// plus one to avoid multiply zero
	lag := int64(math.Sqrt(float64(atomic.LoadUint64(&c.ewma) + 1)))
	load := lag * (atomic.LoadInt64(&c.inflight) + 1)
	if load == 0 {
		return penalty
	}

	return load
}

// Calc 计算节点性能
func (n *Node) Calc() func(ctx context.Context, di balancer.DoneInfo) {
	//start := int64(time.Since(initTime))
	start := time.Now().UnixNano()
	// 保存本次请求的时间点，并取出上次请求时的时间点
	atomic.StoreInt64(&n.last, start)
	return func(ctx context.Context, di balancer.DoneInfo) {
		// 正在处理的请求数减 1
		atomic.AddInt64(&n.inflight, -1)
		now := time.Now().UnixNano()
		// get moving average ratio w
		stamp := atomic.SwapInt64(&n.stamp, now)
		td := start - stamp
		if td < 0 {
			td = 0
		}

		// 用牛顿冷却定律中的衰减函数模型计算EWMA算法中的β值
		belta := math.Exp(float64(-td) / float64(decayTime))
		// 保存本次请求的耗时
		lag := now - start
		if lag < 0 {
			lag = 0
		}
		oldEwma := atomic.LoadUint64(&n.ewma)
		if oldEwma == 0 {
			belta = 0
		}
		atomic.StoreUint64(&n.ewma, uint64(float64(oldEwma)*belta+float64(lag)*(1-belta)))
		success := initSuccess
		if di.Err != nil && !Acceptable(di.Err) {
			success = 0
		}
		oldSuccess := atomic.LoadUint64(&n.success)
		atomic.StoreUint64(&n.success, uint64(float64(oldSuccess)*belta+float64(success)*(1-belta)))

		//stamp := p.stamp.Load()
		//if now-stamp >= logInterval {
		//	if p.stamp.CompareAndSwap(stamp, now) {
		//		p.logStats()
		//	}
		//}
	}
}

// Acceptable checks if given error is acceptable.
func Acceptable(err error) bool {
	switch status.Code(err) {
	case codes.DeadlineExceeded, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unimplemented:
		return false
	default:
		return true
	}
}
