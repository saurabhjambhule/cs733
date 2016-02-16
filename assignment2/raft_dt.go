package main

const (
	FOLL = "follower"
	CAND = "candidate"
	LEAD = "leader"
)

//Contains persistent state of all servers.
type Persi_State struct {
	id       uint32
	currTerm uint32
	votedFor uint32
	log      Log
}

//Contains volatile state of servers.
type Volat_State struct {
	commitIndex int32
	lastApplied int32
	timer       int32
	status      string
	logInd      int32
}

//Contains volatile state of the leader.
type Volat_LState struct {
	nextIndex  int32
	matchIndex []int32
}

type MyLog struct {
	term uint32
	log  string
}

type Log struct {
	log []MyLog
}

//Contains all the state with respect to given machine.
type Leader struct {
	Persi_State
	Volat_LState
	Volat_State
}

type Follower struct {
	Persi_State
	Volat_State
}

type Candidate struct {
	Persi_State
	Volat_State
}

//AppendEntriesRequest: Invoked by leader to replicate log entries and also used as heartbeat.
type AppEntrReq struct {
	term       uint32
	leaderId   uint32
	preLogInd  int32
	preLogTerm uint32
	leaderCom  int32
	log        Log
}

//AppendEntriesResponse: Invoked by servers on AppendEntriesRequest.
type AppEntrResp struct {
	term uint32
	succ bool
}

//VoteRequest: Invoked by candidates to gather votes.
type VoteReq struct {
	term       uint32
	candId     uint32
	preLogInd  int32
	preLogTerm int32
}

//VoteResponse: Invoked by servers on VoteRequest.
type VoteResp struct {
	term      uint32
	voteGrant bool
}

//This is a request from the layer above to append the data to the replicated log.
type Append struct {
	data []byte
}

//A timeout event.
type Timeout struct {
}

//Send this event to a remote node.
type Send struct {
	peerId []uint32
	event  interface{}
}

//Invoked by the leader on Append request. Provides (index + data) or report an error (data + err) to the layer above.
type Commit struct {
	index int32
	data  []byte
	err   []byte
}

//Send a Timeout after t milliseconds.
type Alarm struct {
	t uint32
}

//This is an indication to the node to store the data at the given index.
type LogStore struct {
	index int32
	data  []byte
}

//Returns respond to given request.
type Reply struct {
	//Maps different output event structures.
	ack interface{}
}

func (appReq AppEntrReq) alarm(sm State_Machine)    {}
func (appResp AppEntrResp) alarm(sm State_Machine)  {}
func (votReq VoteReq) alarm(sm State_Machine)       {}
func (votResp VoteResp) alarm(sm State_Machine)     {}
func (app Append) alarm(sm State_Machine)           {}
func (appReq AppEntrReq) commit(sm State_Machine)   {}
func (appResp AppEntrResp) commit(sm State_Machine) {}
func (votReq VoteReq) commit(sm State_Machine)      {}
func (votResp VoteResp) commit(sm State_Machine)    {}
func (to Timeout) commit(sm State_Machine)          {}
func (app Append) send(sm State_Machine)            {}
func (to Timeout) send(sm State_Machine)            {}
