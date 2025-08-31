package leaf

import (
	"os"
	"os/signal"

	"github.com/name5566/leaf/cluster"
	"github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/console"
	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/module"
)

func Run(mods ...module.Module) {
	// logger
	if conf.LogLevel != "" {
		logger, err := log.New(conf.LogLevel, conf.LogPath, conf.LogFlag)
		if err != nil {
			panic(err)
		}
		log.Export(logger)
		defer logger.Close()
	}

	log.Release("Leaf %v starting up", version)

	// module
	for i := 0; i < len(mods); i++ {
		module.Register(mods[i])
	}
	module.Init()

	// cluster
	cluster.Init()

	// console
	console.Init()

	// close
	//创建一个接受系统信号的channel
	c := make(chan os.Signal, 1)
	//注册监听的信号 操作系统信号会发送到c
	signal.Notify(c, os.Interrupt, os.Kill)
	//主协程在这里阻塞等待
	sig := <-c
	log.Release("Leaf closing down (signal: %v)", sig)

	//销毁
	console.Destroy()
	cluster.Destroy()
	module.Destroy()
}
