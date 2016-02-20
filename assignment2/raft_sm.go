package main

import "reflect"

//This deals with the incomming rquest and invokes repective response event.
type Message interface {
	send(sm *State_Machine)
	commit(sm *State_Machine)
	alarm(sm *State_Machine)
}

//Function for state to become follower.
func (sm *State_Machine) follSys() {
	sm.status = FOLL //Change state status to Follower
	sm.votedFor = 0  //Reinitialize VoteFor
	//Set timeout
	resp := Alarm{t: 200}
	actionCh <- resp
}

//Function for state to become candidate.
func (sm *State_Machine) candSys() {

	sm.status = CAND    //Change state status to candidate
	sm.currTerm += 1    //Increament the term
	sm.votedFor = sm.id //Vote for self
	sm.voteGrant[0] = 0 //This is positive voteGrant counter initialized to 0
	sm.voteGrant[1] = 0 //This is negative voteGrant counter initialized to 0
	//Set election timeout
	resp := Alarm{t: 150}
	actionCh <- resp
	//Send vote request to all other servers
	respp := Send{peerId: 0, event: VoteReq{term: sm.currTerm, candId: sm.id, preLogInd: sm.logInd - 1, preLogTerm: sm.log.log[sm.logInd-1].term}}
	actionCh <- respp
}

//Function for state to become leader.
func (sm *State_Machine) leadSys() {
	sm.status = LEAD //Change state status to leader
	sm.votedFor = 0  //Reinitialize VoteFor
	sm.initialize()  //initialize matchIndex and nestIndex
	resp := Alarm{t: 175}
	//Set heartbeat timeout
	actionCh <- resp
	//Send heartbeat msg to all other servers
	//peerId:0 means to all servers.
	respp := Send{peerId: 0, event: AppEntrReq{term: sm.currTerm, leaderId: sm.id, preLogInd: sm.logInd - 1, preLogTerm: sm.log.log[sm.logInd-1].term, leaderCom: sm.commitIndex}}
	actionCh <- respp
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
		//Generate corrosponding response to the request.
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
		//Generate corrosponding response to the request.
		switch sm.status {
		case FOLL:
			//Change state to candidate.
			sm.candSys()

		case CAND:
			//Start election for next term again.
			sm.candSys()

		case LEAD:
			//Commit the log and send heartbeat msg to all other servers.
			sm.commitLog()
			resp := Send{peerId: 0, event: AppEntrReq{term: sm.currTerm, leaderId: sm.id, preLogInd: sm.logInd - 1, preLogTerm: sm.log.log[sm.logInd-1].term, leaderCom: sm.commitIndex}}
			actionCh <- resp
		}
	}
}

//Process incommimg append entry request.
//Incoming log or any incoming variable means the log or variable form the given incomming request or respoonse msg.
func (appReq AppEntrReq) send(sm *State_Machine) {
	switch sm.status {
	case FOLL:
		//if follower dont have any entries in log meaning he just joined the cluster, then coppy incoming log to local log.
		if len(sm.log.log) == 0 {
			for i := 0; i < len(appReq.log.log); i++ {
				sm.log.log = append(sm.log.log, appReq.log.log[i])
				sm.logInd++
			}
			//if incomming term is higher than local, update the term.
			sm.currTerm = appReq.term
			//Send possitive reply, as log has been copied to local log.
			resp := Send{peerId: appReq.leaderId, event: AppEntrResp{term: sm.currTerm, succ: true}}
			actionCh <- resp
			//Reset the timeout timer.
			respp := Alarm{t: 200}
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
			resp := Send{peerId: appReq.leaderId, event: AppEntrResp{term: sm.currTerm, succ: false}}
			respp := Alarm{t: 200}
			actionCh <- resp
			actionCh <- respp
			return
		}
		if appReq.leaderCom > sm.logInd {
			sm.commitIndex = sm.logInd
		} else {
			sm.commitIndex = appReq.leaderCom
		}
		sm.currTerm = appReq.term
		sm.logInd, sm.log = copyLog(sm.currTerm, sm.logInd, appReq.preLogInd, sm.log, appReq.log)
		//Send possitive reply, as log has been copied to local log.
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

//Process incommimg append entry response.
func (appRes AppEntrResp) send(sm *State_Machine) {
	switch sm.status {
	case LEAD:
		if appRes.succ == true {

			sm.matchIndex[peer[appRes.peer]] = sm.logInd - 1
			sm.nextIndex[peer[appRes.peer]] = sm.logInd

		}
		if appRes.succ == false {
			sm.nextIndex[peer[appRes.peer]] -= 1
			temp := sm.nextIndex[peer[appRes.peer]] - 1
			entry := sm.log.log[temp:]
			entry1 := Log{log: entry}
			sm.commitLog()
			resp := Send{peerId: appRes.peer, event: AppEntrReq{term: sm.currTerm, leaderId: sm.id, preLogInd: temp, preLogTerm: sm.log.log[temp].term, leaderCom: sm.commitIndex, log: entry1}}
			actionCh <- resp

		}
	}
}

//Process incommimg vote request.
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

//Process incommimg vote response.
func (votRes VoteResp) send(sm *State_Machine) {
	switch sm.status {
	case CAND:
		//Count incomming positive responses.
		if votRes.voteGrant == true {
			sm.voteGrant[0] += 1
		}
		//Count incomming negative responses.
		if votRes.voteGrant == false {
			sm.voteGrant[1] += 1
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
	}
}

//Process incommimg append request.
func (app Append) commit(sm *State_Machine) {
	switch sm.status {
	case FOLL:
		resp := Commit{data: []byte("5000"), err: []byte("I'm not leader")}
		actionCh <- resp

	case CAND:
		resp := Commit{data: []byte("5000"), err: []byte("I'm not leader")}
		actionCh <- resp

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

//Commit the Log, function initiated by leader.
func (sm *State_Machine) commitLog() {
	for i := sm.commitIndex + 1; i < sm.logInd; i++ {
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

/*
//Random function to select random time for election timeout.
//Not covered in test cases as output will be non deterministic.
func random() int {
	min := 150
	max := 300
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
*/

//Used to copy log from given request to state machine.
func copyLog(term int32, myInd int32, preInd int32, oldLog Log, newLog Log) (int32, Log) {
	for i := 1; i < len(newLog.log); i++ {
		temp := preInd + int32(i)
		oldLog.log = append(oldLog.log[:temp], newLog.log[i])
		myInd++
	}
	return myInd, oldLog
}

//Initializing the matchIndex and nextIndex.
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

/*
//Main function: Starts machine in follower state and assign a unique Id to machine.
func main() {
	//Start the server in Follower state
	sm := State_Machine{Persi_State: Persi_State{id: 1000, currTerm: 0, status: FOLL}, Volat_State: Volat_State{commitIndex: 0, lastApplied: 0}}
	sm.follSys()
}
*/
