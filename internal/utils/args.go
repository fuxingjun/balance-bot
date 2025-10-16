package utils

import (
	"flag"
	"sync"
)

type Args struct {
	Host  string
	Port  int
	Debug bool
}

var once sync.Once
var args *Args

func GetArgs() *Args {
	once.Do(func() {
		args = &Args{}
		flag.StringVar(&args.Host, "host", "127.0.0.1", "服务地址")
		flag.IntVar(&args.Port, "port", 12808, "服务端口")
		flag.BoolVar(&args.Debug, "debug", false, "是否开启调试模式")
		flag.Parse()
	})
	return args
}
