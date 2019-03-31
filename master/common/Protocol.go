package common

import (
	"encoding/json"
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
