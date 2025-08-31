package conf

var (
	LenStackBuf = 4096 // 栈缓冲区大小，用于捕获 panic 时的 stack 信息

	// log 配置
	LogLevel string // 日志等级，例如 "DEBUG", "INFO"
	LogPath  string // 日志文件路径
	LogFlag  int    // 日志输出格式标志

	// console 配置
	ConsolePort   int               // 控制台监听端口
	ConsolePrompt string = "Leaf# " // 控制台提示符
	ProfilePath   string            // 性能分析文件路径

	// cluster 配置
	ListenAddr      string   // 当前服务监听地址，用于集群通信
	ConnAddrs       []string // 要连接的其他集群节点地址列表
	PendingWriteNum int      // 待写消息缓冲队列长度
)
