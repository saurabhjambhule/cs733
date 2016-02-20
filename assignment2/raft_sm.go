package main

import (
	"fmt"
	"math/rand"
	"reflect"
	"time"
)

//This deals with the incomming rquest and invokes repective response event.
type Message interface {
	send(sm *State_Machine)
	commit(sm *State_Machine)
	alarm(sm *State_Machine)
}

//Function for follower state.
func (sm *State_Machine) follSys() {
	sm.status = FOLL
	sm.votedFor = 0
	resp := Alarm{t: 200}
	actionCh <- resp
	//sm.eventProcess()
}

//Function for candidate state.
func (sm *State_Machine) candSys() {
	sm.status = CAND
	sm.currTerm += 1
	sm.votedFor = sm.id
	sm.voteGrant[0] = 0
	sm.voteGrant[1] = 0
	fmt.Println(sm.log)
	resp := Alarm{t: 150}
	respp := Send{peerId: 0, event: VoteReq{term: sm.currTerm, candId: sm.id, preLogInd: sm.logInd - 1, preLogTerm: sm.log.log[sm.logInd-1].term}}
	actionCh <- resp
	actionCh <- respp

	//sm.eventProcess()
}

//Function for leader state.
func (sm *State_Machine) leadSys() {
	sm.status = LEAD
	sm.votedFor = 0
	sm.initialize()
	resp := Alarm{t: 175}
	respp := Send{peerId: 0, event: AppEntrReq{term: sm.currTerm, leaderId: sm.id, preLogInd: sm.logInd - 1, preLogTerm: sm.log.log[sm.logInd-1].term, leaderCom: sm.commitIndex}}
	actionCh <- resp
	actionCh <- respp
	//sm.eventProcess()
}

//This will keep listening to all incomming channels and procceed the request as it arrives.
func (sm *State_Machine) eventProcess() {
	var msg Message
	select {
	//Requests from client machine.
	case appendMsg := <-clientCh:
		msg = appendMsg.(Append)
		msg.commit(sm)

	//Request from peers in the cluster.
	case peerMsg := <-netCh:
		switch peerMsg.(type) {
		case AppEntrReq:
			msg = peerMsg.(AppEntrReq)
			msg.send(sm)
		case AppEntrResp:
			msg = peerMsg.(AppEntrResp)
			msg.send(sm)
		case VoteReq:
			msg = peerMsg.(VoteReq)
			msg.send(sm)
		case VoteResp:
			msg = peerMsg.(VoteResp)
			msg.send(sm)
		}

	//Timeout event.
	case <-timeoutCh:
		switch sm.status {
		case FOLL:
			fmt.Println("@@@")
			sm.candSys()

		case CAND:
			//tNo := random()
			//resp := Alarm{t: tNo}
			//actionCh <- resp
			sm.candSys()

		case LEAD:
			resp := Send{peerId: 0, event: AppEntrReq{term: sm.currTerm, leaderId: sm.id, preLogInd: sm.logInd - 1, preLogTerm: sm.log.log[sm.logInd-1].term, leaderCom: sm.commitIndex}}
			actionCh <- resp
		}
	}
}

func (sm *State_Machine) chngeState() {
	switch sm.status {
	case FOLL:

	case CAND:

	case LEAD:
	}
}

