package sm

import "math"

const (
	FOLL  = "follower"
	CAND  = "CandIdate"
	LEAD  = "leader"
	PEERS = 5
	MAX   = 3
	FTO   = 0
	CTO   = 1
	LTO   = 2
)

//Contains persistent state of all servers.
type Persi_State struct {
	CurrTerm    int32
	VotedFor    int32
	VoteGrant   [2]int32
	CommitIndex int32
}

//Contains volatile state of servers.
type Volat_State struct {
	Id          int32
	Status      string
	LoggInd     int32
	LastApplied int32
	currLeader  int32
	LeaderId    int32
}

//Contains volatile state of the leader.
type Volat_LState struct {
	NextIndex  [PEERS]int32
	MatchIndex [PEERS]int32
	SentIndex  [PEERS]int32
}

//Stores Logg entries
type MyLogg struct {
	Id   string
	Term int32
	Logg string
}

type Logg struct {
	Logg []MyLogg
}

//Contains all the state with respect to given machine.
type State_Machine struct {
	Persi_State
	Volat_State
	Volat_LState
	Logg Logg
	CommMedium
}

type CommMedium struct {
	//Channel declaration for listening to incomming requests.
	ClientCh  chan interface{}
	NetCh     chan interface{}
	TimeoutCh chan interface{}
	//Channel for provIding respond to given request.
	ActionCh     chan interface{}
	CommitInfoCh chan interface{}
	ShutdownCh   chan interface{}
	CommitCh     chan interface{}
}

//AppendEntriesRequest: Invoked by leader to replicate Logg entries and also used as heartbeat.
type AppEntrReq struct {
	Term        int32
	LeaderId    int32
	PreLoggInd  int32
	PreLoggTerm int32
	LeaderCom   int32
	Logg        Logg
}

//AppendEntriesResponse: Invoked by servers on AppendEntriesRequest.
type AppEntrResp struct {
	Peer    int32
	Term    int32
	Succ    bool
	MyInd   int32
	YourInd int32
}

//VoteRequest: Invoked by CandIdates to gather votes.
type VoteReq struct {
	Term        int32
	CandId      int32
	PreLoggInd  int32
	PreLoggTerm int32
}

//VoteResponse: Invoked by servers on VoteRequest.
type VoteResp struct {
	Term      int32
	VoteGrant bool
}

//This is a request from the layer above to append the data to the replicated Logg.
type Append struct {
	Id   string
	Data []byte
}

//A timeout Event.
type Timeout struct {
}

//Send this Event to a remote node.
type Send struct {
	PeerId int32
	Event  interface{}
}

//Invoked by the leader on Append request. ProvIdes (Index + data) or report an error (data + err) to the layer above.
type Commit struct {
	Index int32
	Data  MyLogg
	Err   []byte
	Exec  bool
}

type CommitInfo struct {
	Index int32
	Data  MyLogg
	Err   []byte
	Exec  bool
}

//Send a Timeout after t milliseconds.
type Alarm struct {
	T int
}

//This is an indication to the node to store the Logg at the given Index.
type LoggStore struct {
	Index int
	Data  []MyLogg
}

//This is an indication to the node to store the state in the memory.
type StateStore struct {
	Data []byte
}

//This deals with the incomming rquest and invokes repective response Event.
type Message interface {
	send(sm *State_Machine)
	commit(sm *State_Machine)
	alarm(sm *State_Machine)
}

//Function for state to become follower.
func (sm *State_Machine) StartFollSys() {
	sm.FollSys()
	sm.EventProcess()
}

func (sm *State_Machine) FollSys() {
	if sm.Status == CAND {
		sm.VotedFor = 1 //Reinitialize VoteFor
	} else {
		sm.VotedFor = 0 //Reinitialize VoteFor
	}
	sm.currLeader = sm.Id
	sm.Status = FOLL //Change state Status to Follower
	sm.VotedFor = 0  //Reinitialize VoteFor
	sm.LeaderId = 0
	sm.VoteGrant[0] = 0 //This is positive VoteGrant counter initialized to 1 i.e. self vote
	sm.VoteGrant[1] = 0 //This is negative VoteGrant counter initialized to 0
	resp := Alarm{T: FTO}
	sm.CommMedium.ActionCh <- resp
}

