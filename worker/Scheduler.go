package worker

import (
	"github.com/chenxull/Crontab/crontab/master/common"
)

//Scheduler 任务调度结构体
type Scheduler struct {
	jobEventChan chan *common.JobEvent
}

func initSchedual() {

}
