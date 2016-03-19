package main

import (
	"time"

	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/log"
)

//------------------------||RAFT NODE DS||------------------------//
//Constant for timeout.
const (
	FTIME = 5   //seconds
	CTIME = 3   //seconds(re-election)
	LTIME = 100 //milliseconds(haertbeat)
	RANGE = 3   //timeout upperlimit in seconds for candidate and follower
)

type CommitInfo struct {
	Data  []byte
	Index int32
	Err   error
}

//Contains raft node related data .
type Config struct {
	Id               int    //this node's id. One of the cluster's entries should match
	LogDir           string //Log file directory for this node
	ElectionTimeout  int
	HeartbeatTimeout int
	DoTO             *time.Timer //timeout the state after DoTO
	lg               *log.Log
}

//Contains all raft node's data of entire cluster.
type MyConfig struct {
	Details []Config
}

//Contains server related data.
type RaftMachine struct {
	Node cluster.Server
	SM   *State_Machine
	Conf *Config
}

//Contains data of entire cluster.
//and the channel through which client communicate to raft.
type Raft struct {
	Cluster    []*RaftMachine
	CommitInfo chan interface{}
}

type incomming interface {
}

//----------------------||STATE MACHINE DS||----------------------//
const (
	FOLL  = "follower"
	CAND  = "Candidate"
	LEAD  = "leader"
	PEERS = 5
	MAX   = 3
	FTO   = 0
	CTO   = 1
	LTO   = 2
)

//Contains persistent state of all servers.
type Persi_State struct {
	id        int32
	currTerm  int32
	votedFor  int32
	VoteGrant [2]int32
	LoggInd   int32
	status    string
	LeaderId  int32
}

//Contains volatile state of servers.
type Volat_State struct {
	CommitIndex int32
	LastApplied int32
}

//Contains volatile state of the leader.
type Volat_LState struct {
	NextIndex  [PEERS]int32
	MatchIndex [PEERS]int32
}

//Stores Logg entries
type MyLogg struct {
	Term int32
	Logg string
}

type Logg struct {
	Logg []MyLogg
}

//Stores PEERS
//var Peer map[int32]int32

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
	clientCh  chan interface{}
	netCh     chan interface{}
	timeoutCh chan interface{}
	//Channel for providing respond to given request.
	actionCh chan interface{}
	CommitCh chan interface{}
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
	Peer int32
	Term int32
	Succ bool
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
	Data []byte
}

//A timeout Event.`
type Timeout struct {
}

//Send this Event to a remote node.
type Send struct {
	PeerId int32
	Event  interface{}
}

//Invoked by the leader on Append request. Provides (Index + data) or report an error (data + err) to the layer above.
type Commit struct {
	Index int32
	Data  []byte
	Err   []byte
}

//Send a Timeout after t milliseconds.
type Alarm struct {
	T int
}

//This is an indication to the node to store the Logg at the given Index.
type LoggStore struct {
	//Index int32
	Data []MyLogg
}

//This is an indication to the node to store the state in the memory.
type StateStore struct {
	Data []byte
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
