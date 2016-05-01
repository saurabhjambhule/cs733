package raft

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/saurabhjambhule/cs733/assignment4/raft/cluster"
	"github.com/saurabhjambhule/cs733/assignment4/raft/cluster/mock"

	"github.com/saurabhjambhule/cs733/assignment4/raft/sm"
)

var myRaft Raft
var leaderId int
var clusterT *mock.MockCluster
var partCl0 []int
var partCl1 []int
var partCl2 []int

func Test_StartMockCluster(t *testing.T) {
	//Initialization.
	partCl0 = make([]int, 0)
	partCl1 = make([]int, 0)
	partCl2 = make([]int, 0)
	partCl0 = append(partCl0, 1, 2, 3, 4, 5)

	clusterT = new(mock.MockCluster)
	cleanDB()                                 //Clear the old database.
	myRaft, clusterT = myRaft.makeMockRafts() //make mock cluster
	//Simple cluster can also be created using myraft.makeRafts() method.
	time.Sleep(1 * time.Second)

	leaderId = myRaft.GetLeader() //Get current leader.
	_ = partCl1
	_ = partCl2
}

func Test_BasicReplication(t *testing.T) {
	//Appending Entries And check for replication of entry.
	for j := 1; j <= 1000; j++ {
		str := "test - " + strconv.Itoa(j)
		myRaft.Cluster[leaderId].Append([]byte(str))
	}
	//Checking whether Entries are replicated.
	for j := 1; j <= 1000; j++ {
		for i := 0; i < PEERS; i++ {
			str := "test - " + strconv.Itoa(j)
			select {
			case ci := <-myRaft.Cluster[i].SM.CommMedium.CommitInfoCh:
				tmp := ci.(sm.CommitInfo)
				expectF(t, tmp, str, myRaft.Cluster[i].SM.Id)

			case ci := <-myRaft.Cluster[i].SM.CommMedium.CommitCh:
				tmp := ci.(sm.Commit)
				expectL(t, tmp, str, myRaft.Cluster[i].SM.Id)
			}
		}
	}
}

func Test_Partition(t *testing.T) {
	L := myRaft.GetLeader()
	switch L {
	case 0, 1:
		partCl1 = append(partCl1, 1, 2)
		partCl2 = append(partCl2, 3, 4, 5)
		clusterT.Partition(partCl1, partCl2)
	case 2, 3:
		partCl1 = append(partCl1, 3, 4)
		partCl2 = append(partCl2, 1, 2, 5)
		clusterT.Partition(partCl1, partCl2)
	case 4:
		partCl1 = append(partCl1, 4, 5)
		partCl2 = append(partCl2, 1, 2, 3)
		clusterT.Partition(partCl1, partCl2)
	}

	//Appending Entries And check for replication of entry.
	for j := 6; j <= 10; j++ {
		str := "test - " + strconv.Itoa(j)
		myRaft.Cluster[L].Append([]byte(str))
	}

	//Merging the cluster back.
	clusterT.Heal()
	time.Sleep(1 * time.Second)
	leaderId = myRaft.GetLeader()

	//Appending Entries And check for replication of entry.
	for j := 11; j <= 15; j++ {
		str := "test - " + strconv.Itoa(j)
		myRaft.Cluster[leaderId].Append([]byte(str))
	}
	//Checking whether Entries are replicated.
	for j := 6; j <= 15; j++ {
		for i := 0; i < PEERS; i++ {
			str := "test - " + strconv.Itoa(j)
			select {
			case ci := <-myRaft.Cluster[i].SM.CommMedium.CommitInfoCh:
				tmp := ci.(sm.CommitInfo)
				expectF(t, tmp, str, myRaft.Cluster[i].SM.Id)

			case ci := <-myRaft.Cluster[i].SM.CommMedium.CommitCh:
				tmp := ci.(sm.Commit)
				expectL(t, tmp, str, myRaft.Cluster[i].SM.Id)
			}
		}
	}
	partCl2 = partCl2[:0]
	partCl1 = partCl1[:0]
}

