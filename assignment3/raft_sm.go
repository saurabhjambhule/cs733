package main

import (
	"fmt"
	"math"
	"reflect"
)

//This deals with the incomming rquest and invokes repective response Event.
type Message interface {
	send(sm *State_Machine)
	commit(sm *State_Machine)
	alarm(sm *State_Machine)
}

/*func InitSM() (state State_Machine) {
	var sm1 State_Machine
	return sm1
}
*/

//Function for state to become follower.
func (sm *State_Machine) FollSys() {
	sm.status = FOLL    //Change state status to Follower
	sm.votedFor = 0     //Reinitialize VoteFor
	sm.VoteGrant[0] = 0 //This is positive VoteGrant counter initialized to 1 i.e. self vote
	sm.VoteGrant[1] = 0 //This is negative VoteGrant counter initialized to 0
	//Set timeout
	fmt.Println(">>>", sm)
	resp := Alarm{T: FTO} //200
	actionCh <- resp
	sm.EventProcess()
}

//Function for state to become CandIdate.
func (sm *State_Machine) candSys() {
	sm.status = CAND    //Change state status to CandIdate
	sm.currTerm += 1    //Increament the Term
	sm.votedFor = sm.id //Vote for self
	sm.VoteGrant[0] = 1 //This is positive VoteGrant counter initialized to 1 i.e. self vote
	sm.VoteGrant[1] = 0 //This is negative VoteGrant counter initialized to 0
	fmt.Println(">>>", sm)
	//Set election timeout
	resp := Alarm{T: CTO} //150
	actionCh <- resp
	//Sending vote request null information of previous entry as CandIdate had just joined the cluster.
	if len(sm.Logg.Logg) == 0 {
		respp := Send{PeerId: 0, Event: VoteReq{Term: sm.currTerm, CandId: sm.id, PreLoggInd: 0, PreLoggTerm: 0}}
		actionCh <- respp
	} else {
		//Send vote request to all other servers.
		respp := Send{PeerId: 0, Event: VoteReq{Term: sm.currTerm, CandId: sm.id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term}}
		actionCh <- respp
	}
}

//Function for state to become leader.
func (sm *State_Machine) leadSys() {
	sm.status = LEAD    //Change state status to leader
	sm.votedFor = 0     //Reinitialize VoteFor
	sm.VoteGrant[0] = 0 //This is positive VoteGrant counter initialized to 1 i.e. self vote
	sm.VoteGrant[1] = 0 //This is negative VoteGrant counter initialized to 0
	sm.initialize()     //initialize MatchIndex and nextIndex
	fmt.Println(">>>", sm)
	resp := Alarm{T: LTO} //175
	//Set heartbeat timeout
	actionCh <- resp
	//Send heartbeat msg to all other servers
	//PeerId:0 means to all servers.
	if len(sm.Logg.Logg) == 0 {
		respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: 0, PreLoggTerm: 0, LeaderCom: 0}}
		actionCh <- respp
	} else {
		respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term, LeaderCom: sm.CommitIndex}}
		actionCh <- respp
	}
}

