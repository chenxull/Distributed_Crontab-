package master

import (
	"context"

	"github.com/chenxull/Crontab/crontab/master/Error"

	"github.com/chenxull/Crontab/crontab/common"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//LogMgr MongoDB 管理器
type LogMgr struct {
	client        *mongo.Client
	logCollection *mongo.Collection
}

var (
	GlobalLogMgr *LogMgr
)

//InitLogMgr 初始化日志模块
func InitLogMgr() error {
	client, err := mongo.Connect(context.TODO(),
		options.Client().ApplyURI(GlobalConfig.MongodbUri))

	if err != nil {
		return err
	}

	GlobalLogMgr = &LogMgr{
		client:        client,
		logCollection: client.Database("cron").Collection("log"),
	}

	//启动 MongoDB 处理协程

	return err

}

//ListLog 查看任务日志
func (logMgr *LogMgr) ListLog(name string, skip, limit int) (logArr []*common.JobLog, err error) {

	//初始化 logArr 防止出现空指针问题
	logArr = make([]*common.JobLog, 0)

	//1.生成查询条件
	jobFiliter := &common.JobLogFilter{JobName: name}

	//排序方式
	logSort := &common.SortLogByStartTime{SortOrder: -1}

	// 查询
	cursor, err := logMgr.logCollection.Find(context.TODO(),
		jobFiliter,
		options.Find().SetSort(logSort),
		options.Find().SetLimit(int64(limit)),
		options.Find().SetSkip(int64(skip)))

	if err != nil {
		Error.CheckErr(err, "查询数据库失败")
		return nil, err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		jobLog := &common.JobLog{}

		//反序列化 bson,将游标的信息写入到 jobLog 中
		if err = cursor.Decode(jobLog); err != nil {
			Error.CheckErr(err, "Decode the Log from Cursor error")
			continue

		}
		logArr = append(logArr, jobLog)
	}
	return

}
