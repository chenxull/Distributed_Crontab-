[![996.ICU](https://img.shields.io/badge/link-996.icu-red.svg)](https://996.icu)
# 开发记录

mater节点和 worker 节点之间的交互都是通过 etcd 来实现的。mster 负责向 etcd 中增加，删除，强杀任务;worker 通过监听 etcd 中任务的变化来执行相应的执行任务操作。


## 2019年3月30日
主要以master 节点为主，完成了一系列功能的编码工作

### 实现功能
1. 实现提交任务的Etcd 的 API接口
2. 实现删除任务的 API接口
3. 实现展示 etcd 中任务目录的API接口
4. 实现杀死任务的API接口


## 2019年3月31日

1. 完成与 master 节点进行交互的前端页面，前端页面使用`bootstript + jquery`进行编写。
2. 开始进行 worker 节点的编写工作，实现了监听 etcd 中任务的功能

### 实现功能
1. 前端页面基本功能完成
2. 与后端 master 服务器进行交互，完成编辑任务，删除任务，修改任务，新建任务，强杀任务等功能模块


## 2019年4月1日
1. 完成 worker 节点上的抢占锁功能
2. 完成强杀进程的编写工作，测试还没有通过


## 项目结构
  

### master
控制节点，当用户提交任务后，master 节点首先对这些请求进行路由处理，然后将任务分配给不同的 worker 节点进行执行

### worker
1. 从 etcd 中把 job 同步到内存中,主要是通过 etcd 的 watch 功能，监听任务的变化
2. 实现调度模块，基于 cron 表达式调度N个 job
3. 实现执行模块，并发的执行多个 job
4. 对 job 的分布式锁，放置集群并发
5. 把执行的日志保存到 MongoDB 中



## BUG

- [x] 删除 job 功能存在问题，经过测试无法获取传入的 job name。
- [x] worker节点和 master 节点各个功能正常，但是无法直接通过命令行的形式，对 etcd 服务器进行操作。put 和 get 任务都没用，put 任务后，worker 节点能监听到插入信息，但是会报错，插入不成功。  原因：插入目录写错了，写成了`/crob/jobs/`应该为`ETCDCTL_API=3 ./etcdctl watch --prefix "/cron/jobs/"` 缺少了参数
- [x] 运行worker.go的时候，在释放锁的时候出现问题，出现以下错误:

```
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x1530acb]

goroutine 10 [running]:
github.com/chenxull/Crontab/crontab/worker.(*JobLock).Unlock(0xc0000fff80)
	/Users/chenxu/code/mygo/src/github.com/chenxull/Crontab/crontab/worker/JobLock.go:123 +0x3b
github.com/chenxull/Crontab/crontab/worker.(*Executor).ExecuteJob.func1(0xc0000884c0)
	/Users/chenxu/code/mygo/src/github.com/chenxull/Crontab/crontab/worker/Executor.go:66 +0x35f
created by github.com/chenxull/Crontab/crontab/worker.(*Executor).ExecuteJob
	/Users/chenxu/code/mygo/src/github.com/chenxull/Crontab/crontab/worker/Executor.go:24 +0x3f
exit status 2
```
问题原因：代码逻辑错误，在`JobLock.go`文件中，释放锁逻辑判断错误，应该为当有锁的时候才释放锁，我写成没有锁的时候释放锁。所有才会有空指针错误。


- [ ] 无法执行杀死 shell 进程逻辑，可能原jobExecutingTable表中没有存储数据.经过测试问题就是这里，还没有找到解决办法。
- [ ] 数据没有存储到 MongoDB 中。MongoDB 的启动方式的问题，使用`nohut mongod --dbpath=./mydata &`方式启动只会接受本地的请求，修改为使用`nohut mongod --dbpath=./mydata & --bind_ip=0.0.0.0 &` 后就可以存储数据了，不过这种方式不安全，端口对全网公开