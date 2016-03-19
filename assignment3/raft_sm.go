package main

import (
	"math"
	"reflect"
)

//This deals with the incomming rquest and invokes repective response Event.
type Message interface {
	send(sm *State_Machine)
	commit(sm *State_Machine)
	alarm(sm *State_Machine)
}

//Function for state to become follower.
func (sm *State_Machine) FollSys() {
	if sm.status == CAND {
		sm.votedFor = 1 //Reinitialize VoteFor
	} else {
		sm.votedFor = 0 //Reinitialize VoteFor
	}
	sm.status = FOLL //Change state status to Follower
	sm.votedFor = 0  //Reinitialize VoteFor
	sm.LeaderId = 0
	sm.VoteGrant[0] = 0 //This is positive VoteGrant counter initialized to 1 i.e. self vote
	sm.VoteGrant[1] = 0 //This is negative VoteGrant counter initialized to 0
	//fmt.Println(">>>", sm)
	resp := Alarm{T: FTO} //200
	sm.CommMedium.actionCh <- resp
	sm.EventProcess()
}

//Function for state to become CandIdate.
func (sm *State_Machine) candSys() {
	sm.status = CAND    //Change state status to CandIdate
	sm.currTerm += 1    //Increament the Term
	sm.votedFor = 1     //Vote for self
	sm.VoteGrant[0] = 1 //This is positive VoteGrant counter initialized to 1 i.e. self vote
	sm.VoteGrant[1] = 0 //This is negative VoteGrant counter initialized to 0
	sm.LeaderId = 0
	//fmt.Println(">>>", sm)
	resp := Alarm{T: CTO} //150
	sm.CommMedium.actionCh <- resp
	//Sending vote request null information of previous entry as CandIdate had just joined the cluster.
	if len(sm.Logg.Logg) == 0 {
		respp := Send{PeerId: 0, Event: VoteReq{Term: sm.currTerm, CandId: sm.id, PreLoggInd: 0, PreLoggTerm: 0}}
		sm.CommMedium.actionCh <- respp
	} else {
		//Send vote request to all other servers.
		respp := Send{PeerId: 0, Event: VoteReq{Term: sm.currTerm, CandId: sm.id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term}}
		sm.CommMedium.actionCh <- respp
	}
}

//Function for state to become leader.
func (sm *State_Machine) leadSys() {
	sm.status = LEAD    //Change state status to leader
	sm.votedFor = 0     //Reinitialize VoteFor
	sm.VoteGrant[0] = 0 //This is positive VoteGrant counter initialized to 1 i.e. self vote
	sm.VoteGrant[1] = 0 //This is negative VoteGrant counter initialized to 0
	sm.initialize()     //initialize MatchIndex and nextIndex
	sm.LeaderId = sm.id
	//fmt.Println("***>>>", sm)
	resp := Alarm{T: LTO} //175
	//Set heartbeat timeout
	sm.CommMedium.actionCh <- resp
	//Send heartbeat msg to all other servers
	//PeerId:0 means to all servers.
	if len(sm.Logg.Logg) == 0 {
		respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: 0, PreLoggTerm: 0, LeaderCom: 0}}
		sm.CommMedium.actionCh <- respp
	} else {
		respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term, LeaderCom: sm.CommitIndex}}
		sm.CommMedium.actionCh <- respp
	}
}

