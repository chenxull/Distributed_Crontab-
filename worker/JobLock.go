package worker

import (
	"context"
	"fmt"

	"github.com/chenxull/Crontab/crontab/common"

	"github.com/chenxull/Crontab/crontab/master/Error"

	"go.etcd.io/etcd/clientv3"
)

type JobLock struct {
	kv    clientv3.KV
	lease clientv3.Lease

	jobName    string             //任务名
	cancelFunc context.CancelFunc // 用于终止自动续租
	leaseID    clientv3.LeaseID
	isLock     bool //是否上锁成功
}

//InitJobLock 初始化任务锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (jobLock *JobLock) {
	jobLock = &JobLock{
		jobName: jobName,
		kv:      kv,
		lease:   lease,
	}
	return
}

// TryLock 尝试上锁
func (jobLock *JobLock) TryLock() (err error) {
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		cancelCtx      context.Context
		cancelFunc     context.CancelFunc
		leaseID        clientv3.LeaseID
		keepRespChan   <-chan *clientv3.LeaseKeepAliveResponse

		txnResp *clientv3.TxnResponse
	)
	//1.创建租约
	leaseGrantResp, err = jobLock.lease.Grant(context.TODO(), 5)
	if err != nil {
		Error.CheckErr(err, "上锁失败")
		return
	}

	//context 用于自动续租
	cancelCtx, cancelFunc = context.WithCancel(context.TODO())
	//租约ID
	leaseID = leaseGrantResp.ID

	//2.自动续约
	if keepRespChan, err = jobLock.lease.KeepAlive(cancelCtx, leaseID); err != nil {
		Error.CheckErr(err, "TryLock:自动续约失败")
		cancelFunc()
		jobLock.lease.Revoke(cancelCtx, leaseID)
		return
	}
	//3.处理续租应答的协程
	go func() {
		var (
			keepResp *clientv3.LeaseKeepAliveResponse
		)
		for {
			select {
			case keepResp = <-keepRespChan: //自动续租应答
				if keepResp == nil {
					//fmt.Println("DEBUG::续租已经失效了")
					goto END
				} else {
					fmt.Println("DEBUG::收到自动续租应答，续租成功", keepResp.ID)
				}
			}
		}
	END:
	}()

	//4.事务抢占
	txn := jobLock.kv.Txn(context.TODO())

	//生成锁路径
	lockKey := common.JOB_LOCK_DIR + jobLock.jobName
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseID))).
		Else(clientv3.OpGet(lockKey))

	//提交事务
	if txnResp, err = txn.Commit(); err != nil {
		Error.CheckErr(err, "TryLock:提交事务失败")
		cancelFunc()
		jobLock.lease.Revoke(cancelCtx, leaseID)
		return
	}

	//5.成功返回，失败释放租约
	if !txnResp.Succeeded {
		err = common.ErrLockAlreadyRequired
		//fmt.Println("锁被占用了,get 到的内容为： ", string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value))
		//fmt.Println("锁被占用了", jobLock.isLock)

		cancelFunc()
		jobLock.lease.Revoke(cancelCtx, leaseID)
		return
	}

	//抢锁成功
	fmt.Println("上锁成功")
	jobLock.leaseID = leaseID
	jobLock.cancelFunc = cancelFunc
	jobLock.isLock = true

	return
}

//Unlock 释放锁
func (jobLock *JobLock) Unlock() {
	if jobLock.isLock {
		jobLock.cancelFunc()                                  //取消自动续约协程
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseID) // 释放租约
	}

}
