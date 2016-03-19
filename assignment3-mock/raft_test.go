package main

import (
	"fmt"
	"os"
	"testing"
)

func TestBasic(t *testing.T) {
	var myRaft Raft
	cleanDB()
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
		}
	}
	//Shutdown(myRaft.Cluster[leaderId].Node)
	fmt.Println("Leader Id: ", myRaft.LeaderId())
}

func expect(t *testing.T, ci CommitInfo) {
	if ci.Err != nil {
		t.Fatal(ci.Err)
	}
	if string(ci.Data) != "read test" {
		t.Fatal("Got different data")
	}
}

func cleanDB() {
	os.RemoveAll("./Log_1")
	os.RemoveAll("./Log_2")
	os.RemoveAll("./Log_3")
	os.RemoveAll("./Log_4")
	os.RemoveAll("./Log_5")
}
