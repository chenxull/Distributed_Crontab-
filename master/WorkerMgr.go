package master

import (
	"context"
	"time"

	"github.com/chenxull/Crontab/crontab/common"

	"github.com/chenxull/Crontab/crontab/master/Error"
	"go.etcd.io/etcd/clientv3"
)

type WorkerMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	GlobalWorkerMgr *WorkerMgr
)

//InitWorkerMgr 初始化方法
func InitWorkerMgr() (err error) {

	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
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

	//得到 kv 和 lease 的api 子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	GlobalWorkerMgr = &WorkerMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}
	return
}

//ListWorkers 获取worker 节点
func (workerMgr *WorkerMgr) ListWorkers() (workerArr []string, err error) {
	//初始化 shuju
	workerArr = make([]string, 0)

	getResp, err := workerMgr.kv.Get(context.TODO(), common.JOB_WORKER_DIR, clientv3.WithPrefix())
	if err != nil {
		Error.CheckErr(err, "ListWorkers get the info error")
		return
	}

	//遍历返回结果，放入到workerArr中
	for _, kv := range getResp.Kvs {
		//kv.key : /cron/woerkers/{ip}
		workerIP := common.ExtractWorkerIP(string(kv.Key))
		workerArr = append(workerArr, workerIP)
	}
	return
}
