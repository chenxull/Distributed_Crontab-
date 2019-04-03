package master

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/chenxull/Crontab/crontab/master/Error"
)

//Config 从 master.json 文件中传入的配置信息
type Config struct {
	//通过标签的方式，使用的 json 的反序列化，和 json 文件中数据对应起来。解析到对应的字段中
	APIPort               int      `json:"apiPort"`
	APIReadTimeout        int      `json:"apiReadTimeout"`
	APIWriteTimeout       int      `json:"apiWriteTimeout"`
	EtcdEndpoints         []string `json:"etcdEndpoints"`
	EtcdDialTimeout       int      `json:"etcdDialTimeout"`
	Webroot               string   `json:"webroot"`
	MongodbUri            string   `json:"mongodbUri"`
	MongodbConnectTimeout int      `json:"mongodbConnectTimeout"`
}

//单例，其他模块可以直接访问到
var (
	GlobalConfig *Config
)

//InitConfig 加载配置
func InitConfig(filename string) error {

	//1. 读取配置文件
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		Error.CheckErr(err, "Read the Config file error")
		return err
	}
	var Conf Config
	//2.做反序列化处理，将读取到的配置文件信息存储到Config结构体中
	err = json.Unmarshal(content, &Conf)
	if err != nil {
		Error.CheckErr(err, "Parse the content to Conf error")
		return err
	}
	GlobalConfig = &Conf

	fmt.Println(Conf)

	return err
}
