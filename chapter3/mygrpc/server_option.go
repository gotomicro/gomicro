package mygrpc

//ServerOption option of server
type ServerOption struct {
	serverName string
	address    string
	registry   *Registry
}

type ServerOptions func(o *ServerOption)

//WithRegistry set registry
func WithRegistry(r *Registry) ServerOptions {
	return func(o *ServerOption) {
		o.registry = r
	}
}

func WithServerName(sn string) ServerOptions {
	return func(o *ServerOption) {
		o.serverName = sn
	}
}

func WithAddress(address string) ServerOptions {
	return func(o *ServerOption) {
		o.address = address
	}
}
