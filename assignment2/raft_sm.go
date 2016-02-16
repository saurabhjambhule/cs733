package main

import "fmt"

//This deals with the incomming rquest and invokes repective response event.
type Message interface {
	send(sm State_Machine)
	commit(sm State_Machine)
	alarm(sm State_Machine)
}

type State_Machine interface {
}

//Function for follower state.
func follSys(sm1 State_Machine) {
	sm := sm1.(Follower)
	sm.commitIndex = 0
	sm.lastApplied = 0
	eventLoop(sm)
}

//Function for candidate state.
func candSys(sm1 State_Machine) {
	sm := sm1.(Candidate)
	sm.commitIndex = 0
	sm.lastApplied = 0
	eventLoop(sm)
}

//Function for leader state.
func leadSys(sm1 State_Machine) {
	sm := sm1.(Leader)
	sm.commitIndex = 0
	sm.lastApplied = 0
	eventLoop(sm)
}

//This will keep listening to all incomming channels and procceed the request as it arrives.
func eventLoop(sm State_Machine) {
	var msg Message
	for {
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
			case VoteReq:
			case VoteResp:
			}
		//Timeout event.
		case <-timeoutCh:

		}
	}
}

func (appReq AppEntrReq) send(sm1 State_Machine) {

	switch sm1.(type) {
	case Follower:
		sm := sm1.(Follower)
		if (sm.currTerm > appReq.term) || ((appReq.preLogInd != sm.logInd) && (appReq.preLogTerm == sm.currTerm)) {
			resp := AppEntrResp{term: sm.currTerm, succ: false}
			actionCh <- resp
			return
		}
		//fmt.Println(sm.logInd)
		sm.logInd, sm.log = copyLog(sm.currTerm, sm.logInd, sm.log, appReq.log)
		//fmt.Println(sm.logInd)
		resp := AppEntrResp{term: sm.currTerm, succ: true}
		actionCh <- resp

	case Candidate:
		sm := sm1.(Candidate)
		fmt.Println(sm.status)

	case Leader:
		sm := sm1.(Leader)
		fmt.Println(sm.status)

	}

}

func (appRes AppEntrResp) send(sm State_Machine) {

}

func (votReq VoteReq) send(sm State_Machine) {

}

func (votRes VoteResp) send(sm State_Machine) {

}

func (app Append) commit(sm State_Machine) {
	resp := Commit{data: []byte("5000"), err: []byte("I'm not leader")}
	actionCh <- resp
	//fmt.Println("@##", string(sm.status))
}

func (to Timeout) alarm(sm State_Machine) {

}

func copyLog(term uint32, myInd int32, oldLog Log, newLog Log) (int32, Log) {
	for i := 0; i < len(newLog.log); i++ {
		oldLog.log = append(oldLog.log, newLog.log[i])
		myInd++
	}
	return myInd, oldLog
}

//Channel declaration for listening to incomming requests.
var clientCh = make(chan interface{})
var netCh = make(chan interface{})
var timeoutCh = make(chan interface{})

//Channel for providing respond to given request.
var actionCh = make(chan interface{})

//Main function: Starts machine in follower state and assign a unique Id to machine.
func main() {
	var sm State_Machine
	follObj := Follower{Persi_State: Persi_State{id: 1000, currTerm: 0}, Volat_State: Volat_State{status: FOLL, commitIndex: 0, lastApplied: 0, timer: 1500}}
	sm = follObj
	follSys(sm)
}
