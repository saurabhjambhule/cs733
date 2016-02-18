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
	sm.commitIndex = 0
	sm.lastApplied = 0
	sm.status = FOLL
	//sm.eventProcess()
}

//Function for candidate state.
func (sm *State_Machine) candSys() {
	sm.commitIndex = 0
	sm.lastApplied = 0
	sm.status = CAND
	sm.currTerm += 1
	sm.votedFor = sm.id
	resp := Alarm{t: 5}
	//respp := VoteReq{term: sm.currTerm, candId: sm.id, preLogInd: sm.logInd, preLogTerm: sm.log.log[sm.logInd].term}
	respp := VoteReq{term: sm.currTerm, candId: sm.id}
	actionCh <- resp
	actionCh <- respp
	//sm.eventProcess()
}

//Function for leader state.
func (sm *State_Machine) leadSys() {
	sm.commitIndex = 0
	sm.lastApplied = 0
	sm.eventProcess()
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
			sm.candSys()

		case CAND:
			tNo := random()
			resp := Alarm{t: tNo}
			actionCh <- resp

		case LEAD:
			resp := Alarm{t: 5}
			actionCh <- resp
		}
	}
}

func (sm *State_Machine) chngeState() {
	switch sm.status {
	case FOLL:
		fmt.Println("$$!")

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
			resp := AppEntrResp{term: sm.currTerm, succ: true}
			actionCh <- resp
			return
		}
		if (sm.currTerm > appReq.term) || (appReq.preLogInd > sm.logInd-1) {
			resp := AppEntrResp{term: sm.currTerm, succ: false}
			actionCh <- resp
			return
		}
		if len(appReq.log.log) == 0 {
			resp := Alarm{t: 5}
			actionCh <- resp
			return
		}
		if (appReq.preLogInd != sm.logInd-1) && (appReq.preLogTerm == sm.currTerm) || (!reflect.DeepEqual(sm.log.log[appReq.preLogInd], appReq.log.log[0])) {
			fmt.Println(sm.currTerm, appReq.term)
			resp := AppEntrResp{term: sm.currTerm, succ: false}
			actionCh <- resp
			return
		}
		sm.commitIndex = appReq.leaderCom
		sm.currTerm = appReq.term
		sm.logInd, sm.log = copyLog(sm.currTerm, sm.logInd, appReq.preLogInd, sm.log, appReq.log)
		resp := AppEntrResp{term: sm.currTerm, succ: true}
		actionCh <- resp

	case CAND:

	case LEAD:
	}
	return
}

func (appRes AppEntrResp) send(sm *State_Machine) {
}

func (votReq VoteReq) send(sm *State_Machine) {
	switch sm.status {
	case FOLL:
		if votReq.term < sm.currTerm || sm.votedFor == 1 || votReq.preLogInd < sm.logInd-1 {
			resp := VoteResp{term: sm.currTerm, voteGrant: false}
			actionCh <- resp
			return
		}
		sm.votedFor = 1
		resp := VoteResp{term: sm.currTerm, voteGrant: true}
		actionCh <- resp

	case CAND:
		resp := VoteResp{term: sm.currTerm, voteGrant: false}
		actionCh <- resp

	case LEAD:
		fmt.Println(sm.status)
	}
}

func (votRes VoteResp) send(sm *State_Machine) {

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
		fmt.Println(sm.status)
	}

}

func (to Timeout) alarm(sm1 *State_Machine) {

}

//Used to copy log from given request to state machine.
func copyLog(term uint32, myInd int32, preInd int32, oldLog Log, newLog Log) (int32, Log) {
	fmt.Println("@##@->", preInd, "-", len(newLog.log))

	for i := 1; i < len(newLog.log); i++ {
		fmt.Println("@@->", oldLog.log)
		//fmt.Println(oldLog.log[:(preInd+i)], newLog.log[i])
		temp := preInd + int32(i)
		fmt.Println(temp)
		oldLog.log = append(oldLog.log[:temp], newLog.log[i])
		fmt.Println("@@->", oldLog.log)

		myInd++
	}
	return myInd, oldLog
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
