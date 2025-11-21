package appx

import (
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMonitorService 创建监控服务。
// 支持传入 mws 中间件对 /metrics, /healthz, /debug/pprof 进行保护。
//
// 示例 - 添加 Basic Auth:
//
//	app.Add(server.NewMonitorService(":9090", healthHandler,
//	  httpx.AuthBasic(myValidator, "Monitor"),
//	))
func NewMonitorService(addr string, healthHandler http.Handler, mws ...func(http.Handler) http.Handler) *HttpService {
	mux := http.NewServeMux()

	// 1. Dynamic Health Check
	if healthHandler != nil {
		mux.Handle("/healthz", healthHandler)
	} else {
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})
	}

	// 2. Metrics (Prometheus)
	mux.Handle("/metrics", promhttp.Handler())

	// 3. Pprof
	// 注意：pprof 默认注册在 DefaultServeMux，我们需要手动注册到这个 mux 以实现隔离
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// 4. 应用中间件 (洋葱模型：后传入的先执行)
	var handler http.Handler = mux
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}

	return NewHttpService("monitor", addr, handler)
}
