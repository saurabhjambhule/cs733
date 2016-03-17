package main

import (
	"fmt"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	var myRaft Raft
	myRaft = myRaft.makeRafts()
	time.Sleep(10 * time.Second)
	fmt.Println("Leader Id: ", myRaft.LeaderId()+1)
	fmt.Println("My Id: ", myRaft.Cluster[0].Id())

	//ldr.Append("foo")
	//time.Sleep(1 time.Second)
	//for _, node:= rafts { select {
	// to avoid blocking on channel.
	//}
	//}
	//case ci := <- node.CommitChannel():
	//if ci.err != nil {t.Fatal(ci.err)} if string(ci.data) != "foo" {
	//t.Fatal("Got different data") default: t.Fatal("Expected message on all nodes")
	//}

}
