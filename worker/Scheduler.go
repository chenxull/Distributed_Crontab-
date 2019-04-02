package worker

import (
	"fmt"
	"time"

	"github.com/chenxull/Crontab/crontab/common"
	"github.com/chenxull/Crontab/crontab/master/Error"
)

//Scheduler 任务调度结构体
type Scheduler struct {
	jobEventChan      chan *common.JobEvent
	jobPlanTable      map[string]*common.JobSchedulePlan //任务调度计划表
	jobExecutingTable map[string]*common.JobExecuteInfo  //任务执行信息表
	jobResultChan     chan *common.JobExecuteResult      //任务结果队列
}

var (
	//全局调度器
	GlobalScheduel *Scheduler
)

//处理任务事件
func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	var (
		jobSchedulePlan *common.JobSchedulePlan
		jobExisted      bool
		jobExecuteInfo  *common.JobExecuteInfo
		jobExectuing    bool
		err             error
	)

	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE: //保存任务
		if jobSchedulePlan, err = common.BuildSchedulePlan(jobEvent.Job); err != nil {
			Error.CheckErr(err, "BuildSchedulePlan error")
			return
		}

		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan //将任务添加到任务调度计划表中
	case common.JOB_EVENT_DELETE: //删除任务
		if jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted { //在删除任务之前，要判断任务调度计划表中是否有这个任务，有的话才会选择删除这个任务
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}
	case common.JOB_EVENT_KILL: //强杀任务

		if jobExecuteInfo, jobExectuing = scheduler.jobExecutingTable[jobEvent.Job.Name]; jobExectuing {
			fmt.Println("杀死shell进程", jobExecuteInfo.Job.Name)
			jobExecuteInfo.CancelFunc() //触发 command 杀死对应的shell 子进程
		}
		//通过 context取消 command 的执行,判断任务是否在执行中，是的话才强杀任务
		//判断jobExecuteInfo 是否存在于jobExecutingTable中，如果存在的话，jobExectuing为 true

		//BUGTODIX:无法执行杀死 shell 进程逻辑，可能原jobExecutingTable表中没有存储数据

	}
}

//处理任务的结果
func (scheduler *Scheduler) handleJobResutl(jobResult *common.JobExecuteResult) {

	var (
		jobLog *common.JobLog
	)
	delete(scheduler.jobExecutingTable, jobResult.ExecuteInfo.Job.Name) //删除jobExecutingTabl中的任务状态
	//生成日志
	if jobResult.Err != common.ErrLockAlreadyRequired {
		jobLog = &common.JobLog{
			JobName:      jobResult.ExecuteInfo.Job.Name,
			Command:      jobResult.ExecuteInfo.Job.Commond,
			OutPut:       string(jobResult.OutPut),
			PlanTime:     jobResult.ExecuteInfo.PlanTime.UnixNano() / 1000 / 1000,
			ScheduleTime: jobResult.ExecuteInfo.RealTime.UnixNano() / 1000 / 1000,
			StartTime:    jobResult.StartTime.UnixNano() / 1000 / 1000,
			EndTime:      jobResult.EndTime.UnixNano() / 1000 / 1000,
		}
		if jobResult.Err != nil {
			jobLog.Err = jobResult.Err.Error()
		} else {
			jobLog.Err = ""
		}
		GlobalLogSink.Append(jobLog)
	}

	//TODO:数据存储的 MongoDB 中

	fmt.Println("任务执行完成", jobResult.ExecuteInfo.Job.Name, string(jobResult.OutPut), jobResult.Err)
}

