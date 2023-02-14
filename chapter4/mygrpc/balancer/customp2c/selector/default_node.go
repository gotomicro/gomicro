package selector

import (
	"fmt"
	"sort"
	"strconv"
)

var _ Node = (*DefaultNode)(nil)

// DefaultNode is selector node
type DefaultNode struct {
	scheme   string
	addr     string
	weight   *int64
	version  string
	name     string
	metadata map[string]string
}

// Scheme is node scheme
func (n *DefaultNode) Scheme() string {
	return n.scheme
}

// Address is node address
func (n *DefaultNode) Address() string {
	return n.addr
}

// ServiceName is node serviceName
func (n *DefaultNode) ServiceName() string {
	return n.name
}

// InitialWeight is node initialWeight
func (n *DefaultNode) InitialWeight() *int64 {
	return n.weight
}

// Version is node version
func (n *DefaultNode) Version() string {
	return n.version
}

// Metadata is node metadata
func (n *DefaultNode) Metadata() map[string]string {
	return n.metadata
}

// NewNode new node
func NewNode(scheme, addr string, ins *ServiceInstance) Node {
	n := &DefaultNode{
		scheme: scheme,
		addr:   addr,
	}
	if ins != nil {
		n.name = ins.Name
		n.version = ins.Version
		n.metadata = ins.Metadata
		if str, ok := ins.Metadata["weight"]; ok {
			if weight, err := strconv.ParseInt(str, 10, 64); err == nil {
				n.weight = &weight
			}
		}
	}
	return n
}

// ServiceInstance is an instance of a service in a discovery system.
type ServiceInstance struct {
	// ID is the unique instance ID as registered.
	ID string `json:"id"`
	// Name is the service name as registered.
	Name string `json:"name"`
	// Version is the version of the compiled.
	Version string `json:"version"`
	// Metadata is the kv pair metadata associated with the service instance.
	Metadata map[string]string `json:"metadata"`
	// Endpoints are endpoint addresses of the service instance.
	// schema:
	//   http://127.0.0.1:8000?isSecure=false
	//   grpc://127.0.0.1:9000?isSecure=false
	Endpoints []string `json:"endpoints"`
}

func (i *ServiceInstance) String() string {
	return fmt.Sprintf("%s-%s", i.Name, i.ID)
}

// Equal returns whether i and o are equivalent.
func (i *ServiceInstance) Equal(o interface{}) bool {
	if i == nil && o == nil {
		return true
	}

	if i == nil || o == nil {
		return false
	}

	t, ok := o.(*ServiceInstance)
	if !ok {
		return false
	}

	if len(i.Endpoints) != len(t.Endpoints) {
		return false
	}

	sort.Strings(i.Endpoints)
	sort.Strings(t.Endpoints)
	for j := 0; j < len(i.Endpoints); j++ {
		if i.Endpoints[j] != t.Endpoints[j] {
			return false
		}
	}

	if len(i.Metadata) != len(t.Metadata) {
		return false
	}

	for k, v := range i.Metadata {
		if v != t.Metadata[k] {
			return false
		}
	}

	return i.ID == t.ID && i.Name == t.Name && i.Version == t.Version
}
