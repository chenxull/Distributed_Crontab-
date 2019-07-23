package worker

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/chenxull/Crontab/crontab/common"

	"github.com/chenxull/Crontab/crontab/master/Error"
	"go.etcd.io/etcd/clientv3"
)

// Register 注册节点到 etcd 中 ： /etcd/workers/IP地址
type Register struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease

	localIP string // 本机IP
}

var (
	GlobalRegister *Register
)

//InitRegister 初始化服务注册
func InitRegister() (err error) {

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

	//本机地址
	localIP, err := getLocalIP()
	fmt.Println(localIP)
	if err != nil {
		return
	}
	//配置 单例
	GlobalRegister = &Register{
		client:  client,
		kv:      kv,
		lease:   lease,
		localIP: localIP,
	}

	//服务注册
	go GlobalRegister.keepOnline()
	return
}

//获取本机地址

func getLocalIP() (ipv4 string, err error) {

	//获取所有IP地址
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}

	//取第一个非 lo 的网卡IP
	for _, addr := range addrs {
		//这个网络地址是 ipv4 或ipv6
		ipNet, isIPNet := addr.(*net.IPNet)
		if isIPNet && !ipNet.IP.IsLoopback() {
			//跳过IP v6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String()
				return
			}
		}
	}
	err = common.ErrNoLocalIPFound
	Error.CheckErr(err, "错误")
	return
}

func (register *Register) keepOnline() {
	var (
		cancelCtx  context.Context
		cancelFunc context.CancelFunc
	)

	regkey := common.JOB_WORKER_DIR + register.localIP

	for {

		//创建租约
		leaseGrantResp, err := register.lease.Grant(context.TODO(), 10)
		if err != nil {
			goto RETRY
		}

		//自动续约
		keepAliveChan, err := register.lease.KeepAlive(context.TODO(), leaseGrantResp.ID)
		if err != nil {
			goto RETRY
		}

		cancelCtx, cancelFunc = context.WithCancel(context.TODO())
		//注册到 etcd
		_, err = register.kv.Put(cancelCtx, regkey, "", clientv3.WithLease(leaseGrantResp.ID))
		if err != nil {
			goto RETRY
		}

		for {
			select {
			case keepAliveResp := <-keepAliveChan:
				// 如果没有在活跃状态，取消 worker 节点 IP 的注册
				if keepAliveResp == nil {
					goto RETRY
				}
			}
		}

	}
RETRY:
	time.Sleep(1 * time.Second)
	if cancelFunc != nil {
		cancelFunc()
	}

}
