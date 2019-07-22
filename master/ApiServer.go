package master

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/chenxull/Crontab/crontab/common"
	"github.com/chenxull/Crontab/crontab/master/Error"
)

//ApiServer 任务的 http 接口
type ApiServer struct {
	httpServer *http.Server
}

//保存任务接口
//POST job= {"name":"job","command":"echo hello","cronExpr":"*/5 * * * * *"}
func handleJobServe(w http.ResponseWriter, r *http.Request) {
	//提前把变量声明好，便于理解变量的含义
	var (
		job    common.Job
		oldJob *common.Job
		bytes  []byte
	)
	//任务保存到 etcd 中
	//1.解析表单
	err := r.ParseForm()
	if err != nil {
		Error.CheckErr(err, "HandleJobServe:Parse Form error")
		return
	}

	//2.取表单中的 job 字段
	postJob := r.PostForm.Get("job")
	fmt.Print("DEBUG::保存任务", postJob)

	//3.反序列化 job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		Error.CheckErr(err, "Parse postJon to Job struct error")
		return
	}

	//4.保存到 etcd
	if oldJob, err = GlobalJobMgr.Savejob(&job); err != nil {
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
		err        error
		deletename string
		oldJob     *common.Job
		bytes      []byte
	)
	//1.解析表单
	if err = r.ParseForm(); err != nil {
		Error.CheckErr(err, "HandleJobDelete:Parse Form error")
		return
	}

	//2.删除任务名 TODO 无法获取文件名，等待修复
	deletename = r.PostForm.Get("name")
	fmt.Println("DEBUG::删除任务", deletename)
	//3.删除任务
	if oldJob, err = GlobalJobMgr.DeleteJob(deletename); err != nil {
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
}

//显示 etcd 中的任务
func handleJobList(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		jobList []*common.Job
	)

	//1.解析表单
	if err = r.ParseForm(); err != nil {
		Error.CheckErr(err, "HandleJobList: Parse Form error")
		return
	}

	//获取任务列表
	if jobList, err = GlobalJobMgr.ListJobs(); err != nil {
		Error.CheckErr(err, "Get the jobs list error")
		return
	}

	//返回数据
	if bytes, err := common.BuildResponse(0, "success", jobList); err == nil {
		w.Write(bytes)
	}
}

func handleJobKill(w http.ResponseWriter, r *http.Request) {
	var (
		err error
	)

	if err = r.ParseForm(); err != nil {
		Error.CheckErr(err, "HandleJobKill Parse Form error")
		return
	}
	//杀死任务
	killName := r.PostForm.Get("name")

	if err = GlobalJobMgr.KillJob(killName); err != nil {
		Error.CheckErr(err, "Kill job error ")
		return
	}

	bytes, err := common.BuildResponse(0, "success", nil)
	if err == nil {
		w.Write(bytes)
	} else {
		Error.CheckErr(err, "BuildRespons error")
		w.Write(bytes)
	}
}

//查询任务日志
func handleJobLog(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		skip   int
		limit  int
		logArr []*common.JobLog
		bytes  []byte
	)
	if err = r.ParseForm(); err != nil {
		Error.CheckErr(err, "HandleJobLog parse form error")
		return
	}

	//获取请求参数 /job/log?name = job1&skip=0&limit=10
	name := r.Form.Get("name")
	skipParam := r.Form.Get("skip")
	LimitParam := r.Form.Get("limit")

	if len(skipParam) != 0 && len(LimitParam) != 0 {
		if skip, err = strconv.Atoi(skipParam); err != nil {
			Error.CheckErr(err, "HandleJobLog parse string to byte error ")
			skip = 0

		}

		if limit, err = strconv.Atoi(LimitParam); err != nil {
			Error.CheckErr(err, "HandleJobLog parse string to byte error ")
			limit = 20
		}
	}

	skip = 0
	limit = 20
	if logArr, err = GlobalLogMgr.ListLog(name, skip, limit); err != nil {
		Error.CheckErr(err, "获取日志信息失败")
	}

	if bytes, err = common.BuildResponse(0, "success", logArr); err != nil {
		Error.CheckErr(err, "Build LogList Response error")
		return
	}
	w.Write(bytes)

}

//服务发现
func handleWorkerList(w http.ResponseWriter, r *http.Request) {
	var (
		workerArr []string
		bytes     []byte
		err       error
	)
	if workerArr, err = GlobalWorkerMgr.ListWorkers(); err != nil {
		Error.CheckErr(err, "APISERVER:获取 worker IP 失败")
	}
	if bytes, err = common.BuildResponse(0, "success", workerArr); err != nil {
		Error.CheckErr(err, "Build LogList Response error")
		return
	}
	w.Write(bytes)

}

var (
	//GlobalAPIServer 单例对象，供其他包访问这个变量
	GlobalAPIServer *ApiServer
)

//InitApiServer 初始化后端服务器
func InitApiServer() (err error) {

	var (
		listener      net.Listener
		httpServer    *http.Server
		staticDir     http.Dir     //静态文件目录
		staticHandler http.Handler // 静态文件处理
	)

	//配置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobServe) //注册服务，当web 端请求对应的路径时，就会调用对应函数
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)
	mux.HandleFunc("/job/log", handleJobLog)
	mux.HandleFunc("/worker/list", handleWorkerList)

	//静态文件目录
	staticDir = http.Dir(GlobalConfig.Webroot)
	staticHandler = http.FileServer(staticDir)
	mux.Handle("/", http.StripPrefix("/", staticHandler)) // /index.html/  --> ./webroot/index.html

	//启动监听
	fmt.Println(GlobalConfig.APIPort)
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(GlobalConfig.APIPort)); err != nil {
		Error.CheckErr(err, "start Listener service error  ")
		return
	}

	//创建服务器
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(GlobalConfig.APIReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(GlobalConfig.APIWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}
	//配置单例
	GlobalAPIServer = &ApiServer{
		httpServer: httpServer,
	}
	//启动服务
	go httpServer.Serve(listener)

	return
}
