package master

import (
	"encoding/json"
	"fmt"
	"github.com/chenxull/Crontab/crontab/master/Error"
	"io/ioutil"
)

//配置
type Config struct {
	//通过标签的方式，使用的 json 的反序列化，和 json 文件中数据对应起来。解析到对应的字段中
	ApiPort         int      `json:"apiPort"`
	ApiReadTimeout  int      `json:"apiReadTimeout"`
	ApiWriteTimeout int      `json:"apiWriteTimeout"`
	EtcdEndpoints   []string `json:"etcdEndpoints"`
	EtcdDialTimeout int      `json:"etcdDialTimeout"`
}

//单例，其他模块可以直接访问到
var (
	G_config *Config
)

// 加载配置
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
	G_config = &Conf

	fmt.Println(Conf)

	return
}
