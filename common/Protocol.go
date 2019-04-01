package common

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gorhill/cronexpr"

	"github.com/chenxull/Crontab/crontab/master/Error"
)

//Job 定时任务
type Job struct {
	Name     string `json:"name"`
	Commond  string `json:"command"`
	CronExpr string `json:"cronExpr"`
}

//Response http 接口应答
type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

//JobEvent watch 构造的事件，分为二种：更新事件和删除事件
type JobEvent struct {
	EventType int
	Job       *Job
}

//JobSchedulePlan 任务调度计划表
type JobSchedulePlan struct {
	Job      *Job                 //需要调度的任务
	Expr     *cronexpr.Expression //crontab表达式
	NextTime time.Time            //下次执行的时间
}

//JobExecuteInfo    任务执行状态
type JobExecuteInfo struct {
	Job        *Job
	PlanTime   time.Time          //计划执行时间
	RealTime   time.Time          // 实际执行时间
	CancelCtx  context.Context    //任务 command 的 context
	CancelFunc context.CancelFunc //用于取消 command执行 的函数
}

//JobExecuteResult 任务执行结构
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo //任务执行状态
	OutPut      []byte          //脚本输出
	Err         error
	StartTime   time.Time
	EndTime     time.Time
}

//BuildResponse 构建 http 应答方法
func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	//1.定义一个Response
	var (
		response Response
	)
	response.Errno = errno
	response.Msg = msg
	response.Data = data
	//fmt.Println(response.Data)

	//2.序列化为 json
	resp, err = json.Marshal(response)
	return
}

//UnpackJob 反序列化Job
func UnpackJob(value []byte) (ret *Job, err error) {
	var (
		job *Job
	)
	job = &Job{}
	if err = json.Unmarshal(value, job); err != nil {
		Error.CheckErr(err, "Unmarshal the json to Job error")
		return
	}
	ret = job
	return
}

//ExtractJobName 从 etcd 的 key 中提取任务名
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

//ExtractKillerName 从 etcd 的 key 中提取任务名
func ExtractKillerName(killerKey string) string {
	return strings.TrimPrefix(killerKey, JOB_KILLER_DIR)
}

// BuildJobEvent 任务变化事件的构造:  更新事件和 删除事件:
func BuildJobEvent(eventype int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventype,
		Job:       job,
	}
}

//BuildSchedulePlan 构造任务调度计划，即定时任务下一次什么时候执行
func BuildSchedulePlan(job *Job) (jobSchedulePlan *JobSchedulePlan, err error) {

	//解析JOB中的 cron 表达式
	expr, err := cronexpr.Parse(job.CronExpr)
	if err != nil {
		Error.CheckErr(err, "Parse cronexpr error")
		return
	}

	//构造任务调度计划
	jobSchedulePlan = &JobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}
	return
}

//BuildExecuteInfo 构造任务执行状态信息，
func BuildExecuteInfo(jobSchedulePlan *JobSchedulePlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobSchedulePlan.Job,
		PlanTime: jobSchedulePlan.NextTime,
		RealTime: time.Now(),
	}

	//使得 任务执行信息可控，通过可取消的上下文来实现
	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())
	return
}
