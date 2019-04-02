package worker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/chenxull/Crontab/crontab/master/Error"
)

//Config 从 master.json 文件中传入的配置信息
type Config struct {
	//通过标签的方式，使用的 json 的反序列化，和 json 文件中数据对应起来。解析到对应的字段中
	EtcdEndpoints         []string `json:"etcdEndpoints"`
	EtcdDialTimeout       int      `json:"etcdDialTimeout"`
	MongodbUri            string   `json:"mongodbUri"`
	MongodbConnectTimeout int      `json:"mongodbConnectTimeout"`
	JobLogBatchSize       int      `json:"jobLogBatchSize"`
	JobLogCommitTimeOut   int      `json:"jobLogCommitTimeOut"`
}

//单例，其他模块可以直接访问到
var (
	GlobalConfig *Config
)

//InitConfig 加载配置
func InitConfig(filename string) (err error) {
	var (
		content []byte
		Conf    Config
	)
	//1. 读取配置文件
	if content, err = ioutil.ReadFile(filename); err != nil {
		Error.CheckErr(err, "Read the Config file error")
		return
	}

	//2.做反序列化处理，将读取到的配置文件信息存储到Config结构体中
	if err = json.Unmarshal(content, &Conf); err != nil {
		Error.CheckErr(err, "Parse the content to Conf error")
		return
	}
	GlobalConfig = &Conf

	fmt.Println(Conf)

	return
}
