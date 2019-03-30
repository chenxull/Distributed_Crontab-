package master

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chenxull/Crontab/crontab/master/Error"
	"github.com/chenxull/Crontab/crontab/master/common"
	"go.etcd.io/etcd/clientv3"
	"time"
)

//任务etcd管理器
type JobMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	//单例
	G_jonMgr *JobMgr
)

func InitMgr() (err error) {

	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
	)
	//配置
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}

	//建立连接
	if client, err = clientv3.New(config); err != nil {
		Error.CheckErr(err, "Etcd New client error ")
		return
	}

	//得到 kv 和 lease 的API子集
	kv = clientv3.KV(client)
	lease = clientv3.Lease(client)

	//配置 单例
	G_jonMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return
}

//保存任务到 etcd 中
func (jobMgr *JobMgr) Savejob(job *common.Job) (oldjob *common.Job, err error) {
	var (
		jobKey    string
		jobValue  []byte //要以 json 格式存储在 etcd 中
		putResp   *clientv3.PutResponse
		oldJobObj common.Job
	)
	jobKey = common.JOB_SAVE_DIR + job.Name

	//任务信息,以 json 格式存储在 etcd
	if jobValue, err = json.Marshal(job); err != nil {
		Error.CheckErr(err, "Parse job to json error")
		return
	}

	//put到 etcd 中
	if putResp, err = jobMgr.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		Error.CheckErr(err, "Put job to etcd error")
		return
	}
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
	var (
		jobKey    string
		delResp   *clientv3.DeleteResponse
		oldJobObj common.Job
	)

	jobKey = common.JOB_SAVE_DIR + name

	//从 etcd 中删除
	//TODO BugToFix can't delete the job from the etcd
	if delResp, err = G_jonMgr.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		Error.CheckErr(err, "Delete job from etcd error ")
		return
	}
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