//TryStartJob 尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan) {
	var (
		jobExecuteInfo *common.JobExecuteInfo //单个任务执行的状态
		jobExisted     bool
	)

	fmt.Println("打印任务执行列表中的数据:")
	for testname, testJobinfo := range scheduler.jobExecutingTable {
		fmt.Println("name:", testname, "jobexecuteInfo", testJobinfo)
	}

	//执行的时间可能会很长，比如说任务1分钟会调度60次，但是只能执行1次，这个时候就需要任务去重
	//判断需要执行的任务是否正在执行，如果是，直接跳过
	if jobExecuteInfo, jobExisted = scheduler.jobExecutingTable[jobPlan.Job.Name]; jobExisted {
		fmt.Println("当前任务正在执行，跳过此次执行", jobExecuteInfo.Job.Name)
		return
	}

	jobExecuteInfo = common.BuildExecuteInfo(jobPlan) //构建执行状态

	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo //保存执行状态到 任务执行状态表中

	fmt.Println("执行任务:", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	GlobalExecutor.ExecuteJob(jobExecuteInfo)

}

//TrySchedule 重新计算任务调度状态并执行到期任务   ,计算休眠时间
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan  *common.JobSchedulePlan
		now      time.Time
		nearTime *time.Time //最近将要执行的任务时间
	)

	//如果scheduler.jobPlanTable 为空的话，睡眠1秒
	if len(scheduler.jobPlanTable) == 0 {
		scheduleAfter = 1 * time.Second
		fmt.Println("任务调度表中暂时没有任务。")
		return
	}

	now = time.Now()

	// 遍历所有任务
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			scheduler.TryStartJob(jobPlan)            //尝试执行任务
			jobPlan.NextTime = jobPlan.Expr.Next(now) //更新下次执行的时间
		}

		//3.统计最近将要过期的任务的时间(N 秒后过期 == schedulerAfter ，过期就立即执行)
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	scheduleAfter = (*nearTime).Sub(now) //设置睡眠时间，也就是下次调度时间：最近要执行的任务的调度时间- 当前时间
	return

}

//for循环不停的监听任务，睡眠时间有TrySchedule确定调度协程
func (scheduler *Scheduler) scheduleLoop() {
	var (
		jobEvent      *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult     *common.JobExecuteResult
	)

	scheduleAfter = scheduler.TrySchedule() //初始化睡眠时间，第一次执行时为1秒

	scheduleTimer = time.NewTimer(scheduleAfter) //设置定时器

	for {
		select {
		case jobEvent = <-scheduler.jobEventChan: //监听任务变化事件
			scheduler.handleJobEvent(jobEvent) //对内存中维护的任务列表做增删改查
		case <-scheduleTimer.C: //TODO: 最近的任务到期了，在没到期这个 for 循环是阻塞的吗?
		case jobResult = <-scheduler.jobResultChan: //监听任务结果时间
			scheduler.handleJobResutl(jobResult)
		}
		scheduleAfter = scheduler.TrySchedule() //调度一次任务
		scheduleTimer.Reset(scheduleAfter)      //更新一下定时器

	}
}

//PushJobEvent 推送任务变化事件 ,这里接受到任务之后，通过 channel 发送给 scheduleLoop
func (scheduler *Scheduler) PushJobEvent(job *common.JobEvent) {
	scheduler.jobEventChan <- job
}

//InitScheduel 初始化调度器
func InitScheduel() (err error) {

	GlobalScheduel = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 1000),
		jobPlanTable:      make(map[string]*common.JobSchedulePlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan:     make(chan *common.JobExecuteResult, 1000),
	}

	/*
		使用协程起一个调度循环，不停的进行调度任务。
		调度任务的本质就是接受来自 JobMgr 中的 watchJob 函数调用 PushJobEvent 传来的任务事件，
		任务事件被传递到 scheduleLoop中后，根据任务的类型不同将相应的任务添加到 jobscheduleTable中或从中删除，
	*/
	go GlobalScheduel.scheduleLoop()
	return
}

//PostJobResult 获取任务执行结果
func (scheduler *Scheduler) PostJobResult(jobResult *common.JobExecuteResult) {
	scheduler.jobResultChan <- jobResult
}
