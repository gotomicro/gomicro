package governor

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime/debug"

	"github.com/felixge/fgprof"
	"github.com/gotomicro/ego/core/eapp"
	"github.com/gotomicro/ego/core/elog"
	jsoniter "github.com/json-iterator/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	// DefaultServeMux ...
	DefaultServeMux = http.NewServeMux()
	routes          = []string{}
)

func init() {
	// 获取全部治理路由
	HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		_ = json.NewEncoder(resp).Encode(routes)
	})
	HandleFunc("/debug/fgprof", fgprof.Handler().(http.HandlerFunc))
	HandleFunc("/debug/pprof/", pprof.Index)
	HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	HandleFunc("/debug/pprof/profile", pprof.Profile)
	HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	HandleFunc("/debug/pprof/trace", pprof.Trace)
	if info, ok := debug.ReadBuildInfo(); ok {
		HandleFunc("/module/info", func(w http.ResponseWriter, r *http.Request) {
			encoder := json.NewEncoder(w)
			if r.URL.Query().Get("pretty") == "true" {
				encoder.SetIndent("", "    ")
			}
			_ = encoder.Encode(info)
		})
	}

	HandleFunc("/env/info", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_ = jsoniter.NewEncoder(w).Encode(os.Environ())
	})
	HandleFunc("/build/info", func(w http.ResponseWriter, r *http.Request) {
		serverStats := map[string]string{
			"name":       eapp.Name(),
			"appMode":    eapp.AppMode(),
			"appVersion": eapp.AppVersion(),
			"egoVersion": eapp.EgoVersion(),
			"buildUser":  eapp.BuildUser(),
			"buildHost":  eapp.BuildHost(),
			"buildTime":  eapp.BuildTime(),
			"startTime":  eapp.StartTime(),
			"hostName":   eapp.HostName(),
			"goVersion":  eapp.GoVersion(),
		}
		_ = jsoniter.NewEncoder(w).Encode(serverStats)
	})
}

// Component ...
type Component struct {
	logger *zap.Logger
	*http.Server
	listener net.Listener
}

func NewComponent(addr string, logger *zap.Logger) *Component {
	return &Component{
		logger: logger,
		Server: &http.Server{
			Addr:    addr,
			Handler: DefaultServeMux,
		},
		listener: nil,
	}
}

// Start 开始
func (c *Component) Start() error {
	var listener, err = net.Listen("tcp4", c.Server.Addr)
	if err != nil {
		elog.Panic("governor start error", elog.FieldErr(err))
	}
	c.listener = listener
	HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.HandlerFor(
			prometheus.DefaultGatherer,
			promhttp.HandlerOpts{
				// Opt into OpenMetrics to support exemplars.
				EnableOpenMetrics: true,
			},
		).ServeHTTP(w, r)
	})
	c.logger.Info("治理服务启动监听：" + c.Server.Addr)
	err = c.Server.Serve(c.listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err

}

//Stop ..
func (c *Component) Stop() error {
	return c.Server.Close()
}

//GracefulStop ..
func (c *Component) GracefulStop(ctx context.Context) error {
	return c.Server.Shutdown(ctx)
}

// HandleFunc ...
func HandleFunc(pattern string, handler http.HandlerFunc) {
	// todo: 增加安全管控
	DefaultServeMux.HandleFunc(pattern, handler)
	routes = append(routes, pattern)
}
