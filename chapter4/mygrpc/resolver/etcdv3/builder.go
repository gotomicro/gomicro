package etcdv3

import (
	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

type Builder struct {
	cli *clientv3.Client
}

func NewResolveBuilder(cli *clientv3.Client) *Builder {
	return &Builder{
		cli: cli,
	}
}

func (r *Builder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	etcdResolverObj := newResolver(r.cli)
	etcdResolverObj.watch(cc, target.URL.Path)
	return etcdResolverObj, nil
}

func (r *Builder) Scheme() string {
	return "etcd"
}
