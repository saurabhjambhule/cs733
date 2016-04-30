package sm

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

var cnt int

//Initializing Testing.
func TestCreateRaftSM(t *testing.T) {
	//Initializing Peers.
	Peer := make(map[int32]int32)
	Peer[1000] = 0
	Peer[1001] = 1
	Peer[2000] = 2
	Peer[3000] = 3
	Peer[5000] = 4
	//Initializing state machine as Follower.
	sm := State_Machine{Persi_State: Persi_State{Id: 1000, Status: FOLL, CurrTerm: 1, LoggInd: 0}, Volat_State: Volat_State{CommitIndex: 0, LastApplied: 0}}
	sm.ClientCh = make(chan interface{}, 10)
	sm.NetCh = make(chan interface{}, 10)
	sm.TimeoutCh = make(chan interface{}, 10)
	sm.ActionCh = make(chan interface{}, 10)
	//Follower state testing.
	sm.FollTesting(t)
	//CandIdate state testing.
	//sm.CandTesting(t)
	//creating multiple copies of state machime in CandIdate state for testing purpose.
	//sm1 := sm
	//sm2 := sm
	//sm3 := sm
	//Testing leader with timeout, append and vote requests.
	//sm.LeadTesting(t)
	//Testing leader with append entry request.
	//sm1.LeadTesting1(t)
	//Testing leader with append entry response.
	//sm2.LeadTesting2(t)
	//Testing leader for committing Logg.
	//sm3.LeadTesting3(t)
}

