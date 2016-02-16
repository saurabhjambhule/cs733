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
	follObj := Follower{Persi_State: Persi_State{id: 1000, currTerm: 2}, Volat_State: Volat_State{status: FOLL, commitIndex: 0, lastApplied: 0, timer: 1500, logInd: 0}}
	sm = follObj
	go follSys(sm)

	var follTC TestCases
	var cmdReq = []string{"read test", "read cs733"}

	//Sending an apped request//
	follTC.req = Append{data: []byte(cmdReq[0])}
	follTC.respExp = Commit{data: []byte("5000"), err: []byte("I'm not leader")}
	follTC.t = t
	clientCh <- follTC.req
	follTC.resp = <-actionCh
	follTC.expect()

	//Sending appendEntry request//
	//Suppose leader and follower are at same term.
	entries := Log{log: []MyLog{{2, "read test"}}}
	follTC.req = AppEntrReq{term: 2, leaderId: 5000, preLogInd: 0, preLogTerm: 2, log: entries, leaderCom: 0}
	follTC.respExp = AppEntrResp{term: 2, succ: true}
	follTC.t = t
	netCh <- follTC.req
	follTC.resp = <-actionCh
	follTC.expect()

	//Sending Multiple entries
	entries = Log{log: []MyLog{{2, "read test"}, {2, "read cs733"}}}
	follTC.req = AppEntrReq{term: 2, leaderId: 5000, preLogInd: 0, preLogTerm: 2, log: entries, leaderCom: 0}
	follTC.respExp = AppEntrResp{term: 2, succ: true}
	follTC.t = t
	netCh <- follTC.req
	follTC.resp = <-actionCh
	follTC.expect()

	//Suppose follower has greater term than leader.
	entries = Log{log: []MyLog{{1, "read test"}}}
	follTC.req = AppEntrReq{term: 1, leaderId: 5000, preLogInd: 0, preLogTerm: 1, log: entries, leaderCom: 0}
	follTC.respExp = AppEntrResp{term: 2, succ: false}
	follTC.t = t
	netCh <- follTC.req
	follTC.resp = <-actionCh
	follTC.expect()

	//Suppose leader and follower are at same term, but previoud log index do not match.
	entries = Log{log: []MyLog{{2, "read test"}, {2, "read cs733"}}}
	follTC.req = AppEntrReq{term: 2, leaderId: 5000, preLogInd: 1, preLogTerm: 2, log: entries, leaderCom: 0}
	follTC.respExp = AppEntrResp{term: 2, succ: false}
	follTC.t = t
	netCh <- follTC.req
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

func enterLog(mylog MyLog) Log {
	var log1 Log
	log1.log = append(log1.log, mylog)
	return log1

}
func (tc TestCases) expect() {
	if !reflect.DeepEqual(tc.resp, tc.respExp) {
		tc.t.Error(fmt.Sprintf("Expected: ", tc.resp, "Found: ", tc.respExp))
	}
}