//Function for state to become CandIdate.
func (sm *State_Machine) candSys() {
	sm.Status = CAND    //Change state Status to CandIdate
	sm.CurrTerm += 1    //Increament the Term
	sm.VotedFor = 1     //Vote for self
	sm.VoteGrant[0] = 1 //This is positive VoteGrant counter initialized to 1 i.e. self vote
	sm.VoteGrant[1] = 0 //This is negative VoteGrant counter initialized to 0
	sm.LeaderId = 0
	resp := Alarm{T: CTO} //150
	sm.CommMedium.ActionCh <- resp
	//Sending vote request null information of previous entry as CandIdate had just joined the cluster.
	if len(sm.Logg.Logg) == 0 {
		respp := Send{PeerId: 0, Event: VoteReq{Term: sm.CurrTerm, CandId: sm.Id, PreLoggInd: 0, PreLoggTerm: 0}}
		sm.CommMedium.ActionCh <- respp
	} else {
		//Send vote request to all other servers.
		respp := Send{PeerId: 0, Event: VoteReq{Term: sm.CurrTerm, CandId: sm.Id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term}}
		sm.CommMedium.ActionCh <- respp
	}
}

//Function for state to become leader.
func (sm *State_Machine) leadSys() {
	sm.Status = LEAD    //Change state Status to leader
	sm.VotedFor = 0     //Reinitialize VoteFor
	sm.VoteGrant[0] = 0 //This is positive VoteGrant counter initialized to 1 i.e. self vote
	sm.VoteGrant[1] = 0 //This is negative VoteGrant counter initialized to 0
	sm.initialize()     //initialize MatchIndex and nextIndex
	sm.LeaderId = sm.Id
	//At begining commitIndex is set to -1.
	if len(sm.Logg.Logg) == 0 {
		sm.CommitIndex = -1
	}
	resp := Alarm{T: LTO} //175
	//Set heartbeat timeout
	sm.CommMedium.ActionCh <- resp
	//Send heartbeat msg to all other servers
	//PeerId:0 means to all servers.
	if len(sm.Logg.Logg) == 0 {
		respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.CurrTerm, LeaderId: sm.Id, PreLoggInd: 0, PreLoggTerm: 0, LeaderCom: 0}}
		sm.CommMedium.ActionCh <- respp
	} else {
		respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.CurrTerm, LeaderId: sm.Id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term, LeaderCom: sm.CommitIndex}}
		sm.CommMedium.ActionCh <- respp
	}
}

//This will keep listening to all incomming channels and procceed the request as it arrives.
func (sm *State_Machine) EventProcess() {
	var msg Message
	for {
		select {
		//Shutdown Server.
		case <-sm.CommMedium.ShutdownCh:
			break

		//Timeout Event.
		case <-sm.CommMedium.TimeoutCh:
			//Generate corrosponding response to the request.
			switch sm.Status {
			case FOLL:
				sm.candSys()

			case CAND:
				//Start election for next Term again.
				sm.candSys()

			case LEAD:
				//Commit the Logg and send heartbeat msg to all other servers.
				sm.commitLogg()
				if len(sm.Logg.Logg) == 0 {
					respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.CurrTerm, LeaderId: sm.Id, PreLoggInd: 0, PreLoggTerm: 0, LeaderCom: 0}}
					sm.CommMedium.ActionCh <- respp
					resp := Alarm{T: LTO} //175
					//Set heartbeat timeout
					sm.CommMedium.ActionCh <- resp
				} else {
					respp := Send{PeerId: 0, Event: AppEntrReq{Term: sm.CurrTerm, LeaderId: sm.Id, PreLoggInd: sm.LoggInd - 1, PreLoggTerm: sm.Logg.Logg[sm.LoggInd-1].Term, LeaderCom: sm.CommitIndex}}
					sm.CommMedium.ActionCh <- respp
					resp := Alarm{T: LTO}
					//Set heartbeat timeout
					sm.CommMedium.ActionCh <- resp
				}
			}

		//Requests from client machine.
		case appendMsg := <-sm.CommMedium.ClientCh:
			msg = appendMsg.(Append)
			msg.commit(sm)

		//Request from PEERS in the cluster.
		case PeerMsg := <-sm.CommMedium.NetCh:
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
				msg = PeerMsg.(VoteResp)
				msg.send(sm)
			}
		}
	}
}