//Testing various scenarios against Follower state.
func (sm *State_Machine) FollTesting(t *testing.T) {

	//Creating a follower which has just joined the cluster.
	var follTC TestCases
	follTC.t = t
	var cmdReq = []string{"read test", "read cs733"}

	//<<<|Id:1000|Status:follower|CurrTerm:2|LoggIng:1|votedFor:0|commitInd:0|>>>

	/*Sending an apped request*/
	follTC.req = Append{Data: []byte(cmdReq[0])}
	follTC.respExp = Commit{Data: []byte("5000"), Err: []byte("I'm not leader")}
	sm.ClientCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	/*Sending appendEntry request*/
	//Suppose leader and follower are at same Term.
	entries1 := Logg{Logg: []MyLogg{{1, "read test"}}}
	follTC.req = AppEntrReq{Term: 1, LeaderId: 5000, PreLoggInd: 0, PreLoggTerm: 2, Logg: entries1, LeaderCom: 0}
	follTC.respExp = Send{PeerId: 5000, Event: AppEntrResp{Term: 1, Succ: true}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh //expecting alarm signal
	follTC.respExp = Alarm{T: 0}
	follTC.expect()
	follTC.resp = <-sm.ActionCh //expecting LogStore
	follTC.respExp = Alarm{T: 0}
	follTC.expect()

	//<<<|Id:1000|Status:follower|CurrTerm:1|LoggInd:1|votedFor:0|commitInd:0|lastApp:0|>>>

	//Sending Multiple entries
	entries2 := Logg{Logg: []MyLogg{{1, "read test"}, {1, "read cloud"}, {1, "read cs733"}}}
	follTC.req = AppEntrReq{Term: 1, LeaderId: 5000, PreLoggInd: 0, PreLoggTerm: 1, Logg: entries2, LeaderCom: 1}
	follTC.respExp = Send{PeerId: 5000, Event: AppEntrResp{Term: 1, Succ: true}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh //expecting alarm signal
	follTC.respExp = Alarm{T: 200}
	follTC.expect()

	//<<<|Id:1000|Status:follower|CurrTerm:1|LoggInd:3|votedFor:0|commitInd:1|lastApp:0|>>>

	//Suppose leader has higher Term than follower.
	entries3 := Logg{Logg: []MyLogg{{1, "read cs733"}, {2, "delete test"}}}
	follTC.req = AppEntrReq{Term: 2, LeaderId: 5000, PreLoggInd: 2, PreLoggTerm: 1, Logg: entries3, LeaderCom: 2}
	follTC.respExp = Send{PeerId: 5000, Event: AppEntrResp{Term: 2, Succ: true}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh //expecting alarm signal
	follTC.respExp = Alarm{T: 200}
	follTC.expect()

	//<<<|Id:1000|Status:follower|CurrTerm:2|LoggIng:4|votedFor:0|commitInd:2|lastApp:0|>>>

	//Suppose follower has higher Term than leader.
	entries4 := Logg{Logg: []MyLogg{{1, "read cs733"}, {2, "delete test"}}}
	follTC.req = AppEntrReq{Term: 1, LeaderId: 5000, PreLoggInd: 2, PreLoggTerm: 2, Logg: entries4, LeaderCom: 2}
	follTC.respExp = Send{PeerId: 5000, Event: AppEntrResp{Term: 2, Succ: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh //expecting alarm signal
	follTC.respExp = Alarm{T: 200}
	follTC.expect()

	//Suppose prevIndex does not matches.
	entries4 = Logg{Logg: []MyLogg{{3, "append test"}, {3, "cas test"}}}
	follTC.req = AppEntrReq{Term: 3, LeaderId: 5000, PreLoggInd: 5, PreLoggTerm: 3, Logg: entries4, LeaderCom: 2}
	follTC.respExp = Send{PeerId: 5000, Event: AppEntrResp{Term: 2, Succ: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh //expecting alarm signal
	follTC.respExp = Alarm{T: 200}
	follTC.expect()

	//Suppose prevIndex matches, but entry doesnt.
	entries4 = Logg{Logg: []MyLogg{{3, "append test"}, {3, "cas test"}}}
	follTC.req = AppEntrReq{Term: 3, LeaderId: 5000, PreLoggInd: 3, PreLoggTerm: 3, Logg: entries4, LeaderCom: 2}
	follTC.respExp = Send{PeerId: 5000, Event: AppEntrResp{Term: 2, Succ: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh //expecting alarm signal
	follTC.respExp = Alarm{T: 200}
	follTC.expect()

	//Everything is ok, but no entry.
	follTC.req = AppEntrReq{Term: 3, LeaderId: 5000, PreLoggInd: 3, PreLoggTerm: 3, LeaderCom: 2}
	follTC.respExp = Alarm{T: 200}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	/*Sending vote request*/
	//sending vote request with lower Term.
	follTC.req = VoteReq{Term: 1, CandId: 2000, PreLoggInd: 1, PreLoggTerm: 1}
	follTC.respExp = Send{PeerId: 2000, Event: VoteResp{Term: 2, VoteGrant: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//sending vote request with not updated Logg.
	follTC.req = VoteReq{Term: 3, CandId: 2000, PreLoggInd: 1, PreLoggTerm: 1}
	follTC.respExp = Send{PeerId: 2000, Event: VoteResp{Term: 2, VoteGrant: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//sending vote request with not updated Logg.
	follTC.req = VoteReq{Term: 3, CandId: 2000, PreLoggInd: 5, PreLoggTerm: 4}
	follTC.respExp = Send{PeerId: 2000, Event: VoteResp{Term: 3, VoteGrant: true}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//<<<|Id:1000|Status:follower|CurrTerm:3|LoggIng:4|votedFor:1|commitInd:2|lastApp:0|>>>

	//checking against duplicate vote rquest
	follTC.req = VoteReq{Term: 3, CandId: 2000, PreLoggInd: 5, PreLoggTerm: 4}
	follTC.respExp = Send{PeerId: 2000, Event: VoteResp{Term: 3, VoteGrant: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	/*Sending timeout*/
	follTC.req = Timeout{}
	follTC.respExp = Alarm{T: 150}
	sm.TimeoutCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh //expecting alarm signal
	follTC.expect()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Send{PeerId: 0, Event: VoteReq{Term: 4, CandId: 1000, PreLoggInd: 3, PreLoggTerm: 2}} //also the vote request
	follTC.expect()

}

//Testing various scenarios against CandIdate state.
func (sm *State_Machine) CandTesting(t *testing.T) {
	//Creating a CandIdate.

	//sm := State_Machine{Persi_State: Persi_State{Id: 1001, Status: CAND, CurrTerm: 3, LoggInd: 4}, Volat_State: Volat_State{CommitIndex: 2, LastApplied: 0}}
	var follTC TestCases
	follTC.t = t
	var cmdReq = []string{"read test", "read cs733"}

	//<<<|Id:1000|Status:follower|CurrTerm:3|LoggIng:4|votedFor:1|commitInd:2|lastApp:0|>>>

	/* Sending an apped request */
	follTC.req = Append{Data: []byte(cmdReq[0])}
	follTC.respExp = Commit{Data: []byte("5000"), Err: []byte("I'm not leader")}
	sm.ClientCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	/* Sending vote request*/
	//sending vote request with lower Term.
	follTC.req = VoteReq{Term: 1, CandId: 2000, PreLoggInd: 1, PreLoggTerm: 1}
	follTC.respExp = Send{PeerId: 2000, Event: VoteResp{Term: 4, VoteGrant: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//sending append request with lower Term
	//-->Expected CandIdate to reject append request response.
	entries := Logg{Logg: []MyLogg{{1, "read test"}}}
	follTC.req = AppEntrReq{Term: 1, LeaderId: 5000, PreLoggInd: 1, PreLoggTerm: 2, Logg: entries, LeaderCom: 2}
	follTC.respExp = Send{PeerId: 5000, Event: AppEntrResp{Term: 4, Succ: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//sending append request with higher Term
	//-->Expected CandIdate to become Follower And send Alarm as well as append request response as negative as higher Index.
	follTC.req = AppEntrReq{Term: 4, LeaderId: 5000, PreLoggInd: 5, PreLoggTerm: 2, Logg: entries, LeaderCom: 2}
	follTC.respExp = Send{PeerId: 5000, Event: AppEntrResp{Term: 4, Succ: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Alarm{T: 200}
	follTC.expect()

	/*Sending timeout*/
	//Sending timeout to follower.
	//-->Expected Follower to become CandIdate and send Alarm as well as Vote request.
	follTC.req = Timeout{}
	follTC.respExp = Alarm{T: 150}
	sm.TimeoutCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh //expecting alarm signal
	follTC.expect()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Send{PeerId: 0, Event: VoteReq{Term: 5, CandId: 1000, PreLoggInd: 3, PreLoggTerm: 2}} //also the vote request
	follTC.expect()

	//Sending timeout to CandIdate
	//-->Expected CandIdate to do re-election, results sending Alarm and vote requets and Increase in Term.
	follTC.req = Timeout{}
	follTC.respExp = Alarm{T: 150}
	sm.TimeoutCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh //expecting alarm signal
	follTC.expect()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Send{PeerId: 0, Event: VoteReq{Term: 6, CandId: 1000, PreLoggInd: 3, PreLoggTerm: 2}} //also the vote request
	follTC.expect()

	/* Sending vote response*/
	//sending possitive vote response.
	//-->Expected cadIdate to become leader and send heartbeat msg to other servers.

	fmt.Println("~~~~~~~~")
	follTC.req = VoteResp{Term: 4, VoteGrant: true}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.req = VoteResp{Term: 3, VoteGrant: true}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	//follTC.req = VoteResp{Term: 2, VoteGrant: true}
	//sm.NetCh <- follTC.req
	//sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Alarm{T: 0}
	follTC.expect()

	follTC.resp = <-sm.ActionCh
	follTC.respExp = Alarm{T: 1}
	follTC.expect()

	follTC.respExp = Send{PeerId: 0, Event: AppEntrReq{Term: 6, LeaderId: 1000, PreLoggInd: 3, PreLoggTerm: 2, LeaderCom: 2}}
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	fmt.Println(follTC.resp)

	follTC.resp = <-sm.ActionCh
	fmt.Println(follTC.resp)

	//<<<|Id:1000|Status:follower|CurrTerm:3|LoggInd:4|votedFor:1|commitInd:2|lastApp:0|>>>
	fmt.Println("~~~~~~~~")

	//sending negative vote response.
	sm1 := State_Machine{Persi_State: Persi_State{Id: 1001, Status: CAND, CurrTerm: 3, LoggInd: 4}, Volat_State: Volat_State{CommitIndex: 2, LastApplied: 0}}
	follTC.req = VoteResp{Term: 2, VoteGrant: false}
	fmt.Println("...")

	sm.NetCh <- follTC.req
	fmt.Println("...")

	sm1.EventProcess()
	fmt.Println("...")

	follTC.req = VoteResp{Term: 3, VoteGrant: false}
	fmt.Println("...")

	sm.NetCh <- follTC.req
	sm1.EventProcess()
	follTC.req = VoteResp{Term: 2, VoteGrant: false}
	fmt.Println("...")

	sm.NetCh <- follTC.req
	sm1.EventProcess()
	follTC.respExp = Alarm{T: 200}
	fmt.Println("...")

	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//sending vote response with higher Term.
	sm3 := State_Machine{Persi_State: Persi_State{Id: 1000, Status: CAND, CurrTerm: 3, LoggInd: 4}, Volat_State: Volat_State{CommitIndex: 2, LastApplied: 0}}
	follTC.req = VoteResp{Term: 5, VoteGrant: false}
	sm.NetCh <- follTC.req
	sm3.EventProcess()
	sm.NetCh <- follTC.req
	sm1.EventProcess()
	follTC.respExp = Alarm{T: 200}
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//Suppose due to cluster partition cadIdate gets two positive and two negative responses.
	//-->Expect to start re-election.
	sm2 := State_Machine{Persi_State: Persi_State{Id: 1000, Status: CAND, CurrTerm: 3, LoggInd: 4}, Volat_State: Volat_State{CommitIndex: 2, LastApplied: 0}}
	entry := Logg{Logg: []MyLogg{{1, "read test"}, {1, "read cloud"}, {1, "read cs733"}, {2, "delete test"}}}
	sm2.Logg = entry
	follTC.req = VoteResp{Term: 2, VoteGrant: true}
	sm.NetCh <- follTC.req
	sm2.EventProcess()
	follTC.req = VoteResp{Term: 2, VoteGrant: false}
	sm.NetCh <- follTC.req
	sm2.EventProcess()
	follTC.req = VoteResp{Term: 3, VoteGrant: true}
	sm.NetCh <- follTC.req
	sm2.EventProcess()
	follTC.req = VoteResp{Term: 3, VoteGrant: false}
	sm.NetCh <- follTC.req
	sm2.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Alarm{T: 150} //expecting alarm signal
	follTC.expect()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Send{PeerId: 0, Event: VoteReq{Term: 4, CandId: 1000, PreLoggInd: 3, PreLoggTerm: 2}} //also the vote request
	follTC.expect()

	/*Testing a follower  with no Logg entry became CandIdate and sends vote request*/
	sm5 := State_Machine{Persi_State: Persi_State{Id: 3000, Status: FOLL, CurrTerm: 1, LoggInd: 0}, Volat_State: Volat_State{CommitIndex: 0, LastApplied: 0}}
	//sending vote request from very new CandIdate to very new follower.
	follTC.req = VoteReq{Term: 2, CandId: 3000, PreLoggInd: 0, PreLoggTerm: 0}
	follTC.respExp = Send{PeerId: 3000, Event: VoteResp{Term: 2, VoteGrant: true}}
	sm.NetCh <- follTC.req
	sm5.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//Sending timeout to follower, SM can become CandIdate
	follTC.req = Timeout{}
	follTC.respExp = Alarm{T: 150}
	sm.TimeoutCh <- follTC.req
	sm5.EventProcess()
	follTC.resp = <-sm.ActionCh //expecting alarm signal
	follTC.expect()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Send{PeerId: 0, Event: VoteReq{Term: 3, CandId: 3000, PreLoggInd: 0, PreLoggTerm: 0}} //also the vote request
	follTC.expect()
}

//Testing various scenarios against Leader state.
func (sm *State_Machine) LeadTesting(t *testing.T) {
	var follTC TestCases
	follTC.t = t
	var cmdReq = []string{"rename test"}

	//<<<|Id:1000|Status:leader|CurrTerm:6|LoggInd:4|votedFor:0|commitInd:0|>>>

	/*Sending timeout*/
	//-->Expected to send heartbeat msg to all server.
	follTC.req = Timeout{}
	follTC.respExp = Send{PeerId: 0, Event: AppEntrReq{Term: 6, LeaderId: 1000, PreLoggInd: 3, PreLoggTerm: 2, LeaderCom: 2}}
	sm.TimeoutCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	/* Sending an append request*/
	//-->Expeced LoggStore msg and Appendentry request to all servers containg current and previous entry.
	entr := []MyLogg{sm.Logg.Logg[sm.LoggInd-1], {6, "rename test"}}
	entry := Logg{Logg: entr}
	follTC.req = Append{Data: []byte(cmdReq[0])}
	follTC.respExp = LoggStore{Index: 4, Data: entr}
	sm.ClientCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Send{PeerId: 0, Event: AppEntrReq{Term: 6, LeaderId: 1000, PreLoggInd: 4, PreLoggTerm: 6, LeaderCom: 2, Logg: entry}}
	follTC.expect()

	/* Sending vote request*/
	//sending vote request with lower Term.
	follTC.req = VoteReq{Term: 4, CandId: 2000, PreLoggInd: 1, PreLoggTerm: 1}
	follTC.respExp = Send{PeerId: 2000, Event: VoteResp{Term: 6, VoteGrant: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//sending vote request with higher Term.
	//-->Expected to step down to Follower and as follower send Alarm signal.
	follTC.req = VoteReq{Term: 8, CandId: 2000, PreLoggInd: 3, PreLoggTerm: 2}
	follTC.respExp = Send{PeerId: 2000, Event: VoteResp{Term: 6, VoteGrant: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Alarm{T: 200}
	follTC.expect()
}

func (sm *State_Machine) LeadTesting1(t *testing.T) {
	var follTC TestCases
	follTC.t = t
	var cmdReq = []string{"rename test"}

	//<<<|Id:1000|Status:leader|CurrTerm:6|LoggInd:5|votedFor:0|commitInd:0|>>>

	/* Sending an append request*/
	entr := []MyLogg{sm.Logg.Logg[sm.LoggInd-1], {6, "rename test"}}
	entry := Logg{Logg: entr}
	follTC.req = Append{Data: []byte(cmdReq[0])}
	follTC.respExp = LoggStore{Index: 4, Data: entr}
	sm.ClientCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Send{PeerId: 0, Event: AppEntrReq{Term: 6, LeaderId: 1000, PreLoggInd: 4, PreLoggTerm: 6, LeaderCom: 2, Logg: entry}}
	follTC.expect()

	/* Sending an append entry request*/
	//sending append request with lower Term
	//-->Expected CandIdate to reject append request response.
	entries := Logg{Logg: []MyLogg{{1, "read test"}}}
	follTC.req = AppEntrReq{Term: 4, LeaderId: 5000, PreLoggInd: 0, PreLoggTerm: 2, Logg: entries, LeaderCom: 1}
	follTC.respExp = Send{PeerId: 5000, Event: AppEntrResp{Term: 6, Succ: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//sending append request with higher Term
	//-->Expected Leader to become Follower And send Alarm msg from follower.
	follTC.req = AppEntrReq{Term: 7, LeaderId: 5000, PreLoggInd: 4, PreLoggTerm: 2, Logg: entries, LeaderCom: 1}
	follTC.respExp = Send{PeerId: 5000, Event: AppEntrResp{Term: 6, Succ: false}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Alarm{T: 200}
	follTC.expect()
}

func (sm *State_Machine) LeadTesting2(t *testing.T) {
	var follTC TestCases
	follTC.t = t
	var cmdReq = []string{"rename test"}

	//<<<|Id:1000|Status:leader|CurrTerm:6|LoggInd:5|votedFor:0|commitInd:0|>>>

	/* Sending an append request*/
	entr := []MyLogg{sm.Logg.Logg[sm.LoggInd-1], {6, "rename test"}}
	entry := Logg{Logg: entr}
	follTC.req = Append{Data: []byte(cmdReq[0])}
	follTC.respExp = LoggStore{Index: 4, Data: entr}
	sm.ClientCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
	follTC.resp = <-sm.ActionCh
	follTC.respExp = Send{PeerId: 0, Event: AppEntrReq{Term: 6, LeaderId: 1000, PreLoggInd: 4, PreLoggTerm: 6, LeaderCom: 2, Logg: entry}}
	follTC.expect()

	/*Sending append entry response*/
	//sending positive append entry response with lower Term.
	follTC.req = AppEntrResp{Peer: 1000, Term: 6, Succ: true}
	sm.NetCh <- follTC.req
	sm.EventProcess()

	//sending negative append entry response with lower Term.
	temp := sm.NextIndex[2] - 2
	entry2 := sm.Logg.Logg[temp:]
	entry1 := Logg{Logg: entry2}
	follTC.req = AppEntrResp{Peer: 2000, Term: 6, Succ: false}
	follTC.respExp = Send{PeerId: 2000, Event: AppEntrReq{Term: 6, LeaderId: 1000, PreLoggInd: 2, PreLoggTerm: 1, LeaderCom: 2, Logg: entry1}}
	sm.NetCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()
}

func (sm *State_Machine) LeadTesting3(t *testing.T) {
	var follTC TestCases
	follTC.t = t

	/*Sending new Commit entry*/
	//Setting state machine such that majority of follower had send positive append entry response.
	//<<<|Id:1000|Status:leader|CurrTerm:6|LoggInd:8|votedFor:0|commitInd:1|>>>
	entry := Logg{Logg: []MyLogg{{1, "read test"}, {1, "read cloud"}, {1, "read cs733"}, {2, "delete test"}, {4, "cas test"}, {4, "rename cloud"}, {6, "rename test"}, {6, "cas cs733"}}}
	sm.Logg = entry
	sm.LoggInd = 8
	sm.CommitIndex = 1
	sm.CurrTerm = 6
	match := [5]int32{6, 6, 4, 6, 3}
	sm.MatchIndex = match
	//Sending timeout
	//-->Expected to send heartbeat msg to all server with new commit Index.
	follTC.req = Timeout{}
	follTC.respExp = Send{PeerId: 0, Event: AppEntrReq{Term: 6, LeaderId: 1000, PreLoggInd: 7, PreLoggTerm: 6, LeaderCom: 6}}
	sm.TimeoutCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//<<<|Id:1000|Status:leader|CurrTerm:6|LoggInd:8|votedFor:0|commitInd:6|>>>

	//Setting state machine such that majority of follower had not send positive append entry response.
	//<<<|Id:1000|Status:leader|CurrTerm:6|LoggInd:8|votedFor:0|commitInd:1|>>>
	match = [5]int32{6, 2, 4, 6, 3}
	sm.CommitIndex = 1
	sm.MatchIndex = match
	//Sending timeout
	//-->Expected to send heartbeat msg to all server with old commit Index.
	follTC.req = Timeout{}
	follTC.respExp = Send{PeerId: 0, Event: AppEntrReq{Term: 6, LeaderId: 1000, PreLoggInd: 7, PreLoggTerm: 6, LeaderCom: 1}}
	sm.TimeoutCh <- follTC.req
	sm.EventProcess()
	follTC.resp = <-sm.ActionCh
	follTC.expect()

	//<<<|Id:1000|Status:leader|CurrTerm:6|LoggInd:8|votedFor:0|commitInd:1|>>>
}

//Coppying append command into Logg.
func enterLogg(myLogg MyLogg) Logg {
	var Logg1 Logg
	Logg1.Logg = append(Logg1.Logg, myLogg)
	return Logg1
}

//Evaluating expected and recieved Event responses.
func (tc TestCases) expect() {
	fmt.Println(cnt)
	cnt++
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