//This will keep listening to all incomming channels and procceed the request as it arrives.
func (sm *State_Machine) EventProcess() {
	var msg Message

	go func(sm *State_Machine) {
		//fmt.Println(msg)
		for {
			select {
			//Timeout Event.
			case <-timeoutCh:
				//Generate corrosponding response to the request.
				switch sm.status {
				case FOLL:
					//fmt.Println("***>>", sm.currTerm)
					//Change state to CandIdate.
					//fmt.Println("--->>", sm.currTerm, " : ", time.Now())
					sm.candSys()

				case CAND:
					//Start election for next Term again.
					sm.candSys()

				case LEAD:
					//fmt.Println("Heartbeat-", len(sm.Logg.Logg))
					//Commit the Logg and send heartbeat msg to all other servers.
					sm.commitLogg()
					if len(sm.Logg.Logg) == 0 {
						respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: 0, PreLoggTerm: 0, LeaderCom: 0}}
						//fmt.Println("--->>", sm.currTerm, " : ", time.Now())
						actionCh <- respp
						resp := Alarm{T: LTO} //175
						//Set heartbeat timeout
						actionCh <- resp
					} else {
						respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term, LeaderCom: sm.CommitIndex}}
						actionCh <- respp
						resp := Alarm{T: LTO} //175
						//Set heartbeat timeout
						actionCh <- resp
					}
				}
			}
		}
	}(sm)

	for {
		//fmt.Println("In EventProcess")
		select {
		//Timeout Event.
		case <-timeoutCh:
			//Generate corrosponding response to the request.
			switch sm.status {
			case FOLL:
				//fmt.Println("***>>", sm.currTerm)
				//Change state to CandIdate.
				sm.candSys()

			case CAND:
				//Start election for next Term again.
				sm.candSys()

			case LEAD:
				//fmt.Println("Heartbeat-", len(sm.Logg.Logg))
				//Commit the Logg and send heartbeat msg to all other servers.
				sm.commitLogg()
				if len(sm.Logg.Logg) == 0 {
					respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: 0, PreLoggTerm: 0, LeaderCom: 0}}
					actionCh <- respp
					resp := Alarm{T: LTO} //175
					//Set heartbeat timeout
					actionCh <- resp
				} else {
					respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term, LeaderCom: sm.CommitIndex}}
					actionCh <- respp
					resp := Alarm{T: LTO} //175
					//Set heartbeat timeout
					actionCh <- resp
				}
			}

		//Requests from client machine.
		case appendMsg := <-clientCh:
			msg = appendMsg.(Append)
			msg.commit(sm)

		//Request from PEERS in the cluster.
		case PeerMsg := <-netCh:
			//Generate corrosponding response to the request.
			switch PeerMsg.(type) {
			case AppEntrReq:
				msg = PeerMsg.(AppEntrReq)
				msg.send(sm)
			case AppEntrResp:
				msg = PeerMsg.(AppEntrResp)
				msg.send(sm)
			case VoteReq:
				msg = PeerMsg.(VoteReq)
				msg.send(sm)
			case VoteResp:
				//fmt.Println("<<<IN<<<")
				msg = PeerMsg.(VoteResp)
				msg.send(sm)
			}

		}
	}
}

//Process incommimg append entry request.
//Incoming Logg or any incoming variable means the Logg or variable form the given incomming request or respoonse msg.
func (appReq AppEntrReq) send(sm *State_Machine) {
	switch sm.status {
	//For every incoming signal from leader to follower reset the timeout time as leader is still alive.
	case FOLL:
		flag := false
		//if follower dont have any entries in Logg meaning he just joined the cluster, then coppy incoming Logg to local Logg.
		//Or if Logg is NULL i.e. Heartbeat msg, reset the timeout.
		if len(sm.Logg.Logg) == 0 {
			for i := 0; i < len(appReq.Logg.Logg); i++ {
				sm.Logg.Logg = append(sm.Logg.Logg, appReq.Logg.Logg[i])
				sm.LoggInd++
				flag = true
			}
			//if incomming Term is higher than local, update the Term.
			sm.currTerm = appReq.Term
			//if false then it is Heartbeat message.
			if flag == false {
				//Reset the timeout timer.
				//fmt.Println("--->>", appReq.Term, " : ", time.Now())
				respp := Alarm{T: FTO}
				actionCh <- respp
				return
			}
			//Send possitive reply, as Logg has been copied to local Logg.
			resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: true}}
			actionCh <- resp
			//Reset the timeout timer.
			respp := Alarm{T: FTO}
			actionCh <- respp
			return
		}
		//Send regative reply, if incoming Term is lower than local Term or previous Index does not match.
		if (sm.currTerm > appReq.Term) || (appReq.PreLoggInd > sm.LoggInd-1) {
			resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: false}}
			actionCh <- resp
			respp := Alarm{T: FTO}
			actionCh <- respp
			return
		}
		//Send regative reply, if previous entry does not match.
		if (appReq.PreLoggInd != sm.LoggInd-1) && (appReq.PreLoggTerm == sm.currTerm) || (!reflect.DeepEqual(sm.Logg.Logg[appReq.PreLoggInd], appReq.Logg.Logg[0])) {
			resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: false}}
			actionCh <- resp
			respp := Alarm{T: FTO}
			actionCh <- respp
			return
		}
		//Update local CommitIndex with minimum of incomming LeaderCommit and local Logg Index
		sm.CommitIndex = int32(math.Min(float64(appReq.LeaderCom), float64(sm.LoggInd-1)))
		//Update local Term to incomming Term.
		sm.currTerm = appReq.Term
		//Copy incoming Logg into local Logg.
		sm.LoggInd, sm.Logg = copyLogg(sm.currTerm, sm.LoggInd, appReq.PreLoggInd, sm.Logg, appReq.Logg)
		//Send possitive reply, as Logg has been copied to local Logg.
		resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: true}}
		actionCh <- resp
		//Reset the timeout timer.
		respp := Alarm{T: FTO}
		actionCh <- respp

	case CAND:
		//Become follower and amd process incomming append entry request, if incomming Term higher than or eqaul(already have Term) to local Term.
		if appReq.Term >= sm.currTerm {
			sm.status = FOLL
			sm.votedFor = 0
			sm.currTerm = appReq.Term
			appReq.send(sm)
			return
		}
		//Reply negative if incomming Term is lower.
		resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: false}}
		actionCh <- resp

	case LEAD:
		//Become a follower, if incomming Term higher than local Term.
		if appReq.Term > sm.currTerm {
			resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: false}}
			actionCh <- resp
			sm.currTerm = appReq.Term
			sm.FollSys()
			return
		}
		//Reply negative if incomming Term is lower.
		resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: false}}
		actionCh <- resp
	}
	return
}

