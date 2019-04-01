package main

import (
	"flag"
	"fmt"
	"runtime"
	"time"

	"github.com/chenxull/Crontab/crontab/master/Error"
	"github.com/chenxull/Crontab/crontab/worker"
)

var (
	confFile string // 配置文件路径
)

//初始传入参数
func initArgs() {
	// worker -config ./master.json
	flag.StringVar(&confFile, "config", "worker.json", "指定配置文件")
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
	if err = worker.InitConfig(confFile); err != nil {
		Error.CheckErr(err, "WORKER:InitConfig meet some problems")
	}

	if err = worker.InitExecutor(); err != nil {
		Error.CheckErr(err, "WORKER:InitExecutor meet some problems")

	}
	//启动调度器
	if err = worker.InitScheduel(); err != nil {
		Error.CheckErr(err, "InitScheduel meet some problems")
	}

	//初始化任务管理器
	if err = worker.InitJobMgr(); err != nil {
		Error.CheckErr(err, "WORKER:InitJobMgr meet some problems")
	}

	for {
		time.Sleep(1 * time.Second)
	}
	fmt.Println("start successful")

}