//This will keep listening to all incomming channels and procceed the request as it arrives.
func (sm *State_Machine) EventProcess() {
	var msg Message
	for {
		//fmt.Println("In EventProcess")
		select {

		//Timeout Event.
		case <-sm.CommMedium.timeoutCh:
			//Generate corrosponding response to the request.
			switch sm.status {
			case FOLL:
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
					sm.CommMedium.actionCh <- respp
					resp := Alarm{T: LTO} //175
					//Set heartbeat timeout
					sm.CommMedium.actionCh <- resp
				} else {
					respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term, LeaderCom: sm.CommitIndex}}
					sm.CommMedium.actionCh <- respp
					resp := Alarm{T: LTO} //175
					//Set heartbeat timeout
					sm.CommMedium.actionCh <- resp
				}
			}

		//Requests from client machine.
		case appendMsg := <-sm.CommMedium.clientCh:
			msg = appendMsg.(Append)
			msg.commit(sm)

		//Request from PEERS in the cluster.
		case PeerMsg := <-sm.CommMedium.netCh:
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
	//fmt.Println("--->", sm.id, appReq, len(appReq.Logg.Logg))

	switch sm.status {
	//For every incoming signal from leader to follower reset the timeout time as leader is still alive.
	case FOLL:
		//fmt.Println("--->>>>", appReq)

		if len(appReq.Logg.Logg) == 0 {
			//Reset the timeout timer.
			sm.currTerm = appReq.Term
			respp := Alarm{T: FTO}
			sm.CommMedium.actionCh <- respp
			sm.commitLogg()
			return
		}

		//Send regative reply, if incoming Term is lower than local Term or previous Index does not match.
		//fmt.Println(((appReq.PreLoggInd > sm.LoggInd-1) && len(sm.Logg.Logg) != 0))
		if (sm.currTerm > appReq.Term) || ((appReq.PreLoggInd > sm.LoggInd-1) && len(sm.Logg.Logg) != 0) {
			//	fmt.Println(">>> 2 :", sm.id, "=", appReq.Term)
			//	fmt.Println(appReq.PreLoggInd, " ~~ ", sm.LoggInd-1)

			resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: false}}
			sm.CommMedium.actionCh <- resp
			respp := Alarm{T: FTO}
			sm.CommMedium.actionCh <- respp
			return
		}

		//Send regative reply, if previous entry does not match.   , (!reflect.DeepEqual(sm.Logg.Logg[appReq.PreLoggInd], appReq.Logg.Logg[0]))
		if ((appReq.PreLoggInd != sm.LoggInd-1) && (len(sm.Logg.Logg) != 0)) && (appReq.PreLoggTerm == sm.currTerm) || ((len(sm.Logg.Logg) != 0) && (!reflect.DeepEqual(sm.Logg.Logg[appReq.PreLoggInd], appReq.Logg.Logg[0]))) {
			//	fmt.Println(">>> 3:", sm.id)

			resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: false}}
			sm.CommMedium.actionCh <- resp
			respp := Alarm{T: FTO}
			sm.CommMedium.actionCh <- respp
			return
		}

		//Update local CommitIndex with minimum of incomming LeaderCommit and local Logg Index
		sm.CommitIndex = int32(math.Min(float64(appReq.LeaderCom), float64(sm.LoggInd-1)))
		//Update local Term to incomming Term.
		sm.currTerm = appReq.Term
		prevInd := sm.LoggInd //index on which data will store.
		//Copy incoming Logg into local Logg.
		sm.LoggInd, sm.Logg = copyLogg(sm.currTerm, sm.LoggInd, appReq.PreLoggInd, sm.Logg, appReq.Logg)
		//Send possitive reply, as Logg has been copied to local Logg.
		resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: true}}
		sm.CommMedium.actionCh <- resp
		//Reset the timeout timer.
		respp := Alarm{T: FTO}
		sm.CommMedium.actionCh <- respp
		sm.commitLogg()
		sm.actionCh <- LoggStore{Data: (sm.Logg.Logg[prevInd:])}
		sm.CommMedium.CommitCh <- CommitInfo{Data: []byte(appReq.Logg.Logg[1].Logg), Err: nil, Index: sm.LoggInd - 1}

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
		sm.CommMedium.actionCh <- resp

	case LEAD:
		//Become a follower, if incomming Term higher than local Term.
		if appReq.Term > sm.currTerm {
			if appReq.PreLoggInd > sm.LoggInd-1 {
				resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: false}}
				sm.CommMedium.actionCh <- resp
				sm.currTerm = appReq.Term
				sm.FollSys()
			}
			resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: false}}
			sm.CommMedium.actionCh <- resp
			return
		}
		//Reply negative if incomming Term is lower.
		resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.currTerm, Succ: false}}
		sm.CommMedium.actionCh <- resp
	}
	return
}

//Process incommimg append entry response.
func (appRes AppEntrResp) send(sm *State_Machine) {
	switch sm.status {
	case LEAD:
		//On positive response, update MatchIndex and NextIndex.
		if appRes.Succ == true {
			sm.MatchIndex[appRes.Peer-1] = sm.LoggInd - 1
			sm.NextIndex[appRes.Peer-1] = sm.LoggInd
		}
		//On negative response, decreament the NextIndex with respect to incomming Peer and resend append entry request.
		if appRes.Succ == false {
			if appRes.Term > sm.currTerm {
				sm.currTerm = appRes.Term
				sm.FollSys()
			}
			temp := sm.NextIndex[appRes.Peer-1] - 1
			if temp < 0 {
				temp = 0
			}
			sm.NextIndex[appRes.Peer-1] -= 1
			//temp := sm.NextIndex[appRes.Peer-1] - 1
			entry := sm.Logg.Logg[temp:]
			entry1 := Logg{Logg: entry}
			//Check for Logg to be commited.
			resp := Send{PeerId: appRes.Peer, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: temp, PreLoggTerm: sm.Logg.Logg[temp].Term, LeaderCom: sm.CommitIndex, Logg: entry1}}
			sm.CommMedium.actionCh <- resp
		}
		sm.commitLogg()
	}
}

