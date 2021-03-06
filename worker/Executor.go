package worker

import (
	"fmt"
	"math/rand"
	"os/exec"
	"time"

	"github.com/chenxull/Crontab/crontab/common"
)

//Executor 执行器
type Executor struct {
}

var (
	GlobalExecutor *Executor
)

//ExecuteJob 执行任务,执行任务实质上调用 exec.CommandContext执行任务的 command
func (executor *Executor) ExecuteJob(Info *common.JobExecuteInfo) {

	//任务执行协程
	go func() {

		var (
			cmd     *exec.Cmd
			err     error
			output  []byte
			result  *common.JobExecuteResult
			joblock *JobLock
		)

		result = &common.JobExecuteResult{
			ExecuteInfo: Info,
			OutPut:      make([]byte, 0),
		}

		joblock = GlobalJobMgr.CreateJobLock(Info.Job.Name) //首先获取分布式锁

		result.StartTime = time.Now()

		//随机随眠
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		err = joblock.TryLock() //上锁
		defer joblock.Unlock()

		if err != nil {
			result.Err = err
			result.EndTime = time.Now()
			fmt.Println("上锁失败::", err)
		} else {
			result.StartTime = time.Now() //上锁后，获得任务 的执行时间
			//核心语句，用来执行 shell 脚本
			cmd = exec.CommandContext(Info.CancelCtx, "/bin/bash", "-c", Info.Job.Commond) //抢到锁就执行，不然就无法执行,Info.CancelCtx 可控的，用于强杀任务的指令
			//获取任务执行结果
			output, err = cmd.CombinedOutput() //获取返回的结果
			result.EndTime = time.Now()        //结束时间
			result.OutPut = output
			result.Err = err

		}
		GlobalScheduel.PostJobResult(result) // 任务执行完成之后，需要把任务执行结果返回个 Scheduler，Scheduler会从ExecutingTabl中将正在执行的任务状态给删除
	}()

}

//InitExecutor 初始化执行器
func InitExecutor() (err error) {
	GlobalExecutor = &Executor{}
	return
}