func random() int {
	min := 150
	max := 300
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func (appReq AppEntrReq) send(sm *State_Machine) {
	switch sm.status {
	case FOLL:
		if len(sm.log.log) == 0 {
			for i := 0; i < len(appReq.log.log); i++ {
				sm.log.log = append(sm.log.log, appReq.log.log[i])
				sm.logInd++
			}
			sm.currTerm = appReq.term
			resp := Send{peerId: appReq.leaderId, event: AppEntrResp{term: sm.currTerm, succ: true}}
			respp := Alarm{t: 200}
			actionCh <- resp
			actionCh <- respp
			return
		}
		if (sm.currTerm > appReq.term) || (appReq.preLogInd > sm.logInd-1) {
			resp := Send{peerId: appReq.leaderId, event: AppEntrResp{term: sm.currTerm, succ: false}}
			respp := Alarm{t: 200}
			actionCh <- resp
			actionCh <- respp
			return
		}
		if len(appReq.log.log) == 0 {
			resp := Alarm{t: 200}
			actionCh <- resp
			return
		}
		if (appReq.preLogInd != sm.logInd-1) && (appReq.preLogTerm == sm.currTerm) || (!reflect.DeepEqual(sm.log.log[appReq.preLogInd], appReq.log.log[0])) {
			fmt.Println(sm.currTerm, appReq.term)
			resp := Send{peerId: appReq.leaderId, event: AppEntrResp{term: sm.currTerm, succ: false}}
			respp := Alarm{t: 200}
			actionCh <- resp
			actionCh <- respp
			return
		}
		sm.commitIndex = appReq.leaderCom
		sm.currTerm = appReq.term
		sm.logInd, sm.log = copyLog(sm.currTerm, sm.logInd, appReq.preLogInd, sm.log, appReq.log)
		resp := Send{peerId: appReq.leaderId, event: AppEntrResp{term: sm.currTerm, succ: true}}
		respp := Alarm{t: 200}
		actionCh <- resp
		actionCh <- respp

	case CAND:
		if appReq.term >= sm.currTerm {
			sm.status = FOLL
			sm.votedFor = 0
			sm.currTerm = appReq.term
			appReq.send(sm)
			return
		}
		resp := Send{peerId: appReq.leaderId, event: AppEntrResp{term: sm.currTerm, succ: false}}
		actionCh <- resp

	case LEAD:
		if appReq.term > sm.currTerm {
			resp := Send{peerId: appReq.leaderId, event: AppEntrResp{term: sm.currTerm, succ: false}}
			actionCh <- resp
			sm.currTerm = appReq.term
			sm.follSys()
			return
		}
		resp := Send{peerId: appReq.leaderId, event: AppEntrResp{term: sm.currTerm, succ: false}}
		actionCh <- resp
	}
	return
}

func (appRes AppEntrResp) send(sm *State_Machine) {
	switch sm.status {
	case FOLL:

	case CAND:

	case LEAD:
		if appRes.succ == true {

			sm.matchIndex[peer[appRes.peer]] = sm.logInd - 1
			sm.nextIndex[peer[appRes.peer]] = sm.logInd

		}
		if appRes.succ == false {
			fmt.Println("------>>", sm.matchIndex)
			fmt.Println("------>>", sm.nextIndex)
			sm.nextIndex[peer[appRes.peer]] -= 1
			temp := sm.nextIndex[peer[appRes.peer]] - 1
			entry := sm.log.log[temp:]
			entry1 := Log{log: entry}
			fmt.Println("!!------>>", entry1)
			fmt.Println("------>>", sm.matchIndex)
			fmt.Println("------>>", sm.nextIndex)
			resp := Send{peerId: appRes.peer, event: AppEntrReq{term: sm.currTerm, leaderId: sm.id, preLogInd: temp, preLogTerm: sm.log.log[temp].term, leaderCom: sm.commitIndex, log: entry1}}
			fmt.Println(resp)
			fmt.Println("###@@@", sm.log)

			actionCh <- resp

		}
	}
}

func (votReq VoteReq) send(sm *State_Machine) {
	switch sm.status {
	case FOLL:
		if votReq.term < sm.currTerm || sm.votedFor != 0 || votReq.preLogInd <= +sm.logInd-1 {
			resp := Send{peerId: votReq.candId, event: VoteResp{term: sm.currTerm, voteGrant: false}}
			actionCh <- resp
			return
		}
		sm.votedFor = 1
		sm.currTerm = votReq.term
		resp := Send{peerId: votReq.candId, event: VoteResp{term: sm.currTerm, voteGrant: true}}
		actionCh <- resp

	case CAND:
		resp := Send{peerId: votReq.candId, event: VoteResp{term: sm.currTerm, voteGrant: false}}
		actionCh <- resp

	case LEAD:
		if sm.currTerm > votReq.term {
			resp := Send{peerId: votReq.candId, event: VoteResp{term: sm.currTerm, voteGrant: false}}
			actionCh <- resp
		}
		if sm.currTerm < votReq.term {
			resp := Send{peerId: votReq.candId, event: VoteResp{term: sm.currTerm, voteGrant: false}}
			actionCh <- resp
			sm.currTerm = votReq.term
			sm.follSys()
		}
	}
}

func (votRes VoteResp) send(sm *State_Machine) {
	switch sm.status {
	case FOLL:

	case CAND:
		//Count incomming positive responses.
		if votRes.voteGrant == true {
			sm.voteGrant[0] += 1
		}
		//Count incomming negative responses.
		if votRes.voteGrant == false {
			sm.voteGrant[1] += 1
			fmt.Println(votRes.term > sm.currTerm)
			if votRes.term > sm.currTerm {
				sm.follSys()
				return
			}
		}
		//Become Leader if positive responses are atleat 3.
		if sm.voteGrant[0] >= 3 {
			sm.leadSys()
			return
		}
		//Step down to Follower if negative responses are atleat 3.
		if sm.voteGrant[1] >= 3 {
			sm.follSys()
			return
		}
		//Do reelection due to cluster partioning.
		if sm.voteGrant[0] == 2 && sm.voteGrant[1] == 2 {
			sm.candSys()
			return
		}

	case LEAD:

	}
}

func (app Append) commit(sm *State_Machine) {
	switch sm.status {
	case FOLL:
		resp := Commit{data: []byte("5000"), err: []byte("I'm not leader")}
		actionCh <- resp

	case CAND:
		resp := Commit{data: []byte("5000"), err: []byte("I'm not leader")}
		actionCh <- resp
		fmt.Println(sm.status)

	case LEAD:
		ind := sm.logInd
		entry := Log{log: []MyLog{{0, " "}, {sm.currTerm, string(app.data)}}}
		sm.logInd, sm.log = copyLog(sm.currTerm, sm.logInd, sm.logInd-1, sm.log, entry)
		temp := len(sm.log.log) - 2
		entry11 := sm.log.log[temp:]
		entry1 := Log{log: entry11}
		resp := LogStore{index: ind, data: app.data}
		respp := Send{peerId: 0, event: AppEntrReq{term: sm.currTerm, leaderId: sm.id, preLogInd: sm.logInd - 1, preLogTerm: sm.log.log[sm.logInd-1].term, leaderCom: sm.commitIndex, log: entry1}}
		actionCh <- resp
		actionCh <- respp
	}
}

func (sm *State_Machine) commitLog() {
	for i := sm.commitIndex + 1; i <= sm.logInd; i++ {
		if sm.log.log[i].term != sm.currTerm {
			continue
		}
		count := 0
		for j := 0; j < 5; j++ {
			if sm.matchIndex[j] >= i {
				count += 1
			}
		}
		if count >= 3 {
			sm.commitIndex = i
			break
		}
	}
}

//Used to copy log from given request to state machine.
func copyLog(term int32, myInd int32, preInd int32, oldLog Log, newLog Log) (int32, Log) {
	for i := 1; i < len(newLog.log); i++ {
		temp := preInd + int32(i)
		oldLog.log = append(oldLog.log[:temp], newLog.log[i])
		myInd++
	}
	return myInd, oldLog
}

func (sm *State_Machine) initialize() {
	for i := 0; i < 5; i++ {
		sm.matchIndex[i] = 0
		sm.nextIndex[i] = sm.logInd
	}
}

//Channel declaration for listening to incomming requests.
var clientCh = make(chan interface{}, 5)
var netCh = make(chan interface{}, 5)
var timeoutCh = make(chan interface{}, 5)

//Channel for providing respond to given request.
var actionCh = make(chan interface{}, 5)

//Main function: Starts machine in follower state and assign a unique Id to machine.
func main() {
	sm := State_Machine{Persi_State: Persi_State{id: 1000, currTerm: 0, status: FOLL}, Volat_State: Volat_State{commitIndex: 0, lastApplied: 0}}
	sm.follSys()
}
