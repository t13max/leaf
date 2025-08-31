// module 包实现了一个模块管理器 用于统一管理各个模块的生命周期
// 模块需要实现 Module 接口 包含初始化 运行和销毁三个阶段
// 管理器支持注册模块 并发运行模块 接收退出信号 并在关闭时依次销毁模块

package module

import (
	"runtime"
	"sync"

	"github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/log"
)

// Module 接口 定义一个模块必须实现的方法
type Module interface {
	OnInit()                // 模块初始化
	OnDestroy()             // 模块销毁
	Run(closeSig chan bool) // 模块运行 通过 closeSig 通道接收退出信号
}

// module 结构体 封装了一个具体的模块实例和它的管理数据
type module struct {
	mi       Module         // 模块实例
	closeSig chan bool      // 通知模块退出的信号通道
	wg       sync.WaitGroup // 用于等待模块运行结束
}

// mods 保存所有已注册的模块
var mods []*module

// Register 注册一个模块
func Register(mi Module) {
	//新建module结构体
	m := new(module)
	//赋值模块
	m.mi = mi
	//新建一个接收布尔值的channel 缓冲区大小是1
	m.closeSig = make(chan bool, 1)
	//填充m进mods
	mods = append(mods, m)
}

// Init 初始化所有模块 并发启动模块的 Run 方法
func Init() {
	// 先依次调用模块的 OnInit
	for i := 0; i < len(mods); i++ {
		mods[i].mi.OnInit()
	}
	// 再启动每个模块的运行逻辑
	for i := 0; i < len(mods); i++ {
		m := mods[i]
		m.wg.Add(1)
		go run(m)
	}
}

// Destroy 关闭所有模块 按逆序依次发退出信号 并等待结束后调用 OnDestroy
func Destroy() {
	for i := len(mods) - 1; i >= 0; i-- {
		m := mods[i]
		// 通知模块退出
		m.closeSig <- true
		// 等待模块 Run 方法结束
		m.wg.Wait()
		// 调用模块的 OnDestroy
		destroy(m)
	}
}

// run 执行模块的 Run 方法 并在结束后标记完成
func run(m *module) {
	m.mi.Run(m.closeSig)
	m.wg.Done()
}

// destroy 调用模块的 OnDestroy 并捕获可能的 panic 打印堆栈
func destroy(m *module) {
	// defer 匿名函数 用于捕获 panic 避免程序崩溃
	defer func() {
		// recover 捕获 panic 返回 panic 的值 r 如果没有 panic r 为 nil
		if r := recover(); r != nil {
			// 如果配置了堆栈缓冲长度 conf.LenStackBuf > 0
			if conf.LenStackBuf > 0 {
				// 创建 buf 存放堆栈信息
				buf := make([]byte, conf.LenStackBuf)
				// runtime.Stack 获取当前 goroutine 的堆栈信息
				l := runtime.Stack(buf, false)
				// 打印 panic 信息和堆栈日志
				log.Error("%v: %s", r, buf[:l])
			} else {
				// 如果没有配置堆栈缓冲长度 只打印 panic 信息
				log.Error("%v", r)
			}
		}
	}()
	// 调用模块的销毁方法
	m.mi.OnDestroy()
}
