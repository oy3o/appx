package appx

import "context"

// Service 定义了一个可以被 Appx 托管生命周期的组件。
// 无论是 HTTP Server, gRPC Server, 还是 Task Runner，都必须实现此接口。
type Service interface {
	// Name 返回服务名称，用于日志记录
	Name() string

	// Start 启动服务。
	// 实现注意：
	// 1. 这是一个非阻塞调用（或在内部启动 goroutine）。
	// 2. 如果启动失败，应立即返回 error。
	// 3. 如果服务是阻塞运行的（如 http.Serve），请在实现内部 go func()。
	Start(ctx context.Context) error

	// Stop 优雅停止服务。
	// 实现注意：
	// 1. 这是一个阻塞调用，直到服务完全停止或 ctx 超时。
	// 2. 必须处理 ctx.Done() 以避免永久阻塞。
	Stop(ctx context.Context) error
}

// ErrorNotifier 定义错误通知回调函数签名
type ErrorNotifier func(error)

// ErrorNotifiable 是一个可选接口。
// 如果 Service 实现了此接口，Appx 会在 Add 时注入一个回调，
// Service 应在发生致命运行时错误（如 serve crashed）时调用此回调。
type ErrorNotifiable interface {
	SetErrorNotify(ErrorNotifier)
}

// HealthChecker 定义健康检查接口
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) error
}

// ShutdownHook 定义关闭时的清理函数 (如关闭 DB)
type ShutdownHook func(ctx context.Context) error
