package k8s

import (
	"fmt"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type KubernetesEvent struct {
	IPs       []string
	EventType watch.EventType
}

func (c *k8sResolver) addEndpoints(obj interface{}) {
	p, ok := obj.(*v1.Endpoints)
	if !ok {
		c.logger.Warn(fmt.Sprintf("pod-informer got object %T not *v1.Pod", obj))
		return
	}

	addresses := make([]string, 0)
	for _, subsets := range p.Subsets {
		for _, address := range subsets.Addresses {
			addresses = append(addresses, address.IP)
		}
	}

	c.queue.Add(&KubernetesEvent{
		EventType: watch.Added,
		IPs:       addresses,
	})
}

func (c *k8sResolver) updateEndpoints(oldObj, newObj interface{}) {
	c.logger.Debug("updateEndpoints", zap.Any("oldObj", oldObj), zap.Any("newObj", newObj))

	op, ok := oldObj.(*v1.Endpoints)
	if !ok {
		c.logger.Warn(fmt.Sprintf("pod-informer got object %T not *v1.Pod", oldObj))
		return
	}
	np, ok := newObj.(*v1.Endpoints)
	if !ok {
		c.logger.Warn(fmt.Sprintf("pod-informer got object %T not *v1.Pod", newObj))
		return
	}
	if op.GetResourceVersion() == np.GetResourceVersion() {
		return
	}

	addresses := make([]string, 0)
	for _, subsets := range np.Subsets {
		for _, address := range subsets.Addresses {
			addresses = append(addresses, address.IP)
		}
	}

	c.queue.Add(&KubernetesEvent{
		IPs:       addresses,
		EventType: watch.Modified,
	})
}

func (c *k8sResolver) deleteEndpoints(obj interface{}) {
	c.logger.Debug("deleteEndpoints", zap.Any("obj", obj))
	p, ok := obj.(*v1.Endpoints)
	if !ok {
		c.logger.Warn(fmt.Sprintf("pod-informer got object %T not *v1.Pod", obj))
		return
	}

	addresses := make([]string, 0)
	for _, subsets := range p.Subsets {
		for _, address := range subsets.Addresses {
			addresses = append(addresses, address.IP)
		}
	}

	c.queue.Add(&KubernetesEvent{
		IPs:       addresses,
		EventType: watch.Deleted,
	})
}
