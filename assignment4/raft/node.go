package raft

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/cluster/mock"
	"github.com/cs733-iitb/log"
	"github.com/saurabhjambhule/cs733/assignment4/raft/sm"
)

const (
	FTIME       = 3000 //seconds
	CTIME       = 3000 //seconds(re-election)
	LTIME       = 1000 //milliseconds(haertbeat)
	RANGE       = 1000 //timeout upperlimit in seconds for candIdate and follower
	TESTENTRIES = 1000
)
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

//Contains raft node related data .
type Config struct {
	Id     int    //this node's Id. One of the cluster's entries should match
	LogDir string //Log file directory for this node
	StDir  string //State file directory for this node

	ElectionTimeout  int
	HeartbeatTimeout int
	DoTO             *time.Timer  //timeout the state after DoTO
	DoStore          *time.Ticker //store state after tick
	Lg               *log.Log
	St               *log.Log

	TimeLock    sync.RWMutex
	ProcessLock sync.RWMutex
}

//Contains all raft node's data of entire cluster.
type MyConfig struct {
	Details []Config
}

//Contains server related data.
type RaftMachine struct {
	Node cluster.Server
	SM   *sm.State_Machine
	Conf *Config
	sync.RWMutex
}

//Contains data of entire cluster.
//and the channel through which client communicate to raft.
type Raft struct {
	Cluster []*RaftMachine
	sync.RWMutex
}

type incomming interface {
}

//Current Id of leader. -1 if unknown
func (myRaft Raft) LeaderId() int {
	for i := 0; i < PEERS; i++ {
		if myRaft.Cluster[i].SM.Status == LEAD {
			return i + 1
		}
	}
	return -1
}

//Blocks until leader electected and return leader Id.
func (myRaft Raft) GetLeader() int {
	myRaft.RLock()
	for {
		fmt.Print("")
		for i := 0; i < PEERS; i++ {
			if myRaft.Cluster[i].SM.Status == LEAD {
				//getLock.Unlock()
				return i
			}
		}
	}
	myRaft.RUnlock()
	return -1
}

//Signal to shut down all goroutines, stop sockets, flush log and close it, cancel timers.
func (server RaftMachine) Shutdown(cl *mock.MockCluster) {
	server.SM.Status = FOLL
	cl.Servers[int(server.SM.Id)].Close()
	resp := sm.Alarm{T: FTO} //200
	server.SM.CommMedium.ActionCh <- resp
	server.SM.VotedFor = 0 //Reinitialize VoteFor
	server.SM.LeaderId = 0
	server.SM.VoteGrant[0] = 0 //This is positive VoteGrant counter initialized to 1 i.e. self vote
	server.SM.VoteGrant[1] = 0 //This is negative VoteGrant counter initialized to 0
	for i := 0; i < 1; i++ {
		server.SM.CommMedium.ShutdownCh <- nil
	}
	//server.SM.FollSys()
}

//Client's message to Raft node.
func ClientAppend(server *RaftMachine, cliId string, cmdReq []byte) {
	reqApp := sm.Append{Id: cliId, Data: cmdReq}
	//fmt.Println(">->", reqApp)

	server.SM.CommMedium.ClientCh <- reqApp
}

//Client's message to Raft node For TESTING
func (server RaftMachine) Append(cmdReq []byte) {
	reqApp := sm.Append{Data: cmdReq}
	server.SM.CommMedium.ClientCh <- reqApp
}

//Process incoming events from StateMachine.
func processEvents(server cluster.Server, SM *sm.State_Machine, myConf *Config) {
	var incm incomming
	for {
		select {
		case incm := <-SM.CommMedium.ActionCh:
			switch incm.(type) {
			case sm.Send:
				msg := incm.(sm.Send)
				processOutbox(server, SM, msg)

			case sm.Alarm:
				//Reset the timer of timeout.
				myConf.TimeLock.Lock()
				myConf.DoTO.Stop()
				al := incm.(sm.Alarm)
				if al.T == LTO {
					myConf.DoTO = time.AfterFunc(time.Duration(LTIME)*time.Millisecond, func() {
						myConf.DoTO.Stop()
						SM.CommMedium.TimeoutCh <- nil
					})
				}
				if al.T == CTO {
					myConf.ElectionTimeout = (CTIME + rand.Intn(RANGE))
					myConf.DoTO = time.AfterFunc(time.Duration(myConf.ElectionTimeout)*time.Millisecond, func() {
						myConf.DoTO.Stop()
						SM.CommMedium.TimeoutCh <- nil
					})
				}
				if al.T == FTO {
					myConf.ElectionTimeout = (FTIME + rand.Intn(RANGE))
					if SM.Id == 1 {
						myConf.ElectionTimeout = 1000
					}
					myConf.DoTO = time.AfterFunc(time.Duration(myConf.ElectionTimeout)*time.Millisecond, func() {
						myConf.DoTO.Stop()
						SM.CommMedium.TimeoutCh <- nil
					})
				}
				myConf.TimeLock.Unlock()

			case sm.Commit:

			case sm.LoggStore:
				//for adding log into db.
				msg := incm.(sm.LoggStore)
				//fmt.Println("---->>>", msg)
				storeData(msg.Data, myConf, msg.Index)

			case sm.StateStore:
			}

			//		case incm := <-SM.CommMedium.RecoveryCh:
			//			msg := incm.(sm.Send)
			//			processOutbox(server, SM, msg)
		}
	}
	fmt.Println("Bye", incm)
}

