package Error

import "fmt"

func CheckErr(err error, info string) {
	if err != nil {
		fmt.Println("ERROR: " + info + " " + err.Error()) // terminate program
	}
}
