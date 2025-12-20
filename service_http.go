package appx

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/oy3o/appx/cert"
	"github.com/oy3o/httpx"
	"github.com/oy3o/netx"
	"github.com/oy3o/o11y"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/rs/zerolog"
)

// HttpService 是一个生产级的 HTTP 服务封装。
// 它集成了 netx (限流/保活/ReusePort) 和 cert (TLS) 以及 HTTP/3 (QUIC)。
type HttpService struct {
	name    string
	addr    string
	handler http.Handler
	logger  *zerolog.Logger

	// Options
	certMgr         *cert.Manager // 如果非 nil，开启 TLS
	maxConns        int           // 最大并发连接数 (保护)
	readTimeout     time.Duration // 读超时时间
	keepAlivePeriod time.Duration // keepalive 周期
	enableReusePort bool          // 开启 SO_REUSEPORT
	enableHttp3     bool          // 开启 HTTP/3 (QUIC)

	// Network Middlewares (Layer 4)
	netMiddlewares []netx.Middleware    // TCP 中间件扩展
	udpMiddlewares []netx.UDPMiddleware // UDP 中间件扩展

	// Observability Config
	o11yCfg o11y.Config

	// Runtime
	server      *http.Server
	http3Server *http3.Server  // HTTP/3 Server
	listener    net.Listener   // TCP Listener
	udpConn     net.PacketConn // UDP Listener for QUIC
	onFatal     ErrorNotifier
}

var _ Service = (*HttpService)(nil)

func NewHttpService(name, addr string, handler http.Handler) *HttpService {
	return &HttpService{
		name:            name,
		addr:            addr,
		handler:         handler,
		maxConns:        100000,          // 默认保护：10万并发
		readTimeout:     5 * time.Second, // 默认保护：防止 Slowloris
		keepAlivePeriod: 3 * time.Minute, // 默认 3 分钟
	}
}

// SetErrorNotify 实现 ErrorNotifiable 接口
func (s *HttpService) SetErrorNotify(fn ErrorNotifier) {
	s.onFatal = fn
}

// WithNetMiddleware 注入自定义 TCP 网络层中间件 (如 IP 白名单、Proxy Protocol)
func (s *HttpService) WithNetMiddleware(mws ...netx.Middleware) *HttpService {
	s.netMiddlewares = append(s.netMiddlewares, mws...)
	return s
}

// WithUDPMiddleware 注入自定义 UDP 网络层中间件 (如 PPS 限流)
func (s *HttpService) WithUDPMiddleware(mws ...netx.UDPMiddleware) *HttpService {
	s.udpMiddlewares = append(s.udpMiddlewares, mws...)
	return s
}

// WithKeepAlive 设置 TCP 保活探测间隔
func (s *HttpService) WithKeepAlive(d time.Duration) *HttpService {
	s.keepAlivePeriod = d
	return s
}

// WithTLS 启用 HTTPS
func (s *HttpService) WithTLS(mgr *cert.Manager) *HttpService {
	s.certMgr = mgr
	return s
}

// WithMaxConns 设置最大连接数限制
func (s *HttpService) WithMaxConns(n int) *HttpService {
	s.maxConns = n
	return s
}

// WithLogger 设置 Logger
func (s *HttpService) WithLogger(l *zerolog.Logger) *HttpService {
	s.logger = l
	return s
}

// WithObservability 启用自动化可观测性 (Tracing, Metrics, Logging, Panic Recovery)
// 传入全局 o11y.Config 即可，服务会自动应用 o11y.Handler 中间件。
func (s *HttpService) WithObservability(cfg o11y.Config) *HttpService {
	s.o11yCfg = cfg
	return s
}

// WithReusePort 启用端口复用 (SO_REUSEPORT)
// 允许在多核机器上运行多个进程/线程监听同一端口，由内核进行负载均衡，提升 Accept 吞吐。
func (s *HttpService) WithReusePort() *HttpService {
	s.enableReusePort = true
	return s
}

// WithHTTP3 启用 HTTP/3 (QUIC)
// 注意：HTTP/3 必须依赖 TLS (WithTLS)。
func (s *HttpService) WithHTTP3() *HttpService {
	s.enableHttp3 = true
	return s
}

func (s *HttpService) Name() string { return s.name }

