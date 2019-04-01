package common

import (
	"errors"
	"fmt"
)

var (
	//ErrLockAlreadyRequired 锁被占用
	ErrLockAlreadyRequired = errors.New("锁已经被占用")
)

//CheckErr 错误处理
func CheckErr(err error, info string) {
	if err != nil {
		fmt.Println("ERROR: " + info + " " + err.Error()) // terminate program
	}
}
