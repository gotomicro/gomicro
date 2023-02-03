package k8s

import (
	"context"

	"google.golang.org/grpc/resolver"
	"k8s.io/client-go/kubernetes"
)

type Builder struct {
	cli *kubernetes.Clientset
}

func NewResolveBuilder(cli *kubernetes.Clientset) *Builder {
	return &Builder{
		cli: cli,
	}
}

// URL.PATH: servicename.namespace
func (r *Builder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	etcdResolverObj := newResolver(r.cli)
	err := etcdResolverObj.watch(context.Background(), cc, target.URL.Path)
	return etcdResolverObj, err
}

func (r *Builder) Scheme() string {
	return "k8s"
}
