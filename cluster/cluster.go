// Package cluster 包实现了 Leaf 框架的集群 TCP 服务和客户端管理
package cluster

import (
	"math" // 用于获取 MaxInt32/MaxUint32
	"time" // 时间处理

	"github.com/name5566/leaf/conf"    // Leaf 框架配置
	"github.com/name5566/leaf/network" // Leaf 框架网络库
)

var (
	server  *network.TCPServer   // 集群服务端实例
	clients []*network.TCPClient // 集群客户端列表
)

// Init 初始化集群服务端和客户端
func Init() {
	// 如果配置了 ListenAddr，则启动 TCPServer
	if conf.ListenAddr != "" {
		server = new(network.TCPServer)               // 创建 TCPServer 实例
		server.Addr = conf.ListenAddr                 // 设置监听地址
		server.MaxConnNum = int(math.MaxInt32)        // 最大连接数
		server.PendingWriteNum = conf.PendingWriteNum // 待发送队列长度
		server.LenMsgLen = 4                          // 消息长度字段长度
		server.MaxMsgLen = math.MaxUint32             // 最大消息长度
		server.NewAgent = newAgent                    // 新连接回调

		server.Start() // 启动服务端
	}

	// 遍历配置的连接地址，创建 TCP 客户端
	for _, addr := range conf.ConnAddrs {
		client := new(network.TCPClient)              // 创建 TCPClient 实例
		client.Addr = addr                            // 设置服务器地址
		client.ConnNum = 1                            // 每个地址连接数量
		client.ConnectInterval = 3 * time.Second      // 重连间隔
		client.PendingWriteNum = conf.PendingWriteNum // 待发送队列长度
		client.LenMsgLen = 4                          // 消息长度字段长度
		client.MaxMsgLen = math.MaxUint32             // 最大消息长度
		client.NewAgent = newAgent                    // 新连接回调

		client.Start()                    // 启动客户端
		clients = append(clients, client) // 添加到客户端列表
	}
}

// Destroy 关闭集群服务端和所有客户端
func Destroy() {
	// 关闭服务端
	if server != nil {
		server.Close()
	}
	// 关闭客户端
	for _, client := range clients {
		client.Close()
	}
}

// Agent 封装 TCP 连接
type Agent struct {
	conn *network.TCPConn // TCP 连接对象
}

// newAgent 创建新的 Agent 实例
func newAgent(conn *network.TCPConn) network.Agent {
	a := new(Agent)
	a.conn = conn
	return a
}

// Run 实现 network.Agent 接口的 Run 方法
func (a *Agent) Run() {}

// OnClose 实现 network.Agent 接口的 OnClose 方法
func (a *Agent) OnClose() {}
