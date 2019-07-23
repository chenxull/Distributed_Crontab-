package master

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chenxull/Crontab/crontab/common"
	"github.com/chenxull/Crontab/crontab/master/Error"
	"go.etcd.io/etcd/clientv3"
)

//任务etcd管理器
type JobMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	//单例
	GlobalJobMgr *JobMgr
)

func InitJobMgr() (err error) {

	//配置
	config := clientv3.Config{
		Endpoints:   GlobalConfig.EtcdEndpoints,
		DialTimeout: time.Duration(GlobalConfig.EtcdDialTimeout) * time.Millisecond,
	}

	//建立连接
	client, err := clientv3.New(config)
	if err != nil {
		Error.CheckErr(err, "Etcd New client error ")
		return
	}

	//得到 kv 和 lease 的API子集
	kv := clientv3.KV(client)
	lease := clientv3.Lease(client)

	//配置 单例
	GlobalJobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return
}

//保存任务到 etcd 中
func (jobMgr *JobMgr) Savejob(job *common.Job) (oldjob *common.Job, err error) {

	jobKey := common.JOB_SAVE_DIR + job.Name

	//任务信息,以 json 格式存储在 etcd
	jobValue, err := json.Marshal(job)
	if err != nil {
		Error.CheckErr(err, "Parse job to json error")
		return
	}

	//put到 etcd 中，要将[]byte 类型转换为 string 类型
	putResp, err := jobMgr.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV())
	if err != nil {
		Error.CheckErr(err, "Put job to etcd error")
		return
	}

	var oldJobObj common.Job
	//如果是更新操作，返回旧值
	if putResp.PrevKv != nil {
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldJobObj); err != nil {
			err = nil
			return
		}
		oldjob = &oldJobObj
	}
	return
}

//删除任务
func (jobMgr *JobMgr) DeleteJob(name string) (oldJob *common.Job, err error) {

	jobKey := common.JOB_SAVE_DIR + name
	fmt.Println(jobKey)

	//从 etcd 中删除
	delResp, err := GlobalJobMgr.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV())
	if err != nil {
		Error.CheckErr(err, "Delete job from etcd error ")
		return
	}

	var oldJobObj common.Job
	//返回被删除的信息
	if len(delResp.PrevKvs) != 0 {
		if err = json.Unmarshal(delResp.PrevKvs[0].Value, &oldJobObj); err != nil {
			err = nil
			return
		}
		oldJob = &oldJobObj
	} else {
		fmt.Println("prevkvs is nil")
	}

	return

}

func (jobMgr *JobMgr) ListJobs() (jobList []*common.Job, err error) {

	dirKey := common.JOB_SAVE_DIR
	getResp, err := jobMgr.kv.Get(context.TODO(), dirKey, clientv3.WithPrefix())
	if err != nil {
		Error.CheckErr(err, "List the Job error ")
		return
	}

	//初始化空间
	jobList = make([]*common.Job, 0)
	//遍历所有的任务
	for _, kvPair := range getResp.Kvs {
		job := &common.Job{}
		if err = json.Unmarshal(kvPair.Value, job); err != nil {
			err = nil
			continue
		}
		jobList = append(jobList, job)
	}
	return
}

//杀死任务的接口
func (jobMgr *JobMgr) KillJob(name string) (err error) {
	//实现原理，更新一下 key=/cron/killer/任务名
	//设置租约的目的是让各个 worker 节点能够监听到此次操作
	// 只要设置了租约，worker 节点就可以监听到 ？
	killkey := common.JOB_KILLER_DIR + name

	//让 worker 监听到一次 put 操作，创建一个租约让其稍后自动过期
	leaseGrantResp, err := jobMgr.lease.Grant(context.TODO(), 1)
	if err != nil {
		Error.CheckErr(err, "Set Lease Grant error")
		return
	}
	//租约ID
	leaseID := leaseGrantResp.ID

	//设置 killer 标记
	if _, err = jobMgr.kv.Put(context.TODO(), killkey, "", clientv3.WithLease(leaseID)); err != nil {
		Error.CheckErr(err, "KillJob Put task error ")
		return
	}
	return

}
