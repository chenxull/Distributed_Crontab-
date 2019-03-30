package main

import (
	"flag"
	"fmt"
	"github.com/chenxull/Crontab/crontab/master"
	"github.com/chenxull/Crontab/crontab/master/Error"
	"runtime"
)

var (
	confFile string // 配置文件路径
)

//初始传入参数
func initArgs() {
	// master -config ./master.json
	flag.StringVar(&confFile, "config", "master.json", "指定配置文件")
	flag.Parse()
}

//初始化线程，确保线程数和处理器的核心数相同
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var (
		err error
	)
	initArgs()
	initEnv()
	//加载配置
	if err = master.InitConfig(confFile); err != nil {
		Error.CheckErr(err, "InitConfig meet some problems")
	}

	//任务管理器
	if err = master.InitMgr(); err != nil {
		Error.CheckErr(err, "InitMgr meet some problems")
		return
	}
	//启动API http 服务
	if err = master.InitApiServer(); err != nil {
		Error.CheckErr(err, "InitApiServer meet some problems")
		return
	}

	fmt.Println("start successful")

}
