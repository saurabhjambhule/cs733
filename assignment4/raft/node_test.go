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

func TestBasic(t *testing.T) {
	//Initialization.
	var myRaft Raft
	partCl0 := make([]int, 0)
	partCl1 := make([]int, 0)
	partCl2 := make([]int, 0)
	partCl0 = append(partCl0, 1, 2, 3, 4, 5)

	cluster := new(mock.MockCluster)
	//cluster = new(mock.MockCluster)
	cleanDB()                                //Clear the old database.
	myRaft, cluster = myRaft.makeMockRafts() //make mock cluster
	//Simple cluster can also be created using myraft.makeRafts() method.

	leaderId := myRaft.GetLeader() //Get current leader.

	fmt.Println("Basic Replication-", leaderId)
	//Appending Entries And check for replication of entry.
	for j := 1; j <= 5; j++ {
		str := "test - " + strconv.Itoa(j)
		myRaft.Cluster[leaderId].Append([]byte(str))
	}
	//Checking whether Entries are replicated.
	/*for j := 1; j <= 5; j++ {
		for i := 0; i < PEERS; i++ {
			str := "test - " + strconv.Itoa(j)
			ci := <-myRaft.Cluster[i].SM.CommMedium.CommitCh
			tmp := ci.(sm.CommitInfo)
			expect(t, tmp, str, myRaft.Cluster[i].SM.Id)
		}-

	}*/

	//Creating partion.
	L := myRaft.GetLeader()

	//fmt.Println("\n\n\nLeader-", L)
	switch L {
	case 0, 1:
		partCl1 = append(partCl1, 1, 2)
		partCl2 = append(partCl2, 3, 4, 5)
		cluster.Partition(partCl1, partCl2)
		//time.Sleep(1 * time.Second)
		//leaderId = myRaft.GetMockLeader(partCl2)
	case 2, 3:
		partCl1 = append(partCl1, 3, 4)
		partCl2 = append(partCl2, 1, 2, 5)
		cluster.Partition(partCl1, partCl2)
		//time.Sleep(1 * time.Second)
		//leaderId = myRaft.GetMockLeader(partCl2)
	case 4:
		partCl1 = append(partCl1, 4, 5)
		partCl2 = append(partCl2, 1, 2, 3)
		cluster.Partition(partCl1, partCl2)
		//time.Sleep(1 * time.Second)
		//leaderId = myRaft.GetMockLeader(partCl2)
	}

	fmt.Println("Partition_1-", L)
	//Appending Entries And check for replication of entry.
	for j := 6; j <= 10; j++ {
		str := "test - " + strconv.Itoa(j)
		myRaft.Cluster[L].Append([]byte(str))
	}

	//Checking whether Entries are replicated.
	/*for j := 6; j <= 10; j++ {
		for i := 0; i < 2; i++ {
			str := "test - " + strconv.Itoa(j)
			ci := <-myRaft.Cluster[partCl1[i]-1].SM.CommMedium.CommitCh
			tmp := ci.(sm.CommitInfo)
			expect(t, tmp, str, myRaft.Cluster[partCl1[i]-1].SM.Id)
		}
	}*/

	//Merging the cluster back.
	cluster.Heal()
	time.Sleep(1 * time.Second)
	leaderId = myRaft.GetLeader()
	fmt.Println("Healing_1-", leaderId)
	//Appending Entries And check for replication of entry.
	//fmt.Print(myRaft.Cluster[leaderId].SM.LoggInd, "-", myRaft.Cluster[leaderId].SM.Status, ":", myRaft.Cluster[leaderId].SM.NextIndex, "\n")

	for j := 11; j <= 15; j++ {
		str := "test - " + strconv.Itoa(j)
		myRaft.Cluster[leaderId].Append([]byte(str))
	}
	//Checking whether Entries are replicated.
	/*for i := 0; i < PEERS; i++ {
		for j := 11; j <= 15; j++ {
			str := "test - " + strconv.Itoa(j)
			ci := <-myRaft.Cluster[i].SM.CommMedium.CommitCh
			tmp := ci.(sm.CommitInfo)
			expect(t, tmp, str, myRaft.Cluster[i].SM.Id)
		}

	}*/
	//fmt.Print(myRaft.Cluster[leaderId].SM.LoggInd, "-", myRaft.Cluster[leaderId].SM.Status, ":", myRaft.Cluster[leaderId].SM.NextIndex, "\n")

	//printDB(myRaft, 15)

	//Creating partion.
	L = myRaft.GetLeader()
	switch L {
	case 0, 1:
		cluster.Partition(partCl1, partCl2)
		//time.Sleep(1 * time.Second)
		leaderId = myRaft.GetMockLeader(partCl2)
	case 2, 3:
		cluster.Partition(partCl1, partCl2)
		//time.Sleep(1 * time.Second)
		leaderId = myRaft.GetMockLeader(partCl2)
	case 4:
		cluster.Partition(partCl1, partCl2)
		//time.Sleep(1 * time.Second)
		leaderId = myRaft.GetMockLeader(partCl2)
	}

	fmt.Println("Partition_2-", leaderId)

	//fmt.Print(myRaft.Cluster[leaderId].SM.LoggInd, "-", myRaft.Cluster[leaderId].SM.Status, ":", myRaft.Cluster[leaderId].SM.NextIndex, "\n")

	//Appending Entries And check for replication of entry.
	for j := 16; j <= 20; j++ {
		str := "test - " + strconv.Itoa(j)
		myRaft.Cluster[leaderId-1].Append([]byte(str))
	}
	//Checking whether Entries are replicated.
	/*for j := 16; j <= 20; j++ {
		for i := 0; i < 3; i++ {
			str := "test - " + strconv.Itoa(j)
			ci := <-myRaft.Cluster[partCl2[i]-1].SM.CommMedium.CommitCh
			tmp := ci.(sm.CommitInfo)
			expect(t, tmp, str, myRaft.Cluster[partCl2[i]-1].SM.Id)
		}
	}*/
	//printDB(myRaft, 20)
	//for i := 0; i < 5; i++ {
	//	fmt.Println(myRaft.Cluster[i].SM.Id, ")", myRaft.Cluster[i].SM.Logg)
	//}

	//Merging the cluster back.
	cluster.Heal()
	time.Sleep(1 * time.Second)
	leaderId = myRaft.GetLeader()
	fmt.Println("Healing_2-", leaderId)

	//	for i := 0; i < 5; i++ {
	//fmt.Print(myRaft.Cluster[leaderId].SM.LoggInd, "-", myRaft.Cluster[leaderId].SM.Status, ":", myRaft.Cluster[leaderId].SM.NextIndex, "\n")
	//	}
	//fmt.Println("")
	//Appending Entries And check for replication of entry.
	for j := 21; j <= 25; j++ {
		str := "test - " + strconv.Itoa(j)
		myRaft.Cluster[leaderId].Append([]byte(str))
	}

	//time.Sleep(1 * time.Second)
	//fmt.Print(myRaft.Cluster[leaderId].SM.LoggInd, "-", myRaft.Cluster[leaderId].SM.Status, ":", myRaft.Cluster[leaderId].SM.NextIndex, "\n")

	//for i := 0; i < 5; i++ {
	//	fmt.Println(myRaft.Cluster[i].SM.Id, ")", myRaft.Cluster[i].SM.Logg)
	//}
	//printDB(myRaft, 25)

	//Checking whether Entries are replicated.

	/*for i := 0; i < PEERS; i++ {
		str := "test - " + strconv.Itoa(25)
		fmt.Println(myRaft.Cluster[i].Conf.Lg.Get(int64(20)))
		//	expectMatch(t, i, str, myRaft.Cluster[i].SM.Logg.Logg[int32(24)])
		_ = str
	}*/

	fmt.Println("Shutting Down-", leaderId)

	//Shutting down on of the server.
	time.Sleep(1 * time.Second)
	//fmt.Println("\n>>>Shutdown-", leaderId)

	myRaft.Cluster[leaderId].Shutdown(cluster)
	partCl0 = append(partCl0[:leaderId], partCl0[leaderId+1:]...)

	time.Sleep(1 * time.Second)

	leaderId = myRaft.GetLeader()
	fmt.Println("New Leader-", leaderId)

	//Appending Entries And check for replication of entry.
	for j := 26; j <= 30; j++ {
		str := "test - " + strconv.Itoa(j)
		myRaft.Cluster[leaderId].Append([]byte(str))
	}
	/*
		//Checking whether Entries are replicated.
		for i := 0; i < PEERS-1; i++ {
			for j := 26; j <= 30; j++ {
				str := "test - " + strconv.Itoa(j)
				ci := <-myRaft.Cluster[partCl0[i]-1].SM.CommMedium.CommitCh
				tmp := ci.(sm.CommitInfo)
				expect(t, tmp, str, myRaft.Cluster[partCl0[i]-1].SM.Id)
			}

		}

		/*for i := 0; i < 5; i++ {
			fmt.Println(myRaft.Cluster[i].SM.Id, ")", myRaft.Cluster[i].SM.Logg)
		}*/

	time.Sleep(5 * time.Second)
	//for {
	//	flag := false
	for i := 0; i < PEERS; i++ {
		fmt.Print(myRaft.Cluster[i].SM.CommitIndex)
		//if myRaft.Cluster[i].SM.CommitIndex == 29 {
		//			flag = true
		//	}
		//		if flag {
		//			break
	}
	//	}
	//}
	//Printing database entries of all nodes in cluster.
	printDB(myRaft, 30)
}

func expect(t *testing.T, ci sm.CommitInfo, str string, id int32) {
	//fmt.Println("~~~~>", id)
	if ci.Err != nil {
		t.Fatal(ci.Err)
	}
	if string(ci.Data) != str {
		t.Fatal(id, "Got different data", str, " - ", string(ci.Data))
	}
}

func expectMatch(t *testing.T, id int, str string, str1 string) {
	if str != str1 {
		t.Fatal(id, "Got different data", str, " - ", str1)
	}
}

func cleanDB() {
	os.RemoveAll("./log")
}

func (myRaft Raft) GetMockLeader(id []int) int {
	//fmt.Println(id)
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