//Process incommimg append entry request.
//Incoming Logg or any incoming variable means the Logg or variable form the given incomming request or respoonse msg.
func (appReq AppEntrReq) send(sm *State_Machine) {
	sm.currLeader = appReq.LeaderId
	if sm.Id == 1 {
	}
	switch sm.Status {
	//For every incoming signal from leader to follower reset the timeout time as leader is still alive.
	case FOLL:
		if len(appReq.Logg.Logg) == 0 {
			//Reset the timeout timer.
			preCommitInd := sm.CommitIndex
			sm.CommitIndex = int32(math.Min(float64(appReq.LeaderCom), float64(sm.LoggInd-1))) //update commit index
			sm.CurrTerm = appReq.Term

			respp := Alarm{T: FTO}
			sm.CommMedium.ActionCh <- respp
			sm.commitFollLogg(preCommitInd)
			return
		}

		//Send regative reply, if incoming Term is lower than local Term or previous Index does not match.
		if (sm.CurrTerm > appReq.Term) || ((appReq.PreLoggInd > sm.LoggInd-1) && len(sm.Logg.Logg) != 0) {
			resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Peer: sm.Id, Term: sm.CurrTerm, Succ: false, MyInd: sm.LoggInd - 1, YourInd: appReq.PreLoggInd}}
			sm.CommMedium.ActionCh <- resp
			respp := Alarm{T: FTO}
			sm.CommMedium.ActionCh <- respp
			return
		}

		//Send regative reply, if previous index matches, but term of previous entry does not match.   , (!reflect.DeepEqual(sm.Logg.Logg[appReq.PreLoggInd], appReq.Logg.Logg[0]))
		if ((appReq.PreLoggInd == sm.LoggInd-1) && (len(sm.Logg.Logg) != 0)) && (appReq.PreLoggTerm != sm.Logg.Logg[sm.LoggInd-1].Term) {
			resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Peer: sm.Id, Term: sm.CurrTerm, Succ: false, MyInd: sm.LoggInd - 1, YourInd: appReq.PreLoggInd}}
			sm.CommMedium.ActionCh <- resp
			respp := Alarm{T: FTO}
			sm.CommMedium.ActionCh <- respp
			return
		}

		//Ignore duplicate append entry request.
		if (len(sm.Logg.Logg) != 0) && sm.LoggInd-1 > appReq.PreLoggInd {
			if sm.Logg.Logg[appReq.PreLoggInd].Term == appReq.PreLoggTerm && sm.CommitIndex >= appReq.PreLoggInd {
				respp := Alarm{T: FTO}
				sm.CommMedium.ActionCh <- respp
				return
			} else {
				respp := Alarm{T: FTO}
				sm.CommMedium.ActionCh <- respp
			}
		}

		preCommitInd := sm.CommitIndex
		sm.CommitIndex = int32(math.Min(float64(appReq.LeaderCom), float64(sm.LoggInd-1))) //update commit index
		sm.CurrTerm = appReq.Term                                                          //update term
		//Copy incoming Logg into local Logg.
		if len(sm.Logg.Logg) == 0 {
			sm.LoggInd, sm.Logg = copyLogg(sm.CurrTerm, sm.LoggInd, 0, sm.Logg, appReq.Logg)
		} else {
			sm.LoggInd, sm.Logg = copyLogg(sm.CurrTerm, sm.LoggInd, appReq.PreLoggInd, sm.Logg, appReq.Logg)
		}
		//Send possitive reply, as Logg has been copied to local Logg.
		resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Peer: sm.Id, Term: sm.CurrTerm, Succ: true, MyInd: sm.LoggInd - 1}}
		sm.CommMedium.ActionCh <- resp
		//Reset the timeout timer.
		respp := Alarm{T: FTO}
		sm.CommMedium.ActionCh <- respp

		//Insert data into log.
		prevInd := sm.LoggInd - int32(len(appReq.Logg.Logg)) //index on which data was stored.
		sm.ActionCh <- LoggStore{Index: int(prevInd), Data: (sm.Logg.Logg[prevInd:])}
		sm.commitFollLogg(preCommitInd)

	case CAND:
		//Become follower and amd process incomming append entry request, if incomming Term higher than or eqaul(already have Term) to local Term.
		if appReq.Term >= sm.CurrTerm {
			sm.Status = FOLL
			sm.VotedFor = 0
			sm.CurrTerm = appReq.Term
			appReq.send(sm)
			return
		}
		//Reply negative if incomming Term is lower.
		resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.CurrTerm, Succ: false}}
		sm.CommMedium.ActionCh <- resp

	case LEAD:
		//Become a follower, if incomming Term higher than local Term.
		if appReq.Term > sm.CurrTerm {
			if appReq.PreLoggInd < sm.LoggInd-1 {
				resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.CurrTerm, Succ: false, MyInd: sm.LoggInd - 1, YourInd: appReq.PreLoggInd}}
				sm.CommMedium.ActionCh <- resp
				sm.CurrTerm = appReq.Term
				sm.FollSys()
			}
			resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.CurrTerm, Succ: false}}
			sm.CommMedium.ActionCh <- resp
			return
		}
		//Reply negative if incomming Term is lower.
		resp := Send{PeerId: appReq.LeaderId, Event: AppEntrResp{Term: sm.CurrTerm, Succ: false}}
		sm.CommMedium.ActionCh <- resp
	}
	return
}

