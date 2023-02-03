package k8s

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gomicro/chapter4/mygrpc"
	"google.golang.org/grpc/resolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const defaultRsync = 5 * time.Minute

type k8sResolver struct {
	cli    *kubernetes.Clientset
	queue  workqueue.Interface
	logger *zap.Logger
}

//newResolver create a resolver for grpc
func newResolver(cli *kubernetes.Clientset) *k8sResolver {
	return &k8sResolver{
		cli:    cli,
		queue:  workqueue.New(),
		logger: mygrpc.DefaultLogger,
	}
}

// ResolveNow ...
func (r *k8sResolver) ResolveNow(rn resolver.ResolveNowOptions) {}

func (r *k8sResolver) Close() {}

func (r *k8sResolver) watch(ctx context.Context, cc resolver.ClientConn, urlPath string) error {
	svcName, namespaceName, port, err := getServiceNameAndNamespaceNameAndPort(urlPath)
	if err != nil {
		return fmt.Errorf("k8s resolver fail, err: %w", err)
	}
	resolverObj := NewAddressList()
	endPoints, err := r.cli.CoreV1().Endpoints(namespaceName).Get(context.Background(), svcName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("svcName--------------->"+"%+v\n", svcName)
	fmt.Printf("endPoints--------------->"+"%+v\n", endPoints)
	for _, subsets := range endPoints.Subsets {
		for _, address := range subsets.Addresses {
			resolverObj.Put(resolver.Address{
				Addr: address.IP + ":" + port,
			})
		}
	}
	cc.UpdateState(resolver.State{
		Addresses: resolverObj.GetAddressList(),
	})

	informersFactory := informers.NewSharedInformerFactoryWithOptions(
		r.cli,
		defaultRsync,
		informers.WithNamespace(namespaceName),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = "metadata.name=" + endPoints.Name
			options.ResourceVersion = "0"
		}),
	)

	informer := informersFactory.Core().V1().Endpoints()
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    r.addEndpoints,
		UpdateFunc: r.updateEndpoints,
		DeleteFunc: r.deleteEndpoints,
	})
	// 启动该命名空间里监听
	go informersFactory.Start(ctx.Done())
	go func() {
		for r.ProcessWorkItem(func(info *KubernetesEvent) error {
			switch info.EventType {
			case watch.Added:
				for _, ip := range info.IPs {
					resolverObj.Put(resolver.Address{
						Addr: ip + ":" + port,
					})
				}
			case watch.Deleted:
				for _, ip := range info.IPs {
					resolverObj.Put(resolver.Address{
						Addr: ip + ":" + port,
					})
				}
			case watch.Modified:
				for _, ip := range info.IPs {
					resolverObj.Put(resolver.Address{
						Addr: ip + ":" + port,
					})
				}
			}
			cc.UpdateState(resolver.State{
				Addresses: resolverObj.GetAddressList(),
			})
			return nil
		}) {
		}
	}()

	return nil
}

func (c *k8sResolver) ProcessWorkItem(f func(info *KubernetesEvent) error) bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(item)
	o := item.(*KubernetesEvent)
	f(o)
	return true
}

func getServiceNameAndNamespaceNameAndPort(urlPath string) (serviceName string, namespaceName string, port string, err error) {
	if !strings.Contains(urlPath, ":") {
		err = fmt.Errorf("getServiceNameAndNamespaceName urlPath is %s, and must have `:` and `port`", urlPath)
		return
	}
	arrs := strings.Split(urlPath, ":")
	if len(arrs) != 2 {
		err = fmt.Errorf("getAppnameAndPort length error")
		return
	}
	serviceNameAndNamespaceName := arrs[0]
	port = arrs[1]
	arr := strings.Split(serviceNameAndNamespaceName, ".")
	serviceName = strings.TrimPrefix(arr[0], "/")
	namespaceName = arr[1]
	return
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
