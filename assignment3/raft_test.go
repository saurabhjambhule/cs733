package main

import (
	"os"
	"testing"
	"time"

	"github.com/cs733-iitb/cluster/mock"
)

func TestBasic(t *testing.T) {
	//Initialization.
	var myRaft Raft
	cluster := new(mock.MockCluster)
	cluster = new(mock.MockCluster)
	cleanDB()                                //Clear the old database.
	myRaft, cluster = myRaft.makeMockRafts() //make mock cluster
	//Simple cluster can also be created using myraft.makeRafts() method.

	leaderId := myRaft.GetLeader() //Get current leader.

	//Appending Entries And check for replication of entry.
	str := "read test"
	myRaft.Cluster[leaderId].Append([]byte("read test"))
	for i := 0; i < PEERS; i++ {
		select {
		case ci := <-myRaft.Cluster[0].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp, str)
		case ci := <-myRaft.Cluster[1].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp, str)
		case ci := <-myRaft.Cluster[2].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp, str)
		case ci := <-myRaft.Cluster[3].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp, str)
		case ci := <-myRaft.Cluster[4].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp, str)
		}
	}
	//Appending Entries And check for replication of entry.
	str = "read cloud"
	myRaft.Cluster[leaderId].Append([]byte("read cloud"))
	for i := 0; i < PEERS; i++ {
		select {
		case ci := <-myRaft.Cluster[0].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp, str)
		case ci := <-myRaft.Cluster[1].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp, str)
		case ci := <-myRaft.Cluster[2].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp, str)
		case ci := <-myRaft.Cluster[3].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp, str)
		case ci := <-myRaft.Cluster[4].SM.CommMedium.CommitCh:
			tmp := ci.(CommitInfo)
			expect(t, tmp, str)
		}
	}

	//Check non leader server and send timeout signal to it.
	leaderId = myRaft.GetLeader()
	for i := 0; i < PEERS; i++ {
		if i != leaderId {
			myRaft.Cluster[i].SM.CommMedium.timeoutCh <- nil
			break
		}
	}

	//Creating partion.
	cluster.Partition([]int{1, 2, 3, 4}, []int{5}) //Cluster partitions into two.
	time.Sleep(10 * time.Second)

	leaderId = myRaft.GetLeader()

	//Merging the cluster back.
	cluster.Heal()

	leaderId = myRaft.GetLeader()
	_ = cluster
	_ = leaderId
}

func expect(t *testing.T, ci CommitInfo, str string) {
	if ci.Err != nil {
		t.Fatal(ci.Err)
	}
	if string(ci.Data) != str {
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
