package raft

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/cluster/mock"
	"github.com/saurabhjambhule/cs733/assignment4/raft/sm"
)

//"github.com/saurabhjambhule/cs733/assignment4/raft/sm"

var cnt int

func TestBasic(t *testing.T) {
	cnt = 0
	//Initialization.
	var myRaft Raft
	cluster := new(mock.MockCluster)
	//cluster = new(mock.MockCluster)
	cleanDB()                                //Clear the old database.
	myRaft, cluster = myRaft.makeMockRafts() //make mock cluster
	//Simple cluster can also be created using myraft.makeRafts() method.

	leaderId := myRaft.GetLeader() //Get current leader.
	fmt.Println("\n\n\nLeader-", leaderId)

	//Appending Entries And check for replication of entry.
	for j := 1; j <= 1; j++ {
		str := "test - " + strconv.Itoa(j)
		myRaft.Cluster[leaderId].Append([]byte(str))
	}
	//Checking whether Entries are replicated.
	for i := 0; i < PEERS; i++ {
		for j := 1; j <= 1; j++ {
			str := "test - " + strconv.Itoa(j)
			ci := <-myRaft.Cluster[i].SM.CommMedium.CommitCh
			tmp := ci.(sm.CommitInfo)
			expect(t, tmp, str, myRaft.Cluster[i].SM.Id)
		}

	}
	/*
		//Find non leader server and send timeout signal to it.
		//for checking whether term increases or not.

		//Creating partion.
		L := myRaft.GetLeader()
		fmt.Println("\n\n\nLeader-", L)
		switch L {
		case 0, 1:
			cluster.Partition([]int{1, 2}, []int{3, 4, 5})
			time.Sleep(1 * time.Second)
			fmt.Println("Mock Leader-", myRaft.GetMockLeader([]int{2, 3, 4}, 3))
		case 2, 3:
			cluster.Partition([]int{1, 2, 5}, []int{3, 4})
			time.Sleep(1 * time.Second)
			fmt.Println("Mock Leader-", myRaft.GetMockLeader([]int{0, 1, 4}, 3))
		case 4:
			cluster.Partition([]int{1, 2, 3}, []int{4, 5})
			time.Sleep(1 * time.Second)
			fmt.Println("Mock Leader-", myRaft.GetMockLeader([]int{0, 1, 2}, 3))
		}

		//time.Sleep(1 * time.Second)
		//leaderId = myRaft.GetLeader()

		//leaderId = myRaft.GetLeader()

		//Appending Entries And check for replication of entry.
		for j := 6; j <= 10; j++ {
			str := "test - " + strconv.Itoa(j)
			myRaft.Cluster[leaderId].Append([]byte(str))
		}
		//time.Sleep(1 * time.Second)

		//Merging the cluster back.
		cluster.Heal()
		for i := 0; i < 5; i++ {
			fmt.Println(myRaft.Cluster[i].SM.Id, "]", myRaft.Cluster[i].SM.Logg)
		}

		time.Sleep(3 * time.Second)

		leaderId = myRaft.GetLeader()
		fmt.Println("\n\n\nLeader-", leaderId)

		for i := 0; i < 5; i++ {
			fmt.Println(myRaft.Cluster[i].SM.Id, ")", myRaft.Cluster[i].SM.Logg)
		}

		//Appending Entries And check for replication of entry.
		for j := 11; j <= 13; j++ {
			str := "test - " + strconv.Itoa(j)
			myRaft.Cluster[leaderId].Append([]byte(str))
		}

		//Printing the commit index of all nodes in cluster.
		for k := 0; k < PEERS; k++ {
			fmt.Println(myRaft.Cluster[k].SM.CommitIndex)

		}
	*/

	time.Sleep(5 * time.Second)
	leaderId = myRaft.GetLeader() //Get current leader.
	fmt.Println("\n\n\nLeader-", leaderId)

	//Printing database entries of all nodes in cluster.
	var j int64
	for j = 0; j < 13; j++ {
		res, err1 := myRaft.Cluster[0].Conf.lg.Get(j)
		if err1 != nil {
			fmt.Println(err1)
		}
		fmt.Print(res)
		fmt.Print("\t")

		res, err1 = myRaft.Cluster[1].Conf.lg.Get(j)
		if err1 != nil {
			fmt.Print(err1)
		}
		fmt.Print(res)
		fmt.Print("\t")
		res, err1 = myRaft.Cluster[2].Conf.lg.Get(j)
		if err1 != nil {
			fmt.Print(err1)
		}
		fmt.Print(res)
		fmt.Print("\t")
		res, err1 = myRaft.Cluster[3].Conf.lg.Get(j)
		if err1 != nil {
			fmt.Print(err1)
		}
		fmt.Print(res)
		fmt.Print("\t")
		res, err1 = myRaft.Cluster[4].Conf.lg.Get(j)
		if err1 != nil {
			fmt.Print(err1)
		}
		fmt.Print(res)
		fmt.Print("\n")
	}
	/*

		fileDB, err := leveldb.OpenFile("./Log_1", nil)
		if err != nil {
			fmt.Println(err)
		}
		iter := fileDB.NewIterator(nil, nil)
		for iter.Next() {
			key := iter.Key()
			fmt.Println(string(key))
		}
	*/
	_ = cluster
	_ = leaderId
}

func expect(t *testing.T, ci sm.CommitInfo, str string, id int32) {
	//fmt.Println("~~~~>", id)
	cnt++
	if ci.Err != nil {
		t.Fatal(ci.Err)
	}
	if string(ci.Data) != str {
		t.Fatal(id, "Got different data", str, " - ", string(ci.Data))
	}
}

func cleanDB() {
	os.RemoveAll("./log")
}

func (myRaft Raft) GetMockLeader(id []int, len int) int {
	for {
		fmt.Print("")
		for i := 0; i < len; i++ {
			if myRaft.Cluster[id[i]].SM.Status == LEAD {
				return id[i]
			}
		}
	}
	return -1
}

func (myRaft Raft) makeRafts() Raft {
	//myRaft.CommitInfo = make(chan interface{})
	for id := 1; id <= PEERS; id++ {
		//fmt.Println(id)
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
	//myRaft.CommitInfo = make(chan interface{})
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