func Test_MockShutdown(t *testing.T) {
	//Shutting down on of the server.
	time.Sleep(1 * time.Second)
	downSys := leaderId
	myRaft.Cluster[leaderId].Shutdown(clusterT)
	partCl0 = append(partCl0[:leaderId], partCl0[leaderId+1:]...)
	time.Sleep(1 * time.Second)

	leaderId = myRaft.GetLeader()
	//Appending Entries And check for replication of entry.
	for j := 26; j <= 30; j++ {
		str := "test - " + strconv.Itoa(j)
		myRaft.Cluster[leaderId].Append([]byte(str))
	}

	//Checking whether Entries are replicated.
	for j := 26; j <= 30; j++ {
		for i := 0; i < PEERS-1; i++ {
			str := "test - " + strconv.Itoa(j)
			select {
			case ci := <-myRaft.Cluster[partCl0[i]-1].SM.CommMedium.CommitInfoCh:
				tmp := ci.(sm.CommitInfo)
				expectF(t, tmp, str, myRaft.Cluster[partCl0[i]-1].SM.Id)

			case ci := <-myRaft.Cluster[partCl0[i]-1].SM.CommMedium.CommitCh:
				tmp := ci.(sm.Commit)
				expectL(t, tmp, str, myRaft.Cluster[partCl0[i]-1].SM.Id)
			}
		}
	}

	if myRaft.Cluster[downSys].SM.CommitIndex > int32(1020) {
		t.Fatal("Shutdown not working")
	}

}

func expectF(t *testing.T, ci sm.CommitInfo, str string, id int32) {
	if ci.Err != nil {
		t.Fatal(ci.Err)
	}
	if string(ci.Data.Logg) != str {
		t.Fatal(id, "Follower Got different data", str, " - ", string(ci.Data.Logg))
	}
}

func expectL(t *testing.T, ci sm.Commit, str string, id int32) {
	if ci.Err != nil {
		t.Fatal(ci.Err)
	}

	if string(ci.Data.Logg) != str {
		t.Fatal(id, "Leader Got different data", str, " - ", string(ci.Data.Logg))
	}
}

func expectMatch(t *testing.T, id int, str string, str1 string) {
	if str != str1 {
		t.Fatal(id, "Got different data", str, " - ", str1)
	}
}

func cleanDB() {
	os.RemoveAll("./logDir")
	os.RemoveAll("./stateDir")

}

func (myRaft Raft) GetMockLeader(id []int) int {
	for {
		fmt.Print("")
		for i := 0; i < 3; i++ {
			if myRaft.Cluster[id[i]-1].SM.Status == LEAD {
				return id[i]
			}
		}
	}
	return -1
}

func (myRaft Raft) makeRafts() Raft {
	for id := 1; id <= PEERS; id++ {
		myNode := new(RaftMachine)
		SM := new(sm.State_Machine)
		myConf := new(Config)

		server := createNode(id, myConf, SM)
		SM.Id = int32(id)

		myNode.Node = server
		myNode.SM = SM
		myNode.Conf = myConf
		myRaft.Cluster = append(myRaft.Cluster, myNode)
		go startNode(myRaft.Cluster[id-1].Conf, myRaft.Cluster[id-1].Node, myRaft.Cluster[id-1].SM)
	}
	return myRaft
}

func (myRaft Raft) makeMockRafts() (Raft, *mock.MockCluster) {
	//create mock cluster.
	clconfig := cluster.Config{Peers: nil}
	cl, err := mock.NewCluster(clconfig)
	if err != nil {
		panic(err)
	}

	for id := 1; id <= PEERS; id++ {
		//Ojects to store statemachine, config and server node.
		myNode := new(RaftMachine)
		SM := new(sm.State_Machine)
		myConf := new(Config)

		//initialize config and server object.
		server := createMockNode(id, myConf, SM, cl)
		SM.Id = int32(id)
		myNode.Node = server
		myNode.SM = SM
		myNode.Conf = myConf
		//append object related to node into raft array.
		myRaft.Cluster = append(myRaft.Cluster, myNode)
		//start all the processing threads.
		go startNode(myRaft.Cluster[id-1].Conf, myRaft.Cluster[id-1].Node, myRaft.Cluster[id-1].SM)
	}
	return myRaft, cl
}

func printDB(myRaft Raft, len int64) {
	for j := int64(0); j < len; j++ {
		res, err1 := myRaft.Cluster[0].Conf.Lg.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = myRaft.Cluster[1].Conf.Lg.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = myRaft.Cluster[2].Conf.Lg.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = myRaft.Cluster[3].Conf.Lg.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = myRaft.Cluster[4].Conf.Lg.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}
		fmt.Println("")
	}
}
