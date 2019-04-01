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
		err             error
	)

	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE: //保存任务
		if jobSchedulePlan, err = common.BuildSchedulePlan(jobEvent.Job); err != nil {
			Error.CheckErr(err, "BuildSchedulePlan error")
			return
		}
		//将任务添加到任务调度计划表中
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan //job 名是唯一索引
	case common.JOB_EVENT_DELETE: //删除任务
		//在删除任务之前，要判断任务调度计划表中是否有这个任务，有的话才会选择删除这个任务
		if jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}

	}
}

//处理任务的结果
func (scheduler *Scheduler) handleJobResutl(jobResult *common.JobExecuteResult) {

	//1.删除jobExecutingTabl中的任务状态
	delete(scheduler.jobExecutingTable, jobResult.ExecuteInfo.Job.Name)
	fmt.Println("任务执行完成", jobResult.ExecuteInfo.Job.Name, string(jobResult.OutPut), jobResult.Err)
}

//TryStartJob 尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan) {
	var (
		jobExecuteInfo *common.JobExecuteInfo //单个任务执行的状态
		jobExisted     bool
	)
	//调度和执行不同
	//执行的时间可能会很长，比如说任务1分钟会调度60次，但是只能执行1次，这个时候就需要任务去重
	//判断需要执行的任务是否正在执行，如果是，直接跳过
	if jobExecuteInfo, jobExisted = scheduler.jobExecutingTable[jobPlan.Job.Name]; jobExisted {
		fmt.Println("当前任务正在执行，跳过此次执行", jobExecuteInfo.Job.Name)
		return
	}

	//构建执行状态
	jobExecuteInfo = common.BuildExecuteInfo(jobPlan)

	//保存执行状态到 任务执行状态表中
	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo

	//TODO:执行任务
	fmt.Println("执行任务:", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	GlobalExecutor.ExecuteJob(jobExecuteInfo)

	//TODO 删除执行结束的任务。
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
	//当前时间
	now = time.Now()

	// 1. 遍历所有任务
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			//TODO：2.尝试执行任务
			scheduler.TryStartJob(jobPlan)
			//更新下次执行的时间
			jobPlan.NextTime = jobPlan.Expr.Next(now)
		}

		//3.统计最近将要过期的任务的时间(N 秒后过期 == schedulerAfter ，过期就立即执行)
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	//设置睡眠时间，也就是下次调度时间：最近要执行的任务的调度时间- 当前时间
	scheduleAfter = (*nearTime).Sub(now)
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

	//初始化睡眠时间，第一次执行时为1秒
	scheduleAfter = scheduler.TrySchedule()

	scheduleTimer = time.NewTimer(scheduleAfter) //设置定时器

	for {
		select {
		case jobEvent = <-scheduler.jobEventChan: //监听任务变化事件
			//对内存中维护的任务列表做增删改查
			scheduler.handleJobEvent(jobEvent)
		case <-scheduleTimer.C: //TODO: 最近的任务到期了，在没到期这个 for 循环是阻塞的吗?
		case jobResult = <-scheduler.jobResultChan: //监听任务结果时间
			scheduler.handleJobResutl(jobResult)
		}
		//调度一次任务
		scheduleAfter = scheduler.TrySchedule()

		//更新一下定时器
		scheduleTimer.Reset(scheduleAfter)
	}
}

//PushJobEvent 推送任务变化事件 ,这里接受到任务之后，通过 channel 发送给 scheduleLoop
func (scheduler *Scheduler) PushJobEvent(job *common.JobEvent) {
	scheduler.jobEventChan <- job
}

//InitScheduel 初始化调度器
func InitScheduel() (err error) {

	GlobalScheduel = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 100),
		jobPlanTable:      make(map[string]*common.JobSchedulePlan, 100),
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
