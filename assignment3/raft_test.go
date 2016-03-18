package main

import (
	"fmt"
	"testing"
)

func TestBasic(t *testing.T) {
	var myRaft Raft
	myRaft = myRaft.makeRafts()
	//time.Sleep(10 * time.Second)
	//fmt.Println(myRaft)
	leaderId := myRaft.GetLeader()
	fmt.Println("Leader Id: ", myRaft.GetLeader())
	myRaft.Cluster[leaderId].Append([]byte("read test"))
	for i := 0; i < PEERS; i++ {
		select {
		case ci := <-myRaft.Cluster[0].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp)
		case ci := <-myRaft.Cluster[1].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp)
		case ci := <-myRaft.Cluster[2].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp)
		case ci := <-myRaft.Cluster[3].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp)
		case ci := <-myRaft.Cluster[4].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp)
			//default:
			//	t.Fatal("Expected message on all nodes")
		}
	}
}
func expect(t *testing.T, ci CommitInfo) {
	fmt.Println("$$$")

	if ci.Err != nil {
		t.Fatal(ci.Err)
	}
	if string(ci.Data) != "read test" {
		t.Fatal("Got different data")
	}
}
