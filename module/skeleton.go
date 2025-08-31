// module 包实现了 Leaf 框架的 Skeleton 核心模块
// Skeleton 用于统一管理 goroutine、定时器、异步调用和 RPC 通道

package module

import (
	"time" // Go 标准库 时间处理

	"github.com/name5566/leaf/chanrpc" // chanrpc 包 实现异步 RPC
	"github.com/name5566/leaf/console" // console 包 实现控制台命令
	"github.com/name5566/leaf/go"      // g 包 管理协程池
	"github.com/name5566/leaf/timer"   // timer 包 定时器
)

// Skeleton 结构体 管理 Go 协程池、定时器、异步调用和 RPC
type Skeleton struct {
	GoLen              int               // 协程池长度
	TimerDispatcherLen int               // 定时器分发器长度
	AsynCallLen        int               // 异步调用客户端长度
	ChanRPCServer      *chanrpc.Server   // 用户传入的 RPC 服务器
	g                  *g.Go             // 协程池实例
	dispatcher         *timer.Dispatcher // 定时器分发器实例
	client             *chanrpc.Client   // 异步调用客户端
	server             *chanrpc.Server   // RPC 服务器实例
	commandServer      *chanrpc.Server   // 命令行 RPC 服务器
}

// Init 初始化 Skeleton 配置和内部组件
func (s *Skeleton) Init() {
	// 如果 GoLen 未设置 或 <=0 设置为 0
	if s.GoLen <= 0 {
		s.GoLen = 0
	}
	// 如果 TimerDispatcherLen 未设置 或 <=0 设置为 0
	if s.TimerDispatcherLen <= 0 {
		s.TimerDispatcherLen = 0
	}
	// 如果 AsynCallLen 未设置 或 <=0 设置为 0
	if s.AsynCallLen <= 0 {
		s.AsynCallLen = 0
	}

	// 创建协程池
	s.g = g.New(s.GoLen)
	// 创建定时器分发器
	s.dispatcher = timer.NewDispatcher(s.TimerDispatcherLen)
	// 创建异步调用客户端
	s.client = chanrpc.NewClient(s.AsynCallLen)
	// 使用用户提供的 RPC 服务器
	s.server = s.ChanRPCServer

	// 如果用户没有提供 RPC 服务器，则创建默认 RPC 服务器
	if s.server == nil {
		s.server = chanrpc.NewServer(0)
	}
	// 创建命令行 RPC 服务器
	s.commandServer = chanrpc.NewServer(0)
}

// Run 启动 Skeleton 主循环 监听退出信号和各类通道
func (s *Skeleton) Run(closeSig chan bool) {
	for {
		select {
		// 收到退出信号
		case <-closeSig:
			// 关闭命令行 RPC
			s.commandServer.Close()
			// 关闭普通 RPC
			s.server.Close()
			// 等待协程池和异步调用全部空闲后再关闭
			for !s.g.Idle() || !s.client.Idle() {
				s.g.Close()
				s.client.Close()
			}
			// 退出循环
			return
		// 异步调用返回结果
		case ri := <-s.client.ChanAsynRet:
			s.client.Cb(ri)
		// RPC 请求处理
		case ci := <-s.server.ChanCall:
			s.server.Exec(ci)
		// 命令行 RPC 请求处理
		case ci := <-s.commandServer.ChanCall:
			s.commandServer.Exec(ci)
		// 协程池回调处理
		case cb := <-s.g.ChanCb:
			s.g.Cb(cb)
		// 定时器回调处理
		case t := <-s.dispatcher.ChanTimer:
			t.Cb()
		}
	}
}

// AfterFunc 延迟执行一个定时器回调
func (s *Skeleton) AfterFunc(d time.Duration, cb func()) *timer.Timer {
	// 如果没有开启 TimerDispatcherLen 则 panic
	if s.TimerDispatcherLen == 0 {
		panic("invalid TimerDispatcherLen")
	}

	return s.dispatcher.AfterFunc(d, cb)
}

// CronFunc 注册一个 Cron 表达式定时器
func (s *Skeleton) CronFunc(cronExpr *timer.CronExpr, cb func()) *timer.Cron {
	// 如果没有开启 TimerDispatcherLen 则 panic
	if s.TimerDispatcherLen == 0 {
		panic("invalid TimerDispatcherLen")
	}

	return s.dispatcher.CronFunc(cronExpr, cb)
}

// Go 提交一个函数到协程池执行 并注册回调
func (s *Skeleton) Go(f func(), cb func()) {
	// 如果没有开启 GoLen 则 panic
	if s.GoLen == 0 {
		panic("invalid GoLen")
	}

	s.g.Go(f, cb)
}

// NewLinearContext 创建一个线性协程上下文
func (s *Skeleton) NewLinearContext() *g.LinearContext {
	if s.GoLen == 0 {
		panic("invalid GoLen")
	}

	return s.g.NewLinearContext()
}

// AsynCall 对指定 RPC 服务器发起异步调用
func (s *Skeleton) AsynCall(server *chanrpc.Server, id interface{}, args ...interface{}) {
	if s.AsynCallLen == 0 {
		panic("invalid AsynCallLen")
	}

	s.client.Attach(server)
	s.client.AsynCall(id, args...)
}

// RegisterChanRPC 注册一个 RPC 方法
func (s *Skeleton) RegisterChanRPC(id interface{}, f interface{}) {
	if s.ChanRPCServer == nil {
		panic("invalid ChanRPCServer")
	}

	s.server.Register(id, f)
}

// RegisterCommand 注册一个控制台命令
func (s *Skeleton) RegisterCommand(name string, help string, f interface{}) {
	console.Register(name, help, f, s.commandServer)
}