//Process to listen incomming packets from other Servers.
func processInbox(server cluster.Server, SM *sm.State_Machine) {
	for {
		env := <-server.Inbox()
		//fmt.Printf("\n-%T", env.Msg)
		switch env.Msg.(type) {
		case sm.VoteReq:
			SM.CommMedium.NetCh <- env.Msg
		case sm.VoteResp:
			SM.CommMedium.NetCh <- env.Msg
		case sm.AppEntrReq:
			SM.CommMedium.NetCh <- env.Msg
		case sm.AppEntrResp:
			SM.CommMedium.NetCh <- env.Msg
		}
	}
}

//Process to send packets to other Servers.
func processOutbox(server cluster.Server, SM *sm.State_Machine, msg sm.Send) {
	//broadcast messagess.
	if msg.PeerId == 0 {
		switch msg.Event.(type) {
		case sm.AppEntrReq:
			AppReq := msg.Event.(sm.AppEntrReq)
			server.Outbox() <- &cluster.Envelope{Pid: cluster.BROADCAST, Msg: AppReq}
		case sm.VoteReq:
			VotReq := msg.Event.(sm.VoteReq)
			server.Outbox() <- &cluster.Envelope{Pid: cluster.BROADCAST, Msg: VotReq}
		}
	} else {
		//send to particular node.
		switch msg.Event.(type) {
		case sm.AppEntrReq:
			AppReq := msg.Event.(sm.AppEntrReq)
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), Msg: AppReq}
		case sm.AppEntrResp:
			AppResp := msg.Event.(sm.AppEntrResp)
			AppResp.Peer = int32(server.Pid())
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), Msg: AppResp}
		case sm.VoteResp:
			VotResp := msg.Event.(sm.VoteResp)
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), Msg: VotResp}
		}
	}
}

//Storing log entries in database.
func storeData(data []sm.MyLogg, myConf *Config, ind int) {
	myConf.Lg.TruncateToEnd(int64(ind))
	for i := 0; i < len(data); i++ {
		err := myConf.Lg.Append(data[i])
		if err != nil {
			fmt.Println("error:", err)
		}
	}
}

func restoreData(SM *sm.State_Machine, myConf *Config) {
	//restore persistant state
	last := myConf.St.GetLastIndex()
	if last != -1 {
		state, err := myConf.St.Get(last)
		if err != nil {
			fmt.Println("error:-", err)
		}
		data := state.(sm.Persi_State)
		SM.Persi_State = data
	}

	//restore log
	last = myConf.Lg.GetLastIndex()
	if last != -1 {
		for i := int64(0); i <= last; i++ {
			log, err := myConf.St.Get(i)
			if err != nil {
				fmt.Println("error:-", err)
			}
			logg := log.(sm.MyLogg)
			SM.Logg.Logg[i] = logg
		}
	}
}

//Configuration of Log and Node.
func logConfig(myId int, myConf *Config) {
	var conf MyConfig
	file, errr := os.Open("config/log_config.json")
	if errr != nil {
		fmt.Println("+error:", errr)
	}
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&conf)
	if err != nil {
		fmt.Println("-error:", err)
	}
	foundMyId := false
	//initializing config structure from jason file.
	for _, srv := range conf.Details {
		if srv.Id == myId {
			foundMyId = true
			myConf.Id = myId
			myConf.LogDir = srv.LogDir
			myConf.StDir = srv.StDir
			myConf.ElectionTimeout = srv.ElectionTimeout
			myConf.HeartbeatTimeout = srv.HeartbeatTimeout
		}
	}
	if !foundMyId {
		fmt.Println("Expected this server's Id (\"%d\") to be present in the configuration", myId)
	}
}

