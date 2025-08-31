package console

import (
	"fmt"
	"os"
	"path"
	"runtime/pprof"
	"time"

	"github.com/name5566/leaf/chanrpc"
	"github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/log"
)

// 默认控制台命令列表
var commands = []Command{
	new(CommandHelp),    // 帮助命令
	new(CommandCPUProf), // CPU 性能分析命令
	new(CommandProf),    // pprof 快照命令
}

// Command 接口，所有控制台命令必须实现
type Command interface {
	name() string             // 命令名，线程安全
	help() string             // 命令帮助文本，线程安全
	run(args []string) string // 执行命令，线程安全
}

// ExternalCommand 用于包装注册到 chanrpc.Server 的外部命令
type ExternalCommand struct {
	_name  string
	_help  string
	server *chanrpc.Server
}

// 返回命令名
func (c *ExternalCommand) name() string {
	return c._name
}

// 返回命令帮助文本
func (c *ExternalCommand) help() string {
	return c._help
}

// 执行外部命令，通过 RPC 调用
func (c *ExternalCommand) run(_args []string) string {
	args := make([]interface{}, len(_args))
	for i, v := range _args {
		args[i] = v
	}

	ret, err := c.server.Call1(c._name, args...)
	if err != nil {
		return err.Error()
	}
	output, ok := ret.(string)
	if !ok {
		return "invalid output type"
	}

	return output
}

// 注册外部命令到控制台和 RPC 服务器，必须在 console.Init 之前调用
// 不线程安全
func Register(name string, help string, f interface{}, server *chanrpc.Server) {
	// 检查命令是否已存在
	for _, c := range commands {
		if c.name() == name {
			log.Fatal("command %v is already registered", name)
		}
	}

	server.Register(name, f) // 在 RPC 服务器注册函数

	// 创建 ExternalCommand 并加入命令列表
	c := new(ExternalCommand)
	c._name = name
	c._help = help
	c.server = server
	commands = append(commands, c)
}

// help 命令实现
type CommandHelp struct{}

func (c *CommandHelp) name() string {
	return "help"
}

func (c *CommandHelp) help() string {
	return "this help text"
}

// 输出所有命令及帮助信息
func (c *CommandHelp) run([]string) string {
	output := "Commands:\r\n"
	for _, c := range commands {
		output += c.name() + " - " + c.help() + "\r\n"
	}
	output += "quit - exit console"

	return output
}

// cpuprof 命令实现，用于 CPU 性能分析
type CommandCPUProf struct{}

func (c *CommandCPUProf) name() string {
	return "cpuprof"
}

func (c *CommandCPUProf) help() string {
	return "CPU profiling for the current process"
}

// 命令使用说明
func (c *CommandCPUProf) usage() string {
	return "cpuprof writes runtime profiling data in the format expected by \r\n" +
		"the pprof visualization tool\r\n\r\n" +
		"Usage: cpuprof start|stop\r\n" +
		"  start - enables CPU profiling\r\n" +
		"  stop  - stops the current CPU profile"
}

// 执行 CPU 分析命令
func (c *CommandCPUProf) run(args []string) string {
	if len(args) == 0 {
		return c.usage()
	}

	switch args[0] {
	case "start":
		fn := profileName() + ".cpuprof"
		f, err := os.Create(fn)
		if err != nil {
			return err.Error()
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			f.Close()
			return err.Error()
		}
		return fn
	case "stop":
		pprof.StopCPUProfile()
		return ""
	default:
		return c.usage()
	}
}

// 生成性能分析文件名
func profileName() string {
	now := time.Now()
	return path.Join(conf.ProfilePath,
		fmt.Sprintf("%d%02d%02d_%02d_%02d_%02d",
			now.Year(),
			now.Month(),
			now.Day(),
			now.Hour(),
			now.Minute(),
			now.Second()))
}

// prof 命令实现，用于生成 pprof 快照
type CommandProf struct{}

func (c *CommandProf) name() string {
	return "prof"
}

func (c *CommandProf) help() string {
	return "writes a pprof-formatted snapshot"
}

// 命令使用说明
func (c *CommandProf) usage() string {
	return "prof writes runtime profiling data in the format expected by \r\n" +
		"the pprof visualization tool\r\n\r\n" +
		"Usage: prof goroutine|heap|thread|block\r\n" +
		"  goroutine - stack traces of all current goroutines\r\n" +
		"  heap      - a sampling of all heap allocations\r\n" +
		"  thread    - stack traces that led to the creation of new OS threads\r\n" +
		"  block     - stack traces that led to blocking on synchronization primitives"
}

// 执行 prof 命令
func (c *CommandProf) run(args []string) string {
	if len(args) == 0 {
		return c.usage()
	}

	var (
		p  *pprof.Profile
		fn string
	)
	switch args[0] {
	case "goroutine":
		p = pprof.Lookup("goroutine")
		fn = profileName() + ".gprof"
	case "heap":
		p = pprof.Lookup("heap")
		fn = profileName() + ".hprof"
	case "thread":
		p = pprof.Lookup("threadcreate")
		fn = profileName() + ".tprof"
	case "block":
		p = pprof.Lookup("block")
		fn = profileName() + ".bprof"
	default:
		return c.usage()
	}

	f, err := os.Create(fn)
	if err != nil {
		return err.Error()
	}
	defer f.Close()
	err = p.WriteTo(f, 0)
	if err != nil {
		return err.Error()
	}

	return fn
}
