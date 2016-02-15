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

	fmt.Printf("##> %T", sm1)

	sm, ok := sm1.(Candidate)
	if ok {
		fmt.Println(sm.status)
	}

	sm, ok = sm1.(Follower)
	if ok {
		fmt.Println(sm.status)
	}
	sm, ok = sm1.(Leader)
	if ok {
		fmt.Println(sm.status)
	}
	/*
		var sm State_Machine
		switch sm1.(type) {
		case Follower:
			sm := sm1.(Follower)
			fmt.Print("*$$$*", sm.status)
			fmt.Printf("##> %T", sm)
		case Candidate:
			sm = sm1.(Candidate)
		case Leader:
			sm = sm1.(Leader)
		}
		fmt.Printf("##>> %T", sm)*/
	fmt.Println(sm.status)

	resp := Commit{data: []byte("5000"), err: []byte("I'm not leader")}
	actionCh <- resp
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