func initNode(Id int, myConf *Config, SM *sm.State_Machine) {
	//Register a struct name by giving it a dummy object of that name.
	gob.Register(sm.AppEntrReq{})
	gob.Register(sm.AppEntrResp{})
	gob.Register(sm.VoteReq{})
	gob.Register(sm.VoteResp{})
	gob.Register(sm.StateStore{})
	gob.Register(sm.LoggStore{})
	gob.Register(sm.CommitInfo{})
	gob.Register(sm.MyLogg{})
	gob.Register(sm.Commit{})
	gob.Register(sm.Append{})
	gob.Register(sm.Persi_State{})

	//Channel initialization.
	SM.CommMedium.ClientCh = make(chan interface{}, 10000)
	SM.CommMedium.NetCh = make(chan interface{}, 10000)
	SM.CommMedium.TimeoutCh = make(chan interface{}, 10000)
	SM.CommMedium.ActionCh = make(chan interface{}, 10000)
	SM.CommMedium.ShutdownCh = make(chan interface{}, 10000)
	SM.CommMedium.CommitInfoCh = make(chan interface{}, 10000)
	SM.CommMedium.CommitCh = make(chan interface{}, 10000)

	//Seed randon number generator.
	rand.Seed(time.Now().UTC().UnixNano() * int64(Id))
	//Initialize the timer object for timeuts.
	myConf.DoTO = time.AfterFunc(10, func() {})

	//Store persitant state
	myConf.DoStore = time.NewTicker(3 * time.Second)
	go func() {
		for _ = range myConf.DoStore.C {
			last := myConf.St.GetLastIndex()
			if last == -1 {
				err := myConf.St.Append(SM.Persi_State)
				if err != nil {
					fmt.Println("error:-", err)
				}
				err = myConf.St.Append(SM.Persi_State)
				if err != nil {
					fmt.Println("error:-", err)
				}
			}
			last = myConf.St.GetLastIndex()
			myConf.St.TruncateToEnd(last)

			err := myConf.St.Append(SM.Persi_State)
			if err != nil {
				fmt.Println("error:-", err)
			}

			last = myConf.St.GetLastIndex()
			state, err := myConf.St.Get(last)
			if err != nil {
				fmt.Println("error:-", err)
			}
			data := state.(sm.Persi_State)
			SM.Persi_State = data
		}
	}()

	//Initialize the Log and Node configuration.
	logConfig(Id, myConf)
	var err error
	myConf.Lg, err = log.Open(myConf.LogDir)
	if err != nil {
		panic(err)
	}
	myConf.St, err = log.Open(myConf.StDir)
	if err != nil {
		panic(err)
	}

}

func createMockNode(Id int, myConf *Config, SM *sm.State_Machine, cl *mock.MockCluster) cluster.Server {
	initNode(Id, myConf, SM)
	// Give each raftNode its own "Server" from the cluster.
	server, err := cl.AddServer(Id)
	if err != nil {
		panic(err)
	}
	return server
}

//Craetes node, statemachine & Initializes the node.
func createNode(Id int, myConf *Config, SM *sm.State_Machine) cluster.Server {
	initNode(Id, myConf, SM)
	//Set up details about cluster nodes form json file.
	server, err := cluster.New(Id, "config/cluster_config.json")
	if err != nil {
		panic(err)
	}
	return server
}

func startNode(myConf *Config, server cluster.Server, SM *sm.State_Machine) {
	//Start backaground process to listen incomming packets from other servers.
	go processInbox(server, SM)
	//Start StateMachine in follower state.
	go SM.StartFollSys()
	//Raft node Processing.
	myConf.ProcessLock.Lock()
	processEvents(server, SM, myConf)
	myConf.ProcessLock.Unlock()
}

func StartRaft(myId int) *RaftMachine {
	myNode := new(RaftMachine)
	SM := new(sm.State_Machine)
	myConf := new(Config)

	//Start Node.
	server := createNode(myId, myConf, SM)
	SM.Id = int32(myId)

	//Restore previous test.
	restoreData(SM, myConf)

	myNode.SM = SM
	myNode.Conf = myConf
	myNode.Node = server
	go startNode(myConf, server, SM)
	//fmt.Println(myNode.Conf.Lg)
	return myNode
}

/***************---------------------NORMAL SERVER CODE USING PORTS WORKING---------------------**************
func main() {
	flag.Parse()
	//Get Server Id from command line.
	myId, _ := strconv.Atoi(flag.Args()[0])
	startRaft(myId)
}
***************------------------------------------------------------------------------------***************/
