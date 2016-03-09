package assignment2

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
	//Initializing peers.
	peer = make(map[int32]int32)
	peer[1000] = 0
	peer[1001] = 1
	peer[2000] = 2
	peer[3000] = 3
	peer[5000] = 4
	//Initializing state machine as Follower.
	sm := State_Machine{Persi_State: Persi_State{id: 1000, status: FOLL, currTerm: 1, logInd: 0}, Volat_State: Volat_State{commitIndex: 0, lastApplied: 0}}
	//Follower state testing.
	sm.FollTesting(t)
	//Candidate state testing.
	sm.CandTesting(t)
	//creating multiple copies of state machime in candidate state for testing purpose.
	sm1 := sm
	sm2 := sm
	sm3 := sm
	//Testing leader with timeout, append and vote requests.
	sm.LeadTesting(t)
	//Testing leader with append entry request.
	sm1.LeadTesting1(t)
	//Testing leader with append entry response.
	sm2.LeadTesting2(t)
	//Testing leader for committing log.
	sm3.LeadTesting3(t)
}

//Testing various scenarios against Follower state.
func (sm *State_Machine) FollTesting(t *testing.T) {

	//Creating a follower which has just joined the cluster.
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
	follTC.respExp = Send{peerId: 5000, event: AppEntrResp{term: 1, succ: true}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh //expecting alarm signal
	follTC.respExp = Alarm{t: 200}
	follTC.expect()

	//<<<|id:1000|status:follower|currTerm:1|logInd:1|votedFor:0|commitInd:0|lastApp:0|>>>

	//Sending Multiple entries
	entries2 := Log{log: []MyLog{{1, "read test"}, {1, "read cloud"}, {1, "read cs733"}}}
	follTC.req = AppEntrReq{term: 1, leaderId: 5000, preLogInd: 0, preLogTerm: 1, log: entries2, leaderCom: 1}
	follTC.respExp = Send{peerId: 5000, event: AppEntrResp{term: 1, succ: true}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh //expecting alarm signal
	follTC.respExp = Alarm{t: 200}
	follTC.expect()

	//<<<|id:1000|status:follower|currTerm:1|logInd:3|votedFor:0|commitInd:1|lastApp:0|>>>

	//Suppose leader has higher term than follower.
	entries3 := Log{log: []MyLog{{1, "read cs733"}, {2, "delete test"}}}
	follTC.req = AppEntrReq{term: 2, leaderId: 5000, preLogInd: 2, preLogTerm: 1, log: entries3, leaderCom: 2}
	follTC.respExp = Send{peerId: 5000, event: AppEntrResp{term: 2, succ: true}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh //expecting alarm signal
	follTC.respExp = Alarm{t: 200}
	follTC.expect()

	//<<<|id:1000|status:follower|currTerm:2|logIng:4|votedFor:0|commitInd:2|lastApp:0|>>>

	//Suppose follower has higher term than leader.
	entries4 := Log{log: []MyLog{{1, "read cs733"}, {2, "delete test"}}}
	follTC.req = AppEntrReq{term: 1, leaderId: 5000, preLogInd: 2, preLogTerm: 2, log: entries4, leaderCom: 2}
	follTC.respExp = Send{peerId: 5000, event: AppEntrResp{term: 2, succ: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh //expecting alarm signal
	follTC.respExp = Alarm{t: 200}
	follTC.expect()

	//Suppose prevIndex does not matches.
	entries4 = Log{log: []MyLog{{3, "append test"}, {3, "cas test"}}}
	follTC.req = AppEntrReq{term: 3, leaderId: 5000, preLogInd: 5, preLogTerm: 3, log: entries4, leaderCom: 2}
	follTC.respExp = Send{peerId: 5000, event: AppEntrResp{term: 2, succ: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh //expecting alarm signal
	follTC.respExp = Alarm{t: 200}
	follTC.expect()

	//Suppose prevIndex matches, but entry doesnt.
	entries4 = Log{log: []MyLog{{3, "append test"}, {3, "cas test"}}}
	follTC.req = AppEntrReq{term: 3, leaderId: 5000, preLogInd: 3, preLogTerm: 3, log: entries4, leaderCom: 2}
	follTC.respExp = Send{peerId: 5000, event: AppEntrResp{term: 2, succ: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh //expecting alarm signal
	follTC.respExp = Alarm{t: 200}
	follTC.expect()

	//Everything is ok, but no entry.
	follTC.req = AppEntrReq{term: 3, leaderId: 5000, preLogInd: 3, preLogTerm: 3, leaderCom: 2}
	follTC.respExp = Alarm{t: 200}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	/*Sending vote request*/
	//sending vote request with lower term.
	follTC.req = VoteReq{term: 1, candId: 2000, preLogInd: 1, preLogTerm: 1}
	follTC.respExp = Send{peerId: 2000, event: VoteResp{term: 2, voteGrant: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//sending vote request with not updated log.
	follTC.req = VoteReq{term: 3, candId: 2000, preLogInd: 1, preLogTerm: 1}
	follTC.respExp = Send{peerId: 2000, event: VoteResp{term: 2, voteGrant: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//sending vote request with not updated log.
	follTC.req = VoteReq{term: 3, candId: 2000, preLogInd: 5, preLogTerm: 4}
	follTC.respExp = Send{peerId: 2000, event: VoteResp{term: 3, voteGrant: true}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//<<<|id:1000|status:follower|currTerm:3|logIng:4|votedFor:1|commitInd:2|lastApp:0|>>>

	//checking against duplicate vote rquest
	follTC.req = VoteReq{term: 3, candId: 2000, preLogInd: 5, preLogTerm: 4}
	follTC.respExp = Send{peerId: 2000, event: VoteResp{term: 3, voteGrant: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	/*Sending timeout*/
	follTC.req = Timeout{}
	follTC.respExp = Alarm{t: 150}
	timeoutCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh //expecting alarm signal
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Send{peerId: 0, event: VoteReq{term: 4, candId: 1000, preLogInd: 3, preLogTerm: 2}} //also the vote request
	follTC.expect()

}

//Testing various scenarios against Candidate state.
func (sm *State_Machine) CandTesting(t *testing.T) {
	//Creating a Candidate.

	//sm := State_Machine{Persi_State: Persi_State{id: 1001, status: CAND, currTerm: 3, logInd: 4}, Volat_State: Volat_State{commitIndex: 2, lastApplied: 0}}
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

	/* Sending vote request*/
	//sending vote request with lower term.
	follTC.req = VoteReq{term: 1, candId: 2000, preLogInd: 1, preLogTerm: 1}
	follTC.respExp = Send{peerId: 2000, event: VoteResp{term: 4, voteGrant: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//sending append request with lower term
	//-->Expected Candidate to reject append request response.
	entries := Log{log: []MyLog{{1, "read test"}}}
	follTC.req = AppEntrReq{term: 1, leaderId: 5000, preLogInd: 1, preLogTerm: 2, log: entries, leaderCom: 2}
	follTC.respExp = Send{peerId: 5000, event: AppEntrResp{term: 4, succ: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//sending append request with higher term
	//-->Expected Candidate to become Follower And send Alarm as well as append request response as negative as higher index.
	follTC.req = AppEntrReq{term: 4, leaderId: 5000, preLogInd: 5, preLogTerm: 2, log: entries, leaderCom: 2}
	follTC.respExp = Send{peerId: 5000, event: AppEntrResp{term: 4, succ: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Alarm{t: 200}
	follTC.expect()

	/*Sending timeout*/
	//Sending timeout to follower.
	//-->Expected Follower to become Candidate and send Alarm as well as Vote request.
	follTC.req = Timeout{}
	follTC.respExp = Alarm{t: 150}
	timeoutCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh //expecting alarm signal
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Send{peerId: 0, event: VoteReq{term: 5, candId: 1000, preLogInd: 3, preLogTerm: 2}} //also the vote request
	follTC.expect()

	//Sending timeout to candidate
	//-->Expected Candidate to do re-election, results sending Alarm and vote requets and Increase in term.
	follTC.req = Timeout{}
	follTC.respExp = Alarm{t: 150}
	timeoutCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh //expecting alarm signal
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Send{peerId: 0, event: VoteReq{term: 6, candId: 1000, preLogInd: 3, preLogTerm: 2}} //also the vote request
	follTC.expect()

	/* Sending vote response*/
	//sending possitive vote response.
	//-->Expected cadidate to become leader and send heartbeat msg to other servers.
	follTC.req = VoteResp{term: 4, voteGrant: true}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.req = VoteResp{term: 3, voteGrant: true}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.req = VoteResp{term: 2, voteGrant: true}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.respExp = Alarm{t: 175}
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Send{peerId: 0, event: AppEntrReq{term: 6, leaderId: 1000, preLogInd: 3, preLogTerm: 2, leaderCom: 2}}
	follTC.expect()

	//<<<|id:1000|status:follower|currTerm:3|logInd:4|votedFor:1|commitInd:2|lastApp:0|>>>

	//sending negative vote response.
	sm1 := State_Machine{Persi_State: Persi_State{id: 1001, status: CAND, currTerm: 3, logInd: 4}, Volat_State: Volat_State{commitIndex: 2, lastApplied: 0}}
	follTC.req = VoteResp{term: 2, voteGrant: false}
	netCh <- follTC.req
	sm1.eventProcess()
	follTC.req = VoteResp{term: 3, voteGrant: false}
	netCh <- follTC.req
	sm1.eventProcess()
	follTC.req = VoteResp{term: 2, voteGrant: false}
	netCh <- follTC.req
	sm1.eventProcess()
	follTC.respExp = Alarm{t: 200}
	follTC.resp = <-actionCh
	follTC.expect()

	//sending vote response with higher term.
	sm3 := State_Machine{Persi_State: Persi_State{id: 1000, status: CAND, currTerm: 3, logInd: 4}, Volat_State: Volat_State{commitIndex: 2, lastApplied: 0}}
	follTC.req = VoteResp{term: 5, voteGrant: false}
	netCh <- follTC.req
	sm3.eventProcess()
	netCh <- follTC.req
	sm1.eventProcess()
	follTC.respExp = Alarm{t: 200}
	follTC.resp = <-actionCh
	follTC.expect()

	//Suppose due to cluster partition cadidate gets two positive and two negative responses.
	//-->Expect to start re-election.
	sm2 := State_Machine{Persi_State: Persi_State{id: 1000, status: CAND, currTerm: 3, logInd: 4}, Volat_State: Volat_State{commitIndex: 2, lastApplied: 0}}
	entry := Log{log: []MyLog{{1, "read test"}, {1, "read cloud"}, {1, "read cs733"}, {2, "delete test"}}}
	sm2.log = entry
	follTC.req = VoteResp{term: 2, voteGrant: true}
	netCh <- follTC.req
	sm2.eventProcess()
	follTC.req = VoteResp{term: 2, voteGrant: false}
	netCh <- follTC.req
	sm2.eventProcess()
	follTC.req = VoteResp{term: 3, voteGrant: true}
	netCh <- follTC.req
	sm2.eventProcess()
	follTC.req = VoteResp{term: 3, voteGrant: false}
	netCh <- follTC.req
	sm2.eventProcess()
	follTC.resp = <-actionCh
	follTC.respExp = Alarm{t: 150} //expecting alarm signal
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Send{peerId: 0, event: VoteReq{term: 4, candId: 1000, preLogInd: 3, preLogTerm: 2}} //also the vote request
	follTC.expect()

	/*Testing a follower  with no log entry became candidate and sends vote request*/
	sm5 := State_Machine{Persi_State: Persi_State{id: 3000, status: FOLL, currTerm: 1, logInd: 0}, Volat_State: Volat_State{commitIndex: 0, lastApplied: 0}}
	//sending vote request from very new candidate to very new follower.
	follTC.req = VoteReq{term: 2, candId: 3000, preLogInd: 0, preLogTerm: 0}
	follTC.respExp = Send{peerId: 3000, event: VoteResp{term: 2, voteGrant: true}}
	netCh <- follTC.req
	sm5.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//Sending timeout to follower, SM can become candidate
	follTC.req = Timeout{}
	follTC.respExp = Alarm{t: 150}
	timeoutCh <- follTC.req
	sm5.eventProcess()
	follTC.resp = <-actionCh //expecting alarm signal
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Send{peerId: 0, event: VoteReq{term: 3, candId: 3000, preLogInd: 0, preLogTerm: 0}} //also the vote request
	follTC.expect()
}

//Testing various scenarios against Leader state.
func (sm *State_Machine) LeadTesting(t *testing.T) {
	var follTC TestCases
	follTC.t = t
	var cmdReq = []string{"rename test"}

	//<<<|id:1000|status:leader|currTerm:6|logInd:4|votedFor:0|commitInd:0|>>>

	/*Sending timeout*/
	//-->Expected to send heartbeat msg to all server.
	follTC.req = Timeout{}
	follTC.respExp = Send{peerId: 0, event: AppEntrReq{term: 6, leaderId: 1000, preLogInd: 3, preLogTerm: 2, leaderCom: 2}}
	timeoutCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	/* Sending an append request*/
	//-->Expeced LogStore msg and Appendentry request to all servers containg current and previous entry.
	entry := Log{log: []MyLog{sm.log.log[sm.logInd-1], {6, "rename test"}}}
	follTC.req = Append{data: []byte(cmdReq[0])}
	follTC.respExp = LogStore{index: 4, data: []byte(cmdReq[0])}
	clientCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Send{peerId: 0, event: AppEntrReq{term: 6, leaderId: 1000, preLogInd: 4, preLogTerm: 6, leaderCom: 2, log: entry}}
	follTC.expect()

	/* Sending vote request*/
	//sending vote request with lower term.
	follTC.req = VoteReq{term: 4, candId: 2000, preLogInd: 1, preLogTerm: 1}
	follTC.respExp = Send{peerId: 2000, event: VoteResp{term: 6, voteGrant: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//sending vote request with higher term.
	//-->Expected to step down to Follower and as follower send Alarm signal.
	follTC.req = VoteReq{term: 8, candId: 2000, preLogInd: 3, preLogTerm: 2}
	follTC.respExp = Send{peerId: 2000, event: VoteResp{term: 6, voteGrant: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Alarm{t: 200}
	follTC.expect()
}

func (sm *State_Machine) LeadTesting1(t *testing.T) {
	var follTC TestCases
	follTC.t = t
	var cmdReq = []string{"rename test"}

	//<<<|id:1000|status:leader|currTerm:6|logInd:5|votedFor:0|commitInd:0|>>>

	/* Sending an append request*/
	entry := Log{log: []MyLog{sm.log.log[sm.logInd-1], {6, "rename test"}}}
	follTC.req = Append{data: []byte(cmdReq[0])}
	follTC.respExp = LogStore{index: 4, data: []byte(cmdReq[0])}
	clientCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Send{peerId: 0, event: AppEntrReq{term: 6, leaderId: 1000, preLogInd: 4, preLogTerm: 6, leaderCom: 2, log: entry}}
	follTC.expect()

	/* Sending an append entry request*/
	//sending append request with lower term
	//-->Expected Candidate to reject append request response.
	entries := Log{log: []MyLog{{1, "read test"}}}
	follTC.req = AppEntrReq{term: 4, leaderId: 5000, preLogInd: 0, preLogTerm: 2, log: entries, leaderCom: 1}
	follTC.respExp = Send{peerId: 5000, event: AppEntrResp{term: 6, succ: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//sending append request with higher term
	//-->Expected Leader to become Follower And send Alarm msg from follower.
	follTC.req = AppEntrReq{term: 7, leaderId: 5000, preLogInd: 4, preLogTerm: 2, log: entries, leaderCom: 1}
	follTC.respExp = Send{peerId: 5000, event: AppEntrResp{term: 6, succ: false}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Alarm{t: 200}
	follTC.expect()
}

func (sm *State_Machine) LeadTesting2(t *testing.T) {
	var follTC TestCases
	follTC.t = t
	var cmdReq = []string{"rename test"}

	//<<<|id:1000|status:leader|currTerm:6|logInd:5|votedFor:0|commitInd:0|>>>

	/* Sending an append request*/
	entry := Log{log: []MyLog{sm.log.log[sm.logInd-1], {6, "rename test"}}}
	follTC.req = Append{data: []byte(cmdReq[0])}
	follTC.respExp = LogStore{index: 4, data: []byte(cmdReq[0])}
	clientCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
	follTC.resp = <-actionCh
	follTC.respExp = Send{peerId: 0, event: AppEntrReq{term: 6, leaderId: 1000, preLogInd: 4, preLogTerm: 6, leaderCom: 2, log: entry}}
	follTC.expect()

	/*Sending append entry response*/
	//sending positive append entry response with lower term.
	follTC.req = AppEntrResp{peer: 1000, term: 6, succ: true}
	netCh <- follTC.req
	sm.eventProcess()

	//sending negative append entry response with lower term.
	temp := sm.nextIndex[peer[2000]] - 2
	entry2 := sm.log.log[temp:]
	entry1 := Log{log: entry2}
	follTC.req = AppEntrResp{peer: 2000, term: 6, succ: false}
	follTC.respExp = Send{peerId: 2000, event: AppEntrReq{term: 6, leaderId: 1000, preLogInd: 2, preLogTerm: 1, leaderCom: 2, log: entry1}}
	netCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()
}

func (sm *State_Machine) LeadTesting3(t *testing.T) {
	var follTC TestCases
	follTC.t = t

	/*Sending new Commit entry*/
	//Setting state machine such that majority of follower had send positive append entry response.
	//<<<|id:1000|status:leader|currTerm:6|logInd:8|votedFor:0|commitInd:1|>>>
	entry := Log{log: []MyLog{{1, "read test"}, {1, "read cloud"}, {1, "read cs733"}, {2, "delete test"}, {4, "cas test"}, {4, "rename cloud"}, {6, "rename test"}, {6, "cas cs733"}}}
	sm.log = entry
	sm.logInd = 8
	sm.commitIndex = 1
	sm.currTerm = 6
	match := [5]int32{6, 6, 4, 6, 3}
	sm.matchIndex = match
	//Sending timeout
	//-->Expected to send heartbeat msg to all server with new commit index.
	follTC.req = Timeout{}
	follTC.respExp = Send{peerId: 0, event: AppEntrReq{term: 6, leaderId: 1000, preLogInd: 7, preLogTerm: 6, leaderCom: 6}}
	timeoutCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//<<<|id:1000|status:leader|currTerm:6|logInd:8|votedFor:0|commitInd:6|>>>

	//Setting state machine such that majority of follower had not send positive append entry response.
	//<<<|id:1000|status:leader|currTerm:6|logInd:8|votedFor:0|commitInd:1|>>>
	match = [5]int32{6, 2, 4, 6, 3}
	sm.commitIndex = 1
	sm.matchIndex = match
	//Sending timeout
	//-->Expected to send heartbeat msg to all server with old commit index.
	follTC.req = Timeout{}
	follTC.respExp = Send{peerId: 0, event: AppEntrReq{term: 6, leaderId: 1000, preLogInd: 7, preLogTerm: 6, leaderCom: 1}}
	timeoutCh <- follTC.req
	sm.eventProcess()
	follTC.resp = <-actionCh
	follTC.expect()

	//<<<|id:1000|status:leader|currTerm:6|logInd:8|votedFor:0|commitInd:1|>>>
}

//Coppying append command into log.
func enterLog(mylog MyLog) Log {
	var log1 Log
	log1.log = append(log1.log, mylog)
	return log1
}

//Evaluating expected and recieved event responses.
func (tc TestCases) expect() {
	if !reflect.DeepEqual(tc.resp, tc.respExp) {
		tc.t.Error(fmt.Sprintf("\nRequested: ", tc.req, "\nExpected: ", tc.respExp, "\nFound: ", tc.resp))
	}
}

//Evaluating expected and recieved  numerical responses.
func expectNum(t *testing.T, res1 int32, exp1 int32, res2 int32, exp2 int32) {
	if res1 != exp1 {
		t.Error(fmt.Sprintf("\nExpected: ", exp1, "\nFound: ", res1))
	}
	if res2 != exp2 {
		t.Error(fmt.Sprintf("\nExpected: ", exp2, "\nFound: ", res2))
	}
}
