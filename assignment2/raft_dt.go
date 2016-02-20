package main

const (
	FOLL  = "follower"
	CAND  = "candidate"
	LEAD  = "leader"
	PEERS = 5
)

//Contains persistent state of all servers.
type Persi_State struct {
	id        int32
	currTerm  int32
	votedFor  int32
	voteGrant [2]int32
	logInd    int32
	status    string
}

//Contains volatile state of servers.
type Volat_State struct {
	commitIndex int32
	lastApplied int32
}

//Contains volatile state of the leader.
type Volat_LState struct {
	nextIndex  [5]int32
	matchIndex [5]int32
}

//Stores log entries
type MyLog struct {
	term int32
	log  string
}

type Log struct {
	log []MyLog
}

//Stores peers
var peer map[int32]int32

//Contains all the state with respect to given machine.
type State_Machine struct {
	Persi_State
	Volat_State
	Volat_LState
	log Log
}

//AppendEntriesRequest: Invoked by leader to replicate log entries and also used as heartbeat.
type AppEntrReq struct {
	term       int32
	leaderId   int32
	preLogInd  int32
	preLogTerm int32
	leaderCom  int32
	log        Log
}

//AppendEntriesResponse: Invoked by servers on AppendEntriesRequest.
type AppEntrResp struct {
	peer int32
	term int32
	succ bool
}

//VoteRequest: Invoked by candidates to gather votes.
type VoteReq struct {
	term       int32
	candId     int32
	preLogInd  int32
	preLogTerm int32
}

//VoteResponse: Invoked by servers on VoteRequest.
type VoteResp struct {
	term      int32
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
	peerId int32
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
	t int
}

//This is an indication to the node to store the log at the given index.
type LogStore struct {
	index int32
	data  []byte
}

//This is an indication to the node to store the state in the memory.
type StateStore struct {
	data []byte
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
