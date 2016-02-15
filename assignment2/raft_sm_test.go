package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

//Test cases to be tested.
//req - is the request command send to the machine.
//resp - is the respond recieved on a given request.
//respExp -  is the expected respond corrsponding to request "req".
type TestCases struct {
	req     interface{}
	resp    interface{}
	respExp interface{}
	t       *testing.T
}

//Initializing Testing.
func TestRaftSM(t *testing.T) {
	FollTesting(t)
}

//Testing various scenarios against Follower state.
func FollTesting(t *testing.T) {
	var sm State_Machine
	//Creating a follower which has just joined the cluster.
	follObj := Follower{Persi_State: Persi_State{id: 1000, currTerm: 1}, Volat_State: Volat_State{status: FOLL, commitIndex: 0, lastApplied: 0, timer: 1500}}
	sm = follObj
	go follSys(sm)

	var follTC TestCases
	var entry = []string{"read test", "read cs733"}

	//Sending an apped request//
	follTC.req = Append{data: []byte(entry[0])}
	follTC.respExp = Commit{data: []byte("5000"), err: []byte("I'm not leader")}
	follTC.t = t
	clientCh <- follTC.req
	follTC.resp = <-actionCh
	follTC.expect()

	//Sending appendEntry request//
	//Suppose leader and follower are at same term.
	follTC.req = AppEntrReq{term: 1, leaderId: 5000, preLogInd: 0, preLogTerm: 1, entries: []byte(entry[1]), leaderCom: 0}
	fmt.Println(follTC.req)
	follTC.respExp = AppEntrResp{term: 1, succ: true}
	follTC.t = t
	netCh <- follTC.req
	follTC.resp = <-actionCh
	follTC.expect()

	follTC.req = Append{data: []byte(entry[1])}
	follTC.respExp = Commit{data: []byte("5000"), err: []byte("I'm not leader")}
	follTC.t = t
	clientCh <- follTC.req
	follTC.resp = <-actionCh
	follTC.expect()
	//fmt.Println(string((follTC.resp).(Commit).err))
	//fmt.Printf("## %T", (follTC.resp).(Commit))
	//fmt.Printf("## %T", follTC.resp.req)

	time.Sleep(5 * time.Second)

}

//Testing various scenarios against Candidate state.
func CandTesting(t *testing.T) {

}

//Testing various scenarios against Leader state.
func LeadTesting(t *testing.T) {

}

func (tc TestCases) expect() {
	fmt.Println("**")
	if !reflect.DeepEqual(tc.resp, tc.respExp) {
		tc.t.Error(fmt.Sprintf("Expected: ", tc.resp, "Found: ", tc.respExp))
	}
}
