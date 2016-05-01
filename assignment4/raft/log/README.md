#log

log.Log is an array on disk. It can be appended or truncated (all action is at the end).
It keeps track of the last index to which data is written (-1 if the
log is empty). 

A log entry is a golang gob-encodable object. If it is a struct, its
type must be registered as shown in the usage below.

# Usage

See the working sample in sample/sample.go.
	
```go
     import "github.com/cs733-iitb/log"
     lg, err := log.Open(LOGFILE)
     lg.RegisterSampleEntry(Foo{})
     assert (err == nil) 

     defer lg.Close()
     
     lg, _ := log.Open("mylog")
     defer lg.Close()
     
     data, err := lg.Get(i) // should return the Foo instance appended above
     assert (err == nil)
     foo, ok := data.(Foo) 
     assert(ok)
     assert(foo.Bar == 10 && foo.Baz[1] == "y")
     
     lg.TruncateToEnd(/*from*/ 1)
     i = lg.GetLastIndex() // should return 0. One entry is left.
     assert (i == 0)
     
     bytes, _ := lg.Get(1) // should return "bar" in bytes
     i := lg.GetLastIndex() // should return 2 as an int64 value
     
     lg.TruncateToEnd(/*from*/ 1)
     i = lg.GetLastIndex() // should return 0. One entry is left.
```

# Installation and Dependencies.

    go get -d github.com/cs733-iitb/log
    go test -race github.com/cs733-iitb/log

This library depends on the github.com/syndtr/leveldb

# Internals and Acknowledgments.

Many thanks to the authors of these two packages.
    
    github.com/syndtr/goleveldb
	github.com/hashicorp/golang-lru
	
The log uses the goleveldb key value store underneath as a quick (but hopefully not dirty) solution. It is certainly not as efficient as a proper log will be. It will one day be migrated to a straightforward disk log with support for compaction

##Author: Sriram Srinivasan. sriram _at_ malhar.net