//Process incommimg append entry response.
func (appRes AppEntrResp) send(sm *State_Machine) {
	switch sm.Status {
	case LEAD:
		//On positive response, update MatchIndex and NextIndex.
		if appRes.Succ == true {
			sm.MatchIndex[appRes.Peer-1] = sm.LoggInd - 1
			sm.NextIndex[appRes.Peer-1] = sm.LoggInd
		}
		//On negative response, decreament the NextIndex with respect to incomming Peer and resend append entry request.
		if appRes.Succ == false {
			if appRes.Term > sm.CurrTerm {
				sm.CurrTerm = appRes.Term
				sm.FollSys()
			}
			temp := sm.NextIndex[appRes.Peer-1] - 1
			if temp < 0 {
				temp = 0
			}

			temp = int32(math.Min(float64(temp), float64(appRes.MyInd)))

			entry := sm.Logg.Logg[temp+1:]
			entry1 := Logg{Logg: entry}
			//Check for Logg to be commited.
			resp := Send{PeerId: appRes.Peer, Event: AppEntrReq{Term: sm.CurrTerm, LeaderId: sm.Id, PreLoggInd: temp, PreLoggTerm: sm.Logg.Logg[temp].Term, LeaderCom: sm.CommitIndex, Logg: entry1}}
			sm.CommMedium.ActionCh <- resp
		}
		sm.commitLogg()
	}
}

//Process incommimg vote request.
func (votReq VoteReq) send(sm *State_Machine) {
	switch sm.Status {
	case FOLL:
		//If CandIdate Logg is not uptodate or incoming Term is lower or already voted in given Term, then reply negative.
		if votReq.Term < sm.CurrTerm || sm.VotedFor != 0 || votReq.PreLoggInd < sm.LoggInd-1 {
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.CurrTerm, VoteGrant: false}}
			sm.CommMedium.ActionCh <- resp
			return
		}
		//Vote to incomming CandIdate and set the VotedFor to 1.
		sm.VotedFor = 1
		sm.CurrTerm = votReq.Term
		resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.CurrTerm, VoteGrant: true}}
		sm.CommMedium.ActionCh <- resp
		respp := Alarm{T: FTO}
		sm.CommMedium.ActionCh <- respp

	case CAND:
		if sm.CurrTerm < votReq.Term {
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.CurrTerm, VoteGrant: true}}
			sm.CommMedium.ActionCh <- resp
			sm.CurrTerm = votReq.Term
			sm.FollSys()
			return
		}
		//Reject the incomming vote Request.
		resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.CurrTerm, VoteGrant: false}}
		sm.CommMedium.ActionCh <- resp

	case LEAD:
		sm.CurrTerm = votReq.Term
		//Reply negative  in any case for vote request.
		if sm.CurrTerm > votReq.Term {
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.CurrTerm, VoteGrant: false}}
			sm.CommMedium.ActionCh <- resp
		}
		//But if incomming Term is higher than local, then step down to follower state.
		if sm.CurrTerm < votReq.Term {
			if votReq.PreLoggInd > sm.LoggInd-1 {
				resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.CurrTerm, VoteGrant: false}}
				sm.CommMedium.ActionCh <- resp
				sm.CurrTerm = votReq.Term
				sm.FollSys()
			}
			resp := Send{PeerId: votReq.CandId, Event: VoteResp{Term: sm.CurrTerm, VoteGrant: false}}
			sm.CommMedium.ActionCh <- resp
		}
	}
}

