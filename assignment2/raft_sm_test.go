package main

import (
	"fmt"
	"reflect"
	"testing"
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
	//CandTesting(t)
}

//Testing various scenarios against Follower state.
func FollTesting(t *testing.T) {

	//Creating a follower which has just joined the cluster.
	sm := State_Machine{Persi_State: Persi_State{id: 1000, status: FOLL, currTerm: 1, logInd: 0}, Volat_State: Volat_State{commitIndex: 0, lastApplied: 0}}
	var follTC TestCases
	follTC.t = t
	var cmdReq = []string{"read test", "read cs733"}

	//<<<|id:1000|status:follower|currTerm:2|logIng:1|votedFor:0|commitInd:0|>>>

	/*Sending an apped request*/
	follTC.req = Append{data: []byte(cmdReq[0])}
	follTC.respExp = Commit{data: []byte("5000"), err: []byte("I'm not leader")}
	clientCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	/*Sending appendEntry request*/
	//Suppose leader and follower are at same term.
	entries1 := Log{log: []MyLog{{1, "read test"}}}
	follTC.req = AppEntrReq{term: 1, leaderId: 5000, preLogInd: 0, preLogTerm: 2, log: entries1, leaderCom: 0}
	follTC.respExp = AppEntrResp{term: 1, succ: true}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//<<<|id:1000|status:follower|currTerm:1|logInd:1|votedFor:0|commitInd:0|lastApp:0|>>>

	//Sending Multiple entries
	entries2 := Log{log: []MyLog{{1, "read test"}, {1, "read cloud"}, {1, "read cs733"}}}
	follTC.req = AppEntrReq{term: 1, leaderId: 5000, preLogInd: 0, preLogTerm: 1, log: entries2, leaderCom: 1}
	follTC.respExp = AppEntrResp{term: 1, succ: true}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//<<<|id:1000|status:follower|currTerm:1|logInd:3|votedFor:0|commitInd:1|lastApp:0|>>>

	//Suppose leader has higher term than follower.
	entries3 := Log{log: []MyLog{{1, "read cs733"}, {2, "delete test"}}}
	follTC.req = AppEntrReq{term: 2, leaderId: 5000, preLogInd: 2, preLogTerm: 1, log: entries3, leaderCom: 2}
	follTC.respExp = AppEntrResp{term: 2, succ: true}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//<<<|id:1000|status:follower|currTerm:2|logIng:4|votedFor:0|commitInd:2|lastApp:0|>>>

	//Suppose follower has higher term than leader.
	entries4 := Log{log: []MyLog{{1, "read cs733"}, {2, "delete test"}}}
	follTC.req = AppEntrReq{term: 1, leaderId: 5000, preLogInd: 2, preLogTerm: 2, log: entries4, leaderCom: 2}
	follTC.respExp = AppEntrResp{term: 2, succ: false}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//Suppose prevIndex does not matches.
	entries4 = Log{log: []MyLog{{3, "append test"}, {3, "cas test"}}}
	follTC.req = AppEntrReq{term: 3, leaderId: 5000, preLogInd: 5, preLogTerm: 3, log: entries4, leaderCom: 2}
	follTC.respExp = AppEntrResp{term: 2, succ: false}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//Suppose prevIndex matches, but entry doesnt.
	entries4 = Log{log: []MyLog{{3, "append test"}, {3, "cas test"}}}
	follTC.req = AppEntrReq{term: 3, leaderId: 5000, preLogInd: 3, preLogTerm: 3, log: entries4, leaderCom: 2}
	follTC.respExp = AppEntrResp{term: 2, succ: false}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//Everything is ok, but no entry.
	follTC.req = AppEntrReq{term: 3, leaderId: 5000, preLogInd: 3, preLogTerm: 3, leaderCom: 2}
	follTC.respExp = Alarm{t: 5}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	/*Sending vote request*/

	//sending vote request with lower term.
	follTC.req = VoteReq{term: 1, candId: 2000, preLogInd: 1, preLogTerm: 1}
	follTC.respExp = VoteResp{term: 2, voteGrant: false}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//sending vote request with not updated log.
	follTC.req = VoteReq{term: 3, candId: 2000, preLogInd: 1, preLogTerm: 1}
	follTC.respExp = VoteResp{term: 2, voteGrant: false}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//sending vote request with not updated log.
	follTC.req = VoteReq{term: 3, candId: 2000, preLogInd: 5, preLogTerm: 4}
	follTC.respExp = VoteResp{term: 2, voteGrant: true}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//<<<|id:1000|status:follower|currTerm:3|logIng:4|votedFor:1|commitInd:2|lastApp:0|>>>

	//checking against duplicate vote rquest
	follTC.req = VoteReq{term: 3, candId: 2000, preLogInd: 5, preLogTerm: 4}
	follTC.respExp = VoteResp{term: 2, voteGrant: false}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	/*Sending timeout*/
	follTC.req = Timeout{}
	follTC.respExp = Alarm{t: 5}
	timeoutCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh //expecting alarm signal
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = VoteReq{term: 3, candId: 1000} //also the vote request
	follTC.expect()

}

//Testing various scenarios against Candidate state.
func CandTesting(t *testing.T) {
	//Creating a Candidate.
	sm := State_Machine{Persi_State: Persi_State{id: 1001, status: CAND, currTerm: 3, logInd: 4}, Volat_State: Volat_State{commitIndex: 2, lastApplied: 0}}
	var follTC TestCases
	follTC.t = t
	var cmdReq = []string{"read test", "read cs733"}

	//<<<|id:1000|status:follower|currTerm:3|logIng:4|votedFor:1|commitInd:2|lastApp:0|>>>

	/* Sending an apped request */
	follTC.req = Append{data: []byte(cmdReq[0])}
	follTC.respExp = Commit{data: []byte("5000"), err: []byte("I'm not leader")}
	clientCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	/* Sending vote request */
	//sending vote request.
	follTC.req = VoteReq{term: 1, candId: 2000, preLogInd: 1, preLogTerm: 1}
	follTC.respExp = VoteResp{term: 3, voteGrant: false}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

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
		tc.t.Error(fmt.Sprintf("\nRequested: ", tc.req, "\nExpected: ", tc.respExp, "\nFound: ", tc.resp))
	}
}