//Process incommimg append entry response.
func (appRes AppEntrResp) send(sm *State_Machine) {
	switch sm.status {
	case LEAD:
		//On positive response, update MatchIndex and NextIndex.
		if appRes.Succ == true {
			sm.MatchIndex[Peer[appRes.Peer]] = sm.LoggInd - 1
			sm.NextIndex[Peer[appRes.Peer]] = sm.LoggInd
		}
		//On negative response, decreament the NextIndex with respect to incomming Peer and resend append entry request.
		if appRes.Succ == false {
			sm.NextIndex[Peer[appRes.Peer]] -= 1
			temp := sm.NextIndex[Peer[appRes.Peer]] - 1
			entry := sm.Logg.Logg[temp:]
			entry1 := Logg{Logg: entry}
			//Check for Logg to be commited.
			sm.commitLogg()
			resp := Send{PeerId: appRes.Peer, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: temp, PreLoggTerm: sm.Logg.Logg[temp].Term, LeaderCom: sm.CommitIndex, Logg: entry1}}
			actionCh <- resp
		}
	}
}

//Process incommimg vote request.
func (votReq VoteReq) send(sm *State_Machine) {
	switch sm.status {
	case FOLL:
		//If CandIdate Logg is not uptodate or incoming Term is lower or already voted in given Term, then reply negative.
		if votReq.Term < sm.currTerm || sm.votedFor != 0 || votReq.PreLoggInd <= sm.LoggInd-1 {
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: false}}
			actionCh <- resp
			return
		}
		//Vote to incomming CandIdate and set the votedFor to 1.
		sm.votedFor = 1
		sm.currTerm = votReq.Term
		resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: true}}
		actionCh <- resp
		//respp := Alarm{T: FCTO}
		//actionCh <- respp

	case CAND:
		if sm.currTerm < votReq.Term {
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: false}}
			actionCh <- resp
			sm.currTerm = votReq.Term
			sm.FollSys()
			return
		}
		//Reject the incomming vote Request.
		resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: false}}
		//fmt.Println("OUT>>>")
		actionCh <- resp

	case LEAD:
		//Reply negative  in any case for vote request.
		if sm.currTerm > votReq.Term {
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: false}}
			actionCh <- resp
		}
		//But if incomming Term is higher than local, then step down to follower state.
		if sm.currTerm < votReq.Term {
			fmt.Println("###", votReq)
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: false}}
			actionCh <- resp
			sm.currTerm = votReq.Term
			sm.FollSys()
		}
	}
}

