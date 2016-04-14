package main

import (
	"fmt"
	"os"

	"github.com/cs733-iitb/log"
)

var LOGFILE string = "sample-log"

type Foo struct {
	Bar int
	Baz []string // only public fields (with capitalized names) are stored
}

func main() {
	rmlog() // for  a clean start

	lg, err := log.Open(LOGFILE)
	lg.RegisterSampleEntry(Foo{})

	defer rmlog()
	assert(err == nil)

	lg.Append("t1")
	//res, err := lg.Get(1) // should return "bar"
	lg.Append("t2")
	lg.Append("t3")
	lg.Append("t4")
	lg.Append("t5")
	lg.Append("t6")

	i := lg.GetLastIndex() // should return 2 as an int64 value
	fmt.Println("...", i)

	/*
		lg.TruncateToEnd( /*from 1)
		i = lg.GetLastIndex() // should return 0. One entry is left.
		assert(i == 0)
	*/

	lg.TruncateToEnd(3)
	lg.Append("t4")

	for j := 0; j < 7; j++ {
		res, err1 := lg.Get(int64(j))
		if err1 != nil {
			fmt.Println(err1)
		}
		fmt.Println(res)
	}
}

func assert(val bool) {
	if !val {
		panic("Assertion Failed")
	}
}

func rmlog() {
	os.RemoveAll(LOGFILE)
}
