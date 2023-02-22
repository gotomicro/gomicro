package gozerop2c

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gomicro/chapter4/mygrpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
)

const (
	// Name is the name of p2c balancer.
	Name            = "gozero_p2c"
	decayTime       = int64(time.Second * 10) // default value from finagle
	forcePick       = int64(time.Second)
	initSuccess     = 1000
	throttleSuccess = initSuccess / 2
	penalty         = int64(math.MaxInt32)
	pickTimes       = 3
	logInterval     = time.Minute
)

var initTime = time.Now().AddDate(-1, -1, -1)

var emptyPickResult balancer.PickResult

func init() {
	balancer.Register(newBuilder())
}

type p2cPickerBuilder struct{}

func (b *p2cPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	readySCs := info.ReadySCs
	if len(readySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	var conns []*subConn
	for conn, connInfo := range readySCs {
		conns = append(conns, &subConn{
			addr:    connInfo.Address,
			conn:    conn,
			success: initSuccess,
		})
	}

	return &p2cPicker{
		conns: conns,
		r:     rand.New(rand.NewSource(time.Now().UnixNano())),
		stamp: mygrpc.NewAtomicDuration(),
	}
}

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(Name, new(p2cPickerBuilder), base.Config{HealthCheck: true})
}

type p2cPicker struct {
	conns []*subConn
	r     *rand.Rand
	stamp *mygrpc.AtomicDuration
	lock  sync.Mutex
}

func (p *p2cPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	var chosen *subConn
	switch len(p.conns) {
	case 0:
		return emptyPickResult, balancer.ErrNoSubConnAvailable
	case 1:
		chosen = p.choose(p.conns[0], nil)
	case 2:
		chosen = p.choose(p.conns[0], p.conns[1])
	default:
		var node1, node2 *subConn
		for i := 0; i < pickTimes; i++ {
			a := p.r.Intn(len(p.conns))
			b := p.r.Intn(len(p.conns) - 1)
			if b >= a {
				b++
			}
			node1 = p.conns[a]
			node2 = p.conns[b]
			if node1.healthy() && node2.healthy() {
				break
			}
		}

		chosen = p.choose(node1, node2)
	}

	atomic.AddInt64(&chosen.inflight, 1)
	atomic.AddInt64(&chosen.requests, 1)

	return balancer.PickResult{
		SubConn: chosen.conn,
		Done:    p.buildDoneFunc(chosen),
	}, nil
}

func (p *p2cPicker) buildDoneFunc(c *subConn) func(info balancer.DoneInfo) {
	start := int64(time.Since(initTime))
	return func(info balancer.DoneInfo) {
		// 正在处理的请求数减 1
		atomic.AddInt64(&c.inflight, -1)
		now := time.Since(initTime)
		// 保存本次请求结束时的时间点，并取出上次请求时的时间点
		last := atomic.SwapInt64(&c.last, int64(now))
		td := int64(now) - last
		if td < 0 {
			td = 0
		}

		// 用牛顿冷却定律中的衰减函数模型计算EWMA算法中的β值
		w := math.Exp(float64(-td) / float64(decayTime))
		// 保存本次请求的耗时
		lag := int64(now) - start
		if lag < 0 {
			lag = 0
		}
		olag := atomic.LoadUint64(&c.lag)
		if olag == 0 {
			w = 0
		}
		atomic.StoreUint64(&c.lag, uint64(float64(olag)*w+float64(lag)*(1-w)))
		success := initSuccess
		if info.Err != nil && !Acceptable(info.Err) {
			success = 0
		}
		osucc := atomic.LoadUint64(&c.success)
		atomic.StoreUint64(&c.success, uint64(float64(osucc)*w+float64(success)*(1-w)))

		stamp := p.stamp.Load()
		if now-stamp >= logInterval {
			if p.stamp.CompareAndSwap(stamp, now) {
				p.logStats()
			}
		}
	}
}

func (p *p2cPicker) choose(c1, c2 *subConn) *subConn {
	start := int64(time.Since(initTime))
	if c2 == nil {
		atomic.StoreInt64(&c1.pick, start)
		return c1
	}

	if c1.load() > c2.load() {
		c1, c2 = c2, c1
	}

	pick := atomic.LoadInt64(&c2.pick)
	if start-pick > forcePick && atomic.CompareAndSwapInt64(&c2.pick, pick, start) {
		return c2
	}

	atomic.StoreInt64(&c1.pick, start)
	return c1
}

func (p *p2cPicker) logStats() {
	var stats []string

	p.lock.Lock()
	defer p.lock.Unlock()

	for _, conn := range p.conns {
		stats = append(stats, fmt.Sprintf("conn: %s, load: %d, reqs: %d", conn.addr.Addr, conn.load(), atomic.SwapInt64(&conn.requests, 0)))
	}

	fmt.Sprintf("p2c - %s", strings.Join(stats, "; "))
}

type subConn struct {
	lag      uint64 // 用来保存 ewma 值
	inflight int64  // 用在保存当前节点正在处理的请求总数
	success  uint64 // 用来标识一段时间内此连接的健康状态
	requests int64  // 用来保存请求总数
	last     int64  // 用来保存上一次请求耗时, 用于计算 ewma 值
	pick     int64  // 保存上一次被选中的时间点
	addr     resolver.Address
	conn     balancer.SubConn
}

func (c *subConn) healthy() bool {
	return atomic.LoadUint64(&c.success) > throttleSuccess
}

// load = ewma * inflight;
func (c *subConn) load() int64 {
	// plus one to avoid multiply zero
	lag := int64(math.Sqrt(float64(atomic.LoadUint64(&c.lag) + 1)))
	load := lag * (atomic.LoadInt64(&c.inflight) + 1)
	if load == 0 {
		return penalty
	}

	return load
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
