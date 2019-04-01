package worker

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/etcd/mvcc/mvccpb"

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
	GlobalJonMgr *JobMgr
)

//监听变化
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
		//反序列化 jsonß 得到 job
		if job, err = common.UnpackJob(kvpair.Value); err != nil {
			Error.CheckErr(err, "WatchJobs Unmarshal Json error")
			return
		} else {
			//把这个 job 同步给调度协程(scheduler)
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
			fmt.Println("DUBG::第一次推送任务到调度器", jobEvent.Job.Name) //这个会在第一次启动 worker 节点的时候执行。
			GlobalScheduel.PushJobEvent(jobEvent)

		}
	}

	//2.从该revision向后监听变
	go func() {
		//从Get时刻的后续版本开始监听变化
		watchStartRevision = getResp.Header.Revision + 1
		fmt.Println(watchStartRevision)
		//监听/cron/jobs/目录的后续变化
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix())

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
					fmt.Println("DUBG::保存事件", jobEvent.Job.Name)
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)

				case mvccpb.DELETE: //任务删除事件
					//推一个删除事件给Scheduler  Delete /cron/jobs/jobName
					//得到将要被删除的任务名
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
					job = &common.Job{Name: jobName}
					//构造一个删除Event
					fmt.Println("DUBG::删除事件", jobEvent.Job.Name)
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, job)

				}
				//TODO:推送给scheduler  GlobalScheduler.PushJobEvent(jobEvent)
				fmt.Println("DEBUG::推送任务到调度器", jobEvent.Job.Name)
				GlobalScheduel.PushJobEvent(jobEvent)
			}
		}

	}()
	return
}

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
	GlobalJonMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	GlobalJonMgr.watchJobs()
	return
}
