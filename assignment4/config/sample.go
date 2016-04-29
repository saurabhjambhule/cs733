package main

import "fmt"
import (
	"github.com/cs733-iitb/log"
)

var Lg1 *log.Log
var Lg2 *log.Log
var Lg3 *log.Log
var Lg4 *log.Log
var Lg5 *log.Log

func main1() {
	var err error
	Lg1, err = log.Open("Log/Log_1")
	fmt.Println(err)
	Lg2, _ = log.Open("Log/Log_2")
	Lg3, _ = log.Open("Log/Log_3")
	Lg4, _ = log.Open("Log/Log_4")
	Lg5, _ = log.Open("Log/Log_5")
	printDB1(500)

}

func printDB1(len int64) {
	for j := int64(0); j < len; j++ {
		res, err1 := Lg1.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = Lg2.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = Lg3.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = Lg4.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = Lg5.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}
		fmt.Println("")
	}
}
