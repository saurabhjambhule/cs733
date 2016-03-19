package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cs733-iitb/cluster/mock"
)

func TestBasic(t *testing.T) {
	var myRaft Raft
	cluster := new(mock.MockCluster)
	cleanDB()
	myRaft, cluster = myRaft.makeMockRafts()
	//time.Sleep(4 * time.Second)
	//fmt.Println(myRaft)
	leaderId := myRaft.GetLeader()
	fmt.Println("Leader Id: ", leaderId)

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
	fmt.Println("data 1 inserted")

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

	time.Sleep(10 * time.Second)
	fmt.Println("partition")

	cluster.Partition([]int{2, 3, 4}, []int{1, 5}) //Cluster partitions into two.
	time.Sleep(15 * time.Second)

	//leaderId = myRaft.GetLeader()
	//fmt.Println("Leader Id: ", leaderId)

	for i := 0; i < PEERS; i++ {
		fmt.Println(">>-- ", myRaft.Cluster[i].SM.status, " -- ", myRaft.Cluster[i].SM.id, " -- ", myRaft.Cluster[i].SM.currTerm)
	}

	for i := 0; i < PEERS; i++ {
		j := myRaft.Cluster[i].Conf.lg.GetLastIndex()
		res, err := myRaft.Cluster[i].Conf.lg.Get(j)
		fmt.Println(">> ", res, err, myRaft.Cluster[i].SM.id)
	}
	fmt.Println("hael")

	cluster.Heal()
	time.Sleep(15 * time.Second)

	for i := 0; i < PEERS; i++ {
		fmt.Println(">>-- ", myRaft.Cluster[i].SM.status, " -- ", myRaft.Cluster[i].SM.id, " -- ", myRaft.Cluster[i].SM.currTerm)
	}
	time.Sleep(30 * time.Second)

	for i := 0; i < PEERS; i++ {
		fmt.Println(">>-- ", myRaft.Cluster[i].SM.status, " -- ", myRaft.Cluster[i].SM.id, " -- ", myRaft.Cluster[i].SM.currTerm)
	}

	leaderId = myRaft.GetLeader()
	fmt.Println("Leader Id: ", leaderId)
	time.Sleep(30 * time.Second)

}

func expect(t *testing.T, ci CommitInfo, str string) {
	fmt.Println("**")

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