//Process incommimg vote request.
func (votReq VoteReq) send(sm *State_Machine) {
	switch sm.status {
	case FOLL:
		//If CandIdate Logg is not uptodate or incoming Term is lower or already voted in given Term, then reply negative.
		if votReq.Term < sm.currTerm || sm.votedFor != 0 || votReq.PreLoggInd <= sm.LoggInd-1 {
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: false}}
			sm.CommMedium.actionCh <- resp
			return
		}
		//Vote to incomming CandIdate and set the votedFor to 1.
		sm.votedFor = 1
		sm.currTerm = votReq.Term
		resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: true}}
		sm.CommMedium.actionCh <- resp
		respp := Alarm{T: FTO}
		sm.CommMedium.actionCh <- respp

	case CAND:
		if sm.currTerm < votReq.Term {
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: true}}
			sm.CommMedium.actionCh <- resp
			sm.currTerm = votReq.Term
			sm.FollSys()
			return
		}
		//Reject the incomming vote Request.
		resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: false}}
		sm.CommMedium.actionCh <- resp

	case LEAD:
		//Reply negative  in any case for vote request.
		if sm.currTerm > votReq.Term {
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: false}}
			sm.CommMedium.actionCh <- resp
		}
		//But if incomming Term is higher than local, then step down to follower state.
		if sm.currTerm < votReq.Term {
			if votReq.PreLoggInd > sm.LoggInd-1 {
				resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: false}}
				sm.CommMedium.actionCh <- resp
				sm.currTerm = votReq.Term
				sm.FollSys()
			}
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.currTerm, VoteGrant: false}}
			sm.CommMedium.actionCh <- resp
		}
	}
}

//Process incommimg vote response.
func (votRes VoteResp) send(sm *State_Machine) {
	switch sm.status {
	case CAND:
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
	}
}

//Process incommimg append request.
func (app Append) commit(sm *State_Machine) {
	switch sm.status {
	case FOLL:
		//Send Error.
		resp := Commit{Data: []byte("5000"), Err: []byte("I'm not leader")}
		sm.CommMedium.actionCh <- resp

	case CAND:
		//Send Error.
		resp := Commit{Data: []byte("5000"), Err: []byte("I'm not leader")}
		sm.CommMedium.actionCh <- resp

	case LEAD:
		//Append the commond into local Logg.
		var temp int
		entry := Logg{Logg: []MyLogg{{-1, " "}, {sm.currTerm, string(app.Data)}}}
		sm.LoggInd, sm.Logg = copyLogg(sm.currTerm, sm.LoggInd, sm.LoggInd-1, sm.Logg, entry)
		entry11 := sm.Logg.Logg
		if len(sm.Logg.Logg) == 1 {
			entry11 = []MyLogg{{-1, "nil"}, {sm.currTerm, string(app.Data)}}
		} else {
			temp = len(sm.Logg.Logg) - 2
			entry11 = sm.Logg.Logg[temp:]
		}
		entry1 := Logg{Logg: entry11}
		entry111 := sm.Logg.Logg[temp:]
		resp := LoggStore{Data: entry111}
		//Send the append entry request to all other servers.
		respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.currTerm, LeaderId: sm.id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term, LeaderCom: sm.CommitIndex, Logg: entry1}}

		sm.CommMedium.actionCh <- resp
		sm.CommMedium.actionCh <- respp
		sm.CommMedium.CommitCh <- CommitInfo{Data: []byte(sm.Logg.Logg[sm.LoggInd-1].Logg), Err: nil, Index: sm.LoggInd - 1}

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
		if len(oldLogg.Logg) != 0 {
			oldLogg.Logg = append(oldLogg.Logg[:temp], newLogg.Logg[i])
		} else {
			oldLogg.Logg = append(oldLogg.Logg, newLogg.Logg[i])
		}
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
