package worker

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"time"

	"github.com/chenxull/Crontab/crontab/common"
	"github.com/chenxull/Crontab/crontab/master/Error"
	"go.etcd.io/etcd/clientv3"
)

//任务etcd管理器
type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

var (
	//单例
	GlobalJobMgr *JobMgr
)

//InitJobMgr 初始化worker的任务管理器
func InitJobMgr() (err error) {

	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
	)
	//配置
	config = clientv3.Config{
		Endpoints:   GlobalConfig.EtcdEndpoints,
		DialTimeout: time.Duration(GlobalConfig.EtcdDialTimeout) * time.Millisecond,
	}

	//建立连接
	if client, err = clientv3.New(config); err != nil {
		Error.CheckErr(err, "Etcd New client error ")
		return
	}

	//得到 kv 和 lease 的API子集
	kv = clientv3.KV(client)
	lease = clientv3.Lease(client)
	watcher = clientv3.NewWatcher(client)
	//配置 单例
	GlobalJobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	//监听任务的变化：执行，删除
	GlobalJobMgr.watchJobs()

	//监听任务的强杀
	GlobalJobMgr.watchKiller()
	return
}

//监听/cron/jobs/中任务的变化
func (jobMgr *JobMgr) watchJobs() (err error) {
	var (
		job                *common.Job
		watchStartRevision int64
		watchChan          clientv3.WatchChan
		watchResp          clientv3.WatchResponse
		watchEvent         *clientv3.Event
		jobName            string
		jobEvent           *common.JobEvent
	)
	//1.get /cron/jobs/目录下的所有任务，并且获知当前集群的revision
	getResp, err := jobMgr.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix())
	if err != nil {
		Error.CheckErr(err, "WATCHJOBS:get the save jobs error ")
		return
	}
	//k
	for _, kvpair := range getResp.Kvs {
		//反序列化 json 得到 job
		if job, err = common.UnpackJob(kvpair.Value); err != nil {
			Error.CheckErr(err, "WatchJobs Unmarshal Json error")
			return
		} else {
			//把这个 job 同步给调度协程(scheduler)
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
			fmt.Println("DUBG::第一次推送任务到调度器", jobEvent.Job.Name) //这个会在第一次启动 worker 节点的时候执行。
			GlobalScheduel.PushJobEvent(jobEvent)
		}
		fmt.Println("for 循环")
	}

	//2.从该revision向后监听变
	// todo 此处有 bug，无法监听etcd
	fmt.Println("bug")
	go func() {
		//从Get时刻的后续版本开始监听变化
		watchStartRevision = getResp.Header.Revision + 1
		fmt.Println(watchStartRevision)
		//监听/cron/jobs/目录的后续变化
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix())
		fmt.Println("watch: debug")
		//处理监听事件
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: //保存事件
					//从 etcd 中反序列化Job，然后将结构体数据推送给Scheduler,即推一个更新事件
					if job, err = common.UnpackJob(watchEvent.Kv.Value); err != nil {
						fmt.Println("watchEvent UnpackJob error ")
						continue
					}
					//构建一个更新Event
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
					fmt.Println("DUBG::保存事件", jobEvent.Job.Name)
				case mvccpb.DELETE: //任务删除事件
					//推一个删除事件给Scheduler  Delete /cron/jobs/jobName
					//得到将要被删除的任务名
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
					job = &common.Job{Name: jobName}
					//构造一个删除Event
					fmt.Println("DUBG::删除事件", jobEvent.Job.Name)
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, job)
				default:
					fmt.Println("测试")

				}
				fmt.Println("DEBUG::推送任务到调度器", jobEvent.Job.Name)
				GlobalScheduel.PushJobEvent(jobEvent)
			}
		}

	}()

	return
}

//监听强杀任务
func (jobMgr *JobMgr) watchKiller() {
	//监听/cron/killer/目录
	//从最新版本开始监听
	go func() {

		//监听/cron/killer/目录的后续变化
		watchChan := jobMgr.watcher.Watch(context.TODO(), common.JOB_KILLER_DIR, clientv3.WithPrefix())

		//处理监听事件
		for watchResp := range watchChan {
			for _, watchEvent := range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: //杀死任务事件
					//提取任务名
					killJobName := common.ExtractKillerName(string(watchEvent.Kv.Key))
					job := &common.Job{Name: killJobName}
					killJobEvent := common.BuildJobEvent(common.JOB_EVENT_KILL, job)

					//推送强杀事件给Scheduler
					fmt.Println("DEBUG::强杀事件", killJobEvent.Job.Name)
					GlobalScheduel.PushJobEvent(killJobEvent)
				case mvccpb.DELETE: //killer 标记过期，被自动删除

				}
			}
		}

	}()
}

//CreateJobLock 创建锁
func (jobMgr *JobMgr) CreateJobLock(jobname string) (jobLock *JobLock) {
	//返回锁
	jobLock = InitJobLock(jobname, jobMgr.kv, jobMgr.lease)
	return
}
