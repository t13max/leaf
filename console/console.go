package console

import (
	"bufio"
	"math"
	"strconv"
	"strings"

	"github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/network"
)

var server *network.TCPServer // 控制台 TCP 服务器实例

// Init 初始化控制台服务
func Init() {
	if conf.ConsolePort == 0 { // 如果未配置端口则不启动控制台
		return
	}

	server = new(network.TCPServer)                             // 创建 TCPServer 实例
	server.Addr = "localhost:" + strconv.Itoa(conf.ConsolePort) // 设置监听地址
	server.MaxConnNum = int(math.MaxInt32)                      // 最大连接数
	server.PendingWriteNum = 100                                // 待写消息缓冲长度
	server.NewAgent = newAgent                                  // 设置新连接回调

	server.Start() // 启动服务器
}

// Destroy 关闭控制台服务
func Destroy() {
	if server != nil {
		server.Close()
	}
}

// Agent 封装控制台连接
type Agent struct {
	conn   *network.TCPConn // TCP 连接对象
	reader *bufio.Reader    // 读取输入缓冲
}

// newAgent 创建新的控制台 Agent
func newAgent(conn *network.TCPConn) network.Agent {
	a := new(Agent)
	a.conn = conn
	a.reader = bufio.NewReader(conn) // 用 bufio.Reader 读取输入
	return a
}

// Run 处理控制台输入命令
func (a *Agent) Run() {
	for {
		if conf.ConsolePrompt != "" {
			a.conn.Write([]byte(conf.ConsolePrompt)) // 输出提示符
		}

		line, err := a.reader.ReadString('\n') // 读取一行输入
		if err != nil {                        // 出现错误则退出循环
			break
		}
		line = strings.TrimSuffix(line[:len(line)-1], "\r") // 去掉换行符

		args := strings.Fields(line) // 按空格拆分命令和参数
		if len(args) == 0 {
			continue
		}
		if args[0] == "quit" { // quit 命令退出
			break
		}

		var c Command
		for _, _c := range commands { // 查找命令实现
			if _c.name() == args[0] {
				c = _c
				break
			}
		}
		if c == nil { // 未找到命令
			a.conn.Write([]byte("command not found, try `help` for help\r\n"))
			continue
		}
		output := c.run(args[1:]) // 执行命令
		if output != "" {
			a.conn.Write([]byte(output + "\r\n")) // 输出命令结果
		}
	}
}

// OnClose 实现 network.Agent 接口的关闭回调
func (a *Agent) OnClose() {}
