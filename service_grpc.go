package appx

import (
	"context"
	"net"
	"time"

	"github.com/oy3o/netx"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

type GrpcService struct {
	name     string
	addr     string
	server   *grpc.Server
	logger   *zerolog.Logger
	listener net.Listener
	onFatal  ErrorNotifier
	maxConns int
}

var _ Service = (*GrpcService)(nil)

func NewGrpcService(name, addr string, srv *grpc.Server) *GrpcService {
	return &GrpcService{
		name:     name,
		addr:     addr,
		server:   srv,
		maxConns: 10000,
	}
}

func (s *GrpcService) WithLogger(l *zerolog.Logger) *GrpcService {
	s.logger = l
	return s
}

func (s *GrpcService) SetErrorNotify(fn ErrorNotifier) {
	s.onFatal = fn
}

func (s *GrpcService) Name() string { return s.name }

func (s *GrpcService) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	// 集成 netx：gRPC 是基于 HTTP/2 的，长时间连接，netx.WithContext 很有用
	ln = netx.Chain(ln,
		netx.WithKeepAlive(5*time.Minute),
		netx.WithContext(nil),
		netx.WithLimit(s.maxConns),
	)
	s.listener = ln

	go func() {
		// 使用统一的 Panic 处理机制
		defer handlePanic(s.logger, s.onFatal)

		// 打印启动信息
		printServiceListening(s.logger, s.name, "gRPC (HTTP/2)", ln.Addr().String())

		if err := s.server.Serve(ln); err != nil {
			if s.logger != nil {
				s.logger.Error().Err(err).Str("name", s.name).Msg("gRPC service crashed")
			}
			if s.onFatal != nil {
				s.onFatal(err)
			}
		}
	}()

	return nil
}

func (s *GrpcService) Stop(ctx context.Context) error {
	// gRPC GracefulStop 是阻塞的，但没有 Context 超时参数
	// 我们可以用一个 goroutine + select 来模拟超时
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.server.Stop() // 强制停止
		return ctx.Err()
	}
}