func (s *HttpService) Start(ctx context.Context) error {
	// 1. 启动 TCP 监听 (HTTP/1.1 & HTTP/2)
	// 使用 netx.ListenTCP 支持 ReusePort
	ln, err := netx.ListenTCP("tcp", s.addr, netx.ListenConfig{
		EnableReusePort: s.enableReusePort,
	})
	if err != nil {
		return err
	}
	s.listener = ln

	// 2. 启动 UDP 监听 (HTTP/3)
	var pc net.PacketConn
	if s.enableHttp3 {
		pc, err = netx.ListenUDP("udp", s.addr, netx.ListenConfig{
			EnableReusePort: s.enableReusePort,
		})
		if err != nil {
			ln.Close()
			return err
		}
		s.udpConn = pc
	}

	// 3. [netx] 构建 TCP 网络层增强链
	// 默认基础链：KeepAlive -> User Custom -> Context -> Limit
	// 这样用户的中间件可以在 Context 绑定之前运行 (例如 Proxy Protocol)，也可以在 Limit 之前运行 (例如 IP 黑名单)
	chain := []netx.Middleware{
		netx.WithKeepAlive(s.keepAlivePeriod),
	}
	// 注入用户自定义中间件
	chain = append(chain, s.netMiddlewares...)
	// 注入核心生命周期与保护中间件
	chain = append(chain,
		netx.WithContext(nil),      // 必须：绑定 Context
		netx.WithLimit(s.maxConns), // 必须：过载熔断
	)

	ln = netx.Chain(ln, chain...)

	// 4. [netx] 构建 UDP 网络层增强链
	if pc != nil {
		udpChain := []netx.UDPMiddleware{
			// QUIC 关键优化：增大 Buffer
			netx.WithUDPBuffer(4*1024*1024, 4*1024*1024),
		}
		udpChain = append(udpChain, s.udpMiddlewares...)
		pc = netx.ChainUDP(pc, udpChain...)
	}

	// 3. 证书与 TLS 配置
	var tlsConfig *tls.Config
	protocol := "HTTP"
	if s.certMgr != nil {
		protocol = "HTTPS"
		if err := s.certMgr.Start(ctx); err != nil {
			return err
		}
		tlsConfig = &tls.Config{
			GetCertificate: s.certMgr.GetCertificate, // 无锁化获取
			MinVersion:     tls.VersionTLS13,
			NextProtos:     []string{"h3", "h2", "http/1.1"}, // 增加 h3 协商
		}

		// 绑定 TLS
		ln = tls.NewListener(ln, tlsConfig)
	} else if s.enableHttp3 {
		return errors.New("HTTP/3 requires TLS, please call WithTLS()")
	}

	// 4. 准备 Handler 链
	// 顺序: Alt-Svc (注入头) -> o11y (监控/日志) -> 业务 Handler
	handler := s.handler

	// 如果启用了 o11y，自动包裹中间件
	if s.o11yCfg.Enabled {
		// o11y.Handler 包含了 Trace, Metrics, Panic Recovery 和 Logger Injection
		handler = o11y.Handler(s.o11yCfg)(handler)
	} else {
		// 即使没有 o11y，也添加一个基础 Recovery
		handler = httpx.Recovery(httpx.WithHook(func(ctx context.Context, err error) {
			s.logger.Error().Err(err).Msg("Panic recovered")
		}))(handler)
	}

	// 通过中间件注入 Alt-Svc 头
	if s.enableHttp3 {
		handler = s.altSvcMiddleware(handler)
	}

	// 5. 启动 HTTP/3 监听 (QUIC over UDP)
	if s.enableHttp3 && tlsConfig != nil {
		s.http3Server = &http3.Server{
			Handler:   handler,
			TLSConfig: tlsConfig,
			QUICConfig: &quic.Config{
				MaxIdleTimeout: 30 * time.Second,
			},
		}

		// 异步启动 HTTP/3 Server
		go func() {
			// 防止 QUIC 协程崩溃导致进程退出
			defer handlePanic(s.logger, s.onFatal)

			printServiceListening(s.logger, s.name, "HTTP/3 (QUIC)", pc.LocalAddr().String())

			// Serve 使用现有的 udpConn (ReusePort)
			if err := s.http3Server.Serve(pc); err != nil && !errors.Is(err, quic.ErrServerClosed) {
				if s.logger != nil {
					s.logger.Error().Err(err).Msg("HTTP/3 service error")
				}
				// 通知 Appx 主进程发生致命错误，触发优雅关闭
				if s.onFatal != nil {
					s.onFatal(err)
				}
			}
		}()
	}

	// 6. 启动 HTTP Server (TCP)
	if s.readTimeout <= 0 {
		s.readTimeout = 30 * time.Second // 给 Header 读取充足的时间
	}
	s.server = &http.Server{
		Handler:           handler,
		MaxHeaderBytes:    1 << 20, // 1MB
		ReadHeaderTimeout: s.readTimeout,
		ReadTimeout:       0, // 设为 0，允许上传大文件
		WriteTimeout:      0, // 防御慢速客户端由操作系统的 TCP 缓冲区管理或反向代理层处理更合适
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		// 使用统一的 Panic 处理机制
		defer handlePanic(s.logger, s.onFatal)

		// 打印启动信息
		printServiceListening(s.logger, s.name, protocol, ln.Addr().String())

		if err := s.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			if s.logger != nil {
				s.logger.Error().Err(err).Str("name", s.name).Msg("HTTP service crashed")
			}
			// 通知 Appx 进程退出
			if s.onFatal != nil {
				s.onFatal(err)
			}
		}
	}()

	return nil
}

func (s *HttpService) Stop(ctx context.Context) error {
	var errs []error

	// 1. 关闭 HTTP/3 (如果存在)
	if s.http3Server != nil {
		// http3.Server 目前(quic-go v0.3x) Close 通常会关闭 PacketConn
		if err := s.http3Server.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// 2. 关闭 TCP Server
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// altSvcMiddleware 返回一个中间件，用于在响应头中注入 Alt-Svc
func (s *HttpService) altSvcMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http3.Server.SetQuicHeaders 会根据 Appx 配置计算正确的 Alt-Svc 值
		// 例如: Alt-Svc: h3=":443"; ma=2592000
		if s.http3Server != nil {
			err := s.http3Server.SetQUICHeaders(w.Header())
			if err != nil && s.logger != nil {
				// 仅记录 Debug 日志避免刷屏，这通常不会失败
				s.logger.Debug().Err(err).Msg("Failed to set Alt-Svc header")
			}
		}
		next.ServeHTTP(w, r)
	})
}
