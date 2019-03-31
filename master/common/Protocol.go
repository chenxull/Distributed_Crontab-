package common

import (
	"encoding/json"
	"strings"

	"github.com/chenxull/Crontab/crontab/master/Error"
)

//定时任务
type Job struct {
	Name     string `json:"name"`
	Commond  string `json:"command"`
	CronExpr string `json:"cronExpr"`
}

//http接口应答
type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

//JobEvent watch 构造的事件，分为二种：更新事件和删除事件
type JobEvent struct {
	EventType int
	job       *Job
}

//http 应答方法
func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	//1.定义一个Response
	var (
		response Response
	)
	response.Errno = errno
	response.Msg = msg
	response.Data = data
	//fmt.Println(response.Data)

	//2.序列化为 json
	resp, err = json.Marshal(response)
	return
}

//UnpackJob 反序列化Job
func UnpackJob(value []byte) (ret *Job, err error) {
	var (
		job *Job
	)
	job = &Job{}
	if err = json.Unmarshal(value, job); err != nil {
		Error.CheckErr(err, "Unmarshal the json to Job error")
		return
	}
	ret = job
	return
}

//ExtractJobName 从 etcd 的 key 中提取任务名
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

// BuildJobEvent 任务变化事件的构造:  更新事件和 删除事件:
func BuildJobEvent(eventype int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventype,
		job:       job,
	}
}