//Process incommimg vote response.
func (votRes VoteResp) send(sm *State_Machine) {
	switch sm.status {
	case CAND:
		//fmt.Println("***>>", sm.VoteGrant[0], "-", sm.VoteGrant[1])
		//Count incomming positive responses.
		if votRes.VoteGrant == true {
			sm.VoteGrant[0] += 1
		}
		//Count incomming negative responses.
		if votRes.VoteGrant == false {
			sm.VoteGrant[1] += 1
			if votRes.Term > sm.currTerm {
				sm.FollSys()
				return
			}
		}
		//Become Leader if positive responses are atleat 3.
		if sm.VoteGrant[0] >= MAX {
			sm.leadSys()
			return
		}
		//Step down to Follower if negative responses are atleat 3.
		if sm.VoteGrant[1] >= MAX {
			sm.FollSys()
			return
		}
		//Do reelection due to cluster partioning.
		if sm.VoteGrant[0] == 2 && sm.VoteGrant[1] == 2 {
			sm.candSys()
			return
		}
		//fmt.Println("***>>", sm.VoteGrant[0], "-", sm.VoteGrant[1])

	}
}

//Process incommimg append request.
func (app Append) commit(sm *State_Machine) {
	switch sm.status {
	case FOLL:
		//Send Error.
		resp := Commit{Data: []byte("5000"), Err: []byte("I'm not leader")}
		actionCh <- resp

	case CAND:
		//Send Error.
		resp := Commit{Data: []byte("5000"), Err: []byte("I'm not leader")}
		actionCh <- resp

	case LEAD:
		//Append the commond into local Logg.
		ind := sm.LoggInd
		entry := Logg{Logg: []MyLogg{{0, " "}, {sm.currTerm, string(app.Data)}}}
		sm.LoggInd, sm.Logg = copyLogg(sm.currTerm, sm.LoggInd, sm.LoggInd-1, sm.Logg, entry)
		temp := len(sm.Logg.Logg) - 2
		entry11 := sm.Logg.Logg[temp:]
		entry1 := Logg{Logg: entry11}
		resp := LoggStore{Index: ind, Data: app.Data}
		//Send the append entry request to all other servers.
		respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term, LeaderCom: sm.CommitIndex, Logg: entry1}}
		actionCh <- resp
		actionCh <- respp
	}
}

//Commit the Logg, function initiated by leader.
func (sm *State_Machine) commitLogg() {
	for i := sm.CommitIndex + 1; i < sm.LoggInd; i++ {
		if sm.Logg.Logg[i].Term != sm.currTerm {
			continue
		}
		count := 0
		for j := 0; j < PEERS; j++ {
			if sm.MatchIndex[j] >= i {
				count += 1
			}
		}
		if count >= MAX {
			sm.CommitIndex = i
			break
		}
	}
}

//Used to copy Logg from given request to state machine.
func copyLogg(Term int32, myInd int32, preInd int32, oldLogg Logg, newLogg Logg) (int32, Logg) {
	for i := 1; i < len(newLogg.Logg); i++ {
		temp := preInd + int32(i)
		oldLogg.Logg = append(oldLogg.Logg[:temp], newLogg.Logg[i])
		myInd++
	}
	return myInd, oldLogg
}

//Initializing the MatchIndex and NextIndex.
func (sm *State_Machine) initialize() {
	for i := 0; i < PEERS; i++ {
		sm.MatchIndex[i] = 0
		sm.NextIndex[i] = sm.LoggInd
	}
}

//Channel declaration for listening to incomming requests.
var clientCh = make(chan interface{})
var netCh = make(chan interface{})
var timeoutCh = make(chan interface{})

//Channel for providing respond to given request.
var actionCh = make(chan interface{})

/*
//Main function: Starts machine in follower state and assign a unique Id to machine.
func main() {
	//Start the server in Follower state
	sm := State_Machine{Persi_State: Persi_State{id: 1000, currTerm: 0, status: FOLL}, Volat_State: Volat_State{CommitIndex: 0, LastApplied: 0}}
	sm.FollSys()
}
*/
