package ewma

import (
	"container/list"
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// The mean lifetime of `cost`, it reaches its half-life after Tau*ln(2).
	tau = int64(time.Millisecond * 600)
	// if statistic not collected,we add a big lag penalty to endpoint
	penalty = uint64(time.Second * 10)
)

// Node is endpoint instance
type Node struct {
	subConn   balancer.SubConn
	lag       int64
	success   uint64
	inflight  int64
	inflights *list.List
	stamp     int64
	predictTs int64
	predict   int64
	// request number in a period time
	reqs int64
	// last lastPick timestamp
	lastPick int64

	//errHandler func(err error) (isErr bool)
	lk sync.RWMutex
}

// Build create a weighted node.
func Build(n balancer.SubConn) *Node {
	s := &Node{
		subConn:   n,
		lag:       0,
		success:   1000,
		inflight:  1,
		inflights: list.New(),
	}
	return s
}

func (n *Node) health() uint64 {
	return atomic.LoadUint64(&n.success)
}

func (n *Node) load() (load uint64) {
	now := time.Now().UnixNano()
	avgLag := atomic.LoadInt64(&n.lag)
	lastPredictTs := atomic.LoadInt64(&n.predictTs)
	predictInterval := avgLag / 5
	if predictInterval < int64(time.Millisecond*5) {
		predictInterval = int64(time.Millisecond * 5)
	}
	if predictInterval > int64(time.Millisecond*200) {
		predictInterval = int64(time.Millisecond * 200)
	}
	if now-lastPredictTs > predictInterval && atomic.CompareAndSwapInt64(&n.predictTs, lastPredictTs, now) {
		var (
			total   int64
			count   int
			predict int64
		)
		n.lk.RLock()
		first := n.inflights.Front()
		for first != nil {
			lag := now - first.Value.(int64)
			if lag > avgLag {
				count++
				total += lag
			}
			first = first.Next()
		}
		if count > (n.inflights.Len()/2 + 1) {
			predict = total / int64(count)
		}
		n.lk.RUnlock()
		atomic.StoreInt64(&n.predict, predict)
	}

	if avgLag == 0 {
		// penalty is the penalty value when there is no data when the node is just started.
		// The default value is 1e9 * 10
		load = penalty * uint64(atomic.LoadInt64(&n.inflight))
		return
	}
	predict := atomic.LoadInt64(&n.predict)
	if predict > avgLag {
		avgLag = predict
	}
	load = uint64(avgLag) * uint64(atomic.LoadInt64(&n.inflight))
	return
}

// Pick pick a node.
func (n *Node) Pick() func(ctx context.Context, di balancer.DoneInfo) {
	now := time.Now().UnixNano()
	atomic.StoreInt64(&n.lastPick, now)
	atomic.AddInt64(&n.inflight, 1)
	atomic.AddInt64(&n.reqs, 1)
	n.lk.Lock()
	e := n.inflights.PushBack(now)
	n.lk.Unlock()
	return func(ctx context.Context, di balancer.DoneInfo) {
		n.lk.Lock()
		n.inflights.Remove(e)
		n.lk.Unlock()
		atomic.AddInt64(&n.inflight, -1)

		now := time.Now().UnixNano()
		// get moving average ratio w
		stamp := atomic.SwapInt64(&n.stamp, now)
		td := now - stamp
		if td < 0 {
			td = 0
		}
		w := math.Exp(float64(-td) / float64(tau))

		start := e.Value.(int64)
		lag := now - start
		if lag < 0 {
			lag = 0
		}
		oldLag := atomic.LoadInt64(&n.lag)
		if oldLag == 0 {
			w = 0.0
		}
		lag = int64(float64(oldLag)*w + float64(lag)*(1.0-w))
		atomic.StoreInt64(&n.lag, lag)

		success := uint64(1000) // error value ,if error set 1
		if di.Err != nil {
			//if n.errHandler != nil && n.errHandler(di.Err) {
			//	success = 0
			//}
			//var netErr net.Error
			//if errors.Is(context.DeadlineExceeded, di.Err) || errors.Is(context.Canceled, di.Err) ||
			//	errors.IsServiceUnavailable(di.Err) || errors.IsGatewayTimeout(di.Err) || errors.As(di.Err, &netErr) {
			//	success = 0
			//}
			if !Acceptable(di.Err) {
				success = 0
			}
		}
		oldSuc := atomic.LoadUint64(&n.success)
		success = uint64(float64(oldSuc)*w + float64(success)*(1.0-w))
		atomic.StoreUint64(&n.success, success)
	}
}

// Weight is node effective weight.
func (n *Node) Weight() (weight float64) {
	weight = float64(n.health()*uint64(time.Second)) / float64(n.load())
	return
}

func (n *Node) PickElapsed() time.Duration {
	return time.Duration(time.Now().UnixNano() - atomic.LoadInt64(&n.lastPick))
}

func (n *Node) GetSubConn() balancer.SubConn {
	return n.subConn
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
