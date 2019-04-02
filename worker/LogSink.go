package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/chenxull/Crontab/crontab/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LogSink struct {
	client         *mongo.Client
	logCollection  *mongo.Collection
	logChan        chan *common.JobLog
	autoCommitChan chan *common.LogBatch
}

var (
	GlobalLogSink *LogSink
)

//writeLoop 日志批量存储
func (logSink *LogSink) writeLoop() {

	var (
		log          *common.JobLog
		logBatch     *common.LogBatch
		commitTimeer *time.Timer
		timeOutBatch *common.LogBatch
	)
	for {
		select {
		case log = <-logSink.logChan:
			if logBatch == nil {
				logBatch = &common.LogBatch{}

				//超时自动上传日志
				commitTimeer = time.AfterFunc(
					time.Duration(GlobalConfig.JobLogCommitTimeOut)*time.Millisecond,
					//使用闭包，防止指针被改动
					func(batch *common.LogBatch) func() {
						return func() {
							logSink.autoCommitChan <- batch //发送给Case
						}
					}(logBatch),
				)
			}

			//把日志追加到 logbatch 中
			logBatch.Logs = append(logBatch.Logs, log)

			//如果批次满了，立即发送
			if len(logBatch.Logs) >= GlobalConfig.JobLogBatchSize {
				logSink.saveLog(logBatch)
				//清空 logbatch
				logBatch = nil

				//如果在超时时间内，batch 被写满，则取消定时器
				commitTimeer.Stop()
			}

		case timeOutBatch = <-logSink.autoCommitChan: //定时器过期时，发送来的 logbatch

			//可能在1秒时，batch中日志数量刚刚到达100，二个 case 可能会同时触发，这里需要做一个判断 ,超时批次和日志批次是否是同一个
			if timeOutBatch != logBatch { //说明 logbatch 被清空或者进入下一个 batch。因为timeoutBatch 使用了闭包的方法获取 logbatch，它们应该是相同的
				continue //跳过已经被提交的批次
			}
			//写入数据到 MongoDB 中
			logSink.saveLog(timeOutBatch)
			logBatch = nil

		}
	}
}

//saveLog 将日志批量的存储到 MongoDB 中
func (logSink *LogSink) saveLog(batch *common.LogBatch) {
	logSink.logCollection.InsertMany(context.TODO(), batch.Logs)
}

//InitLogSink 初始化日志模块
func InitLogSink() error {
	client, err := mongo.Connect(context.TODO(),
		options.Client().ApplyURI(GlobalConfig.MongodbUri))

	if err != nil {
		return err
	}

	GlobalLogSink = &LogSink{
		client:         client,
		logCollection:  client.Database("cron").Collection("log"),
		logChan:        make(chan *common.JobLog, 1000),
		autoCommitChan: make(chan *common.LogBatch, 1000),
	}

	//启动 MongoDB 处理协程
	go GlobalLogSink.writeLoop()
	return err

}

//Append 发送日志
func (logSink *LogSink) Append(jobLog *common.JobLog) {
	fmt.Println(jobLog)
	select {
	case logSink.logChan <- jobLog:
		fmt.Println("收到数据")
	default:
		//队列满了，丢弃日志
		fmt.Println("日志队列满了，丢弃日志")
	}
}
