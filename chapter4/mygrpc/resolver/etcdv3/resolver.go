package etcdv3

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

type etcdResolver struct {
	cli *clientv3.Client
}

//newResolver create a resolver for grpc
func newResolver(cli *clientv3.Client) *etcdResolver {
	return &etcdResolver{
		cli: cli,
	}
}

// ResolveNow ...
func (r *etcdResolver) ResolveNow(rn resolver.ResolveNowOptions) {}

func (r *etcdResolver) Close() {}

func (r *etcdResolver) watch(cc resolver.ClientConn, serviceName string) {
	target := fmt.Sprintf("%s/", serviceName)
	for {
		resolverObj := NewAddressList()
		resp, err := r.cli.Get(context.Background(), target, clientv3.WithPrefix())
		if err != nil {
			time.Sleep(time.Second * 5)
			continue
		}
		// 初始化
		for _, value := range resp.Kvs {
			var resolverInfo resolver.Address
			if err := json.Unmarshal(value.Value, &resolverInfo); err != nil {
				continue
			}
			resolverObj.Put(resolverInfo)
		}

		cc.UpdateState(resolver.State{
			Addresses: resolverObj.GetAddressList(),
		})

		// watch
		ctx, cancel := context.WithCancel(context.Background())
		rch := r.cli.Watch(ctx, target, clientv3.WithPrefix(), clientv3.WithCreatedNotify())
		for n := range rch {
			for _, ev := range n.Events {
				switch ev.Type {
				// 添加或者更新
				case mvccpb.PUT:
					var resolverInfo resolver.Address
					if err := json.Unmarshal(ev.Kv.Value, &resolverInfo); err == nil {
						resolverObj.Put(resolverInfo)
					}

				// 硬删除
				case mvccpb.DELETE:
					var resolverInfo resolver.Address
					if err := json.Unmarshal(ev.Kv.Value, &resolverInfo); err == nil {
						resolverObj.Delete(resolverInfo)
					}
				}
			}
			cc.UpdateState(resolver.State{
				Addresses: resolverObj.GetAddressList(),
			})
		}
		cancel()
	}
}

type AddressList struct {
	serverName string
	store      map[string]resolver.Address
	m          sync.RWMutex
}

func NewAddressList() *AddressList {
	return &AddressList{
		store: make(map[string]resolver.Address),
	}
}

func (a *AddressList) Put(address resolver.Address) {
	a.m.Lock()
	defer a.m.Unlock()
	a.store[address.Addr] = address
}

func (a *AddressList) Delete(address resolver.Address) {
	a.m.Lock()
	defer a.m.Unlock()
	delete(a.store, address.Addr)
}

func (a *AddressList) GetAddressList() []resolver.Address {
	addrs := make([]resolver.Address, 0)
	a.m.RLock()
	defer a.m.RUnlock()
	for _, address := range a.store {
		addrs = append(addrs, address)
	}
	return addrs
}
