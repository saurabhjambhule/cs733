package main

import (
	"github.com/cs733-iitb/log"
	"os"
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

	err = lg.Append("foo")
	assert(err == nil)

	lg.Append("bar")
	res, err := lg.Get(1) // should return "bar"
	assert(err == nil)
	str, ok := res.(string)
	assert(ok)
	assert(str == "bar")

	err = lg.Append(Foo{Bar: 10, Baz: []string{"x", "y"}})
	assert(err == nil)

	i := lg.GetLastIndex() // should return 2 as an int64 value
	assert(i == 2)

	data, err := lg.Get(i) // should return the Foo instance appended above
	assert(err == nil)
	foo, ok := data.(Foo)
	assert(ok)
	assert(foo.Bar == 10 && foo.Baz[1] == "y")

	lg.TruncateToEnd( /*from*/ 1)
	i = lg.GetLastIndex() // should return 0. One entry is left.
	assert(i == 0)
}

func assert(val bool) {
	if !val {
		panic("Assertion Failed")
	}
}

func rmlog() {
	os.RemoveAll(LOGFILE)
}
