package master

import (
	"encoding/json"
	"fmt"
	"github.com/chenxull/Crontab/crontab/master/Error"
	"github.com/chenxull/Crontab/crontab/master/common"
	"net"
	"net/http"
	"strconv"
	"time"
)

//ApiServer 任务的 http 接口
type ApiServer struct {
	httpServer *http.Server
}

//保存任务接口
//POST job= {"name":"job","command":"echo hello","cronExpr":"*/5 * * * * *"}
func handleJobServe(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		postJob string
		job     common.Job
		oldJob  *common.Job
		bytes   []byte
	)
	//任务保存到 etcd 中
	//1.解析表单
	err = r.ParseForm()
	if err != nil {
		Error.CheckErr(err, "HandleJobServe:Parse Form error")
		return
	}

	//2.取表单中的 job 字段
	postJob = r.PostForm.Get("job")

	//3.反序列化 job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		Error.CheckErr(err, "Parse postJon to Job struct error")
		return
	}

	//4.保存到 etcd
	if oldJob, err = G_jonMgr.Savejob(&job); err != nil {
		Error.CheckErr(err, "ApiServer: Save the job to etcd error ")
		return
	}
	fmt.Println(oldJob)
	//5.返回正常应答
	bytes, err = common.BuildResponse(0, "success", oldJob)
	if err == nil {

		w.Write(bytes) //将数据发送回去
	} else {
		Error.CheckErr(err, "Response message to web error ")
		Errbyte, _ := common.BuildResponse(1, err.Error(), nil)
		w.Write(Errbyte)
	}

}

//删除任务接口 POST /job/delete name = job1
func handleJobDelete(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		name   string
		oldJob *common.Job
		bytes  []byte
	)
	//1.解析表单
	if err = r.ParseForm(); err != nil {
		Error.CheckErr(err, "HandleJobDelete:Parse Form error")
		return
	}

	//2.删除任务名
	name = r.PostForm.Get("name")

	//3.删除任务
	if oldJob, err = G_jonMgr.DeleteJob(name); err != nil {
		Error.CheckErr(err, "DeleteJob from etcd error")
		return
	}

	//4.返回正常应答
	bytes, err = common.BuildResponse(0, "success", oldJob)
	if err == nil {

		w.Write(bytes) //将数据发送回去
	} else {
		Error.CheckErr(err, "Response message to web error ")
		Errbyte, _ := common.BuildResponse(1, err.Error(), nil)
		w.Write(Errbyte)
	}
	return
}

//显示 etcd 中的任务
func handleJoblist(w http.ResponseWriter, r *http.Request) {

}

var (
	//单例对象，供其他包访问这个变量
	G_apiServer *ApiServer
)

func InitApiServer() (err error) {

	var (
		mux        *http.ServeMux
		listener   net.Listener
		httpServer *http.Server
	)

	//配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobServe) //注册服务，当web 端请求对应的路径时，就会调用对应函数
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("job/list", handleJobList)

	//启动监听
	fmt.Println(G_config.ApiPort)
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(G_config.ApiPort)); err != nil {
		Error.CheckErr(err, "start Listener service error  ")
		return
	}

	//创建服务器
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}
	//配置单例
	G_apiServer = &ApiServer{
		httpServer: httpServer,
	}
	//启动服务 TODO 没有进行错误处理
	go httpServer.Serve(listener)

	return
}