//Process incommimg vote response.
func (votRes VoteResp) send(sm *State_Machine) {
	switch sm.Status {
	case CAND:
		//Count incomming positive responses.
		if votRes.VoteGrant == true {
			sm.VoteGrant[0] += 1
		}
		//Count incomming negative responses.
		if votRes.VoteGrant == false {
			sm.VoteGrant[1] += 1
			if votRes.Term > sm.CurrTerm {
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
	switch sm.Status {
	case FOLL:
		//Send Error.
		sm.CommMedium.CommitCh <- Commit{Index: int32(sm.currLeader), Err: []byte(string("ERR_REDIRECT ") + app.Id)}

	case CAND:
		//Send Error.
		sm.CommMedium.CommitCh <- Commit{Index: int32(sm.currLeader), Err: []byte(string("ERR_REDIRECT ") + app.Id)}

	case LEAD:
		prevLogInd := sm.LoggInd - 1
		//Append the commond into local Logg.
		entry := Logg{Logg: []MyLogg{{app.Id, sm.CurrTerm, string(app.Data)}}}
		sm.LoggInd, sm.Logg = storeCmd(sm.CurrTerm, sm.LoggInd, sm.Logg, entry)
		resp := LoggStore{Index: int(prevLogInd) + 1, Data: []MyLogg{{app.Id, sm.CurrTerm, string(app.Data)}}}

		//Send the append entry request to all other servers.
		var respp Send
		if len(sm.Logg.Logg) != 1 {
			respp = Send{PeerId: 0, Event: AppEntrReq{Term: sm.CurrTerm, LeaderId: sm.Id, PreLoggInd: prevLogInd, PreLoggTerm: sm.Logg.Logg[prevLogInd].Term, LeaderCom: sm.CommitIndex, Logg: entry}}
		} else {
			respp = Send{PeerId: 0, Event: AppEntrReq{Term: sm.CurrTerm, LeaderId: sm.Id, PreLoggInd: -1, PreLoggTerm: -1, LeaderCom: sm.CommitIndex, Logg: entry}}
		}
		sm.CommMedium.ActionCh <- resp
		sm.CommMedium.ActionCh <- respp
	}
}

//Commit the Logg, function initiated by leader.
func (sm *State_Machine) commitLogg() {
	for i := sm.CommitIndex + 1; i < sm.LoggInd; i++ {
		if sm.Logg.Logg[i].Term != sm.CurrTerm {
			continue
		}
		count := 0
		for j := 0; j < PEERS; j++ {
			if sm.MatchIndex[j] >= i {
				count += 1
			}
		}
		if count >= MAX-1 {
			if sm.CommitIndex == 0 {
				sm.CommitIndex = 1
			}
			sm.CommitIndex = i
			sm.CommMedium.CommitCh <- Commit{Index: sm.CommitIndex, Data: (sm.Logg.Logg[sm.CommitIndex]), Err: nil, Exec: true}
			break
		}
	}
}

func (sm *State_Machine) commitFollLogg(preCommitInd int32) {
	if preCommitInd < sm.CommitIndex {
		entryComm := sm.Logg.Logg[preCommitInd+1 : sm.CommitIndex+1]
		for i := 0; i < len(entryComm); i++ {
			resppp := CommitInfo{Index: preCommitInd + int32(1) + int32(i), Data: (sm.Logg.Logg[preCommitInd+int32(1)+int32(i)]), Err: nil, Exec: false}
			sm.CommMedium.CommitInfoCh <- resppp
		}
	}
}

//Store client's command into leader's log.
func storeCmd(Term int32, myInd int32, oldLogg Logg, newLogg Logg) (int32, Logg) {
	for i := 0; i < len(newLogg.Logg); i++ {
		oldLogg.Logg = append(oldLogg.Logg, newLogg.Logg[i])
		myInd++
	}
	return myInd, oldLogg
}

//Used to copy Logg from given request to state machine.
func copyLogg(Term int32, myInd int32, preInd int32, oldLogg Logg, newLogg Logg) (int32, Logg) {
	for i := 0; i < len(newLogg.Logg); i++ {
		temp := preInd + int32(i) + int32(1)
		if len(oldLogg.Logg) != 0 {
			oldLogg.Logg = append(oldLogg.Logg[:temp], newLogg.Logg[i])
		} else {
			oldLogg.Logg = append(oldLogg.Logg, newLogg.Logg[i])
		}
	}

	myInd = int32(len(oldLogg.Logg))
	return myInd, oldLogg
}

//Initializing the MatchIndex and NextIndex.
func (sm *State_Machine) initialize() {
	for i := 0; i < PEERS; i++ {
		sm.MatchIndex[i] = 0
		sm.NextIndex[i] = sm.LoggInd
	}
}

//Returns respond to given request.
func (appReq AppEntrReq) alarm(sm *State_Machine)    {}
func (appResp AppEntrResp) alarm(sm *State_Machine)  {}
func (votReq VoteReq) alarm(sm *State_Machine)       {}
func (votResp VoteResp) alarm(sm *State_Machine)     {}
func (app Append) alarm(sm *State_Machine)           {}
func (appReq AppEntrReq) commit(sm *State_Machine)   {}
func (appResp AppEntrResp) commit(sm *State_Machine) {}
func (votReq VoteReq) commit(sm *State_Machine)      {}
func (votResp VoteResp) commit(sm *State_Machine)    {}
func (to Timeout) commit(sm *State_Machine)          {}
func (app Append) send(sm *State_Machine)            {}
func (to Timeout) send(sm *State_Machine)            {}
func (to Timeout) alarm(sm1 *State_Machine)          {}
