package main

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/cluster/mock"
	"github.com/cs733-iitb/log"
)

//Node's id
func (server RaftMachine) Id() int32 {
	return server.SM.id
}

//state machines state
func (server RaftMachine) Status() string {
	return server.SM.status
}

//Current Id of leader. -1 if unknown
func (myRaft Raft) LeaderId() int {
	for i := 0; i < PEERS; i++ {
		if myRaft.Cluster[i].SM.status == LEAD {
			return i + 1
		}
	}
	return -1
}

//Blocks until leader electected and return leader id.
func (myRaft Raft) GetLeader() int {
	for {
		for i := 0; i < PEERS; i++ {

			//fmt.Println(">>-- ", myRaft.Cluster[i].SM.id, " -- ", myRaft.Cluster[i].SM.status)

			if myRaft.Cluster[i].SM.status == LEAD {
				return i
			}
		}
	}
	return -1
}

//Returns the data at a log index, or an error.
func (server RaftMachine) Get(index int) ([]byte, []byte) {
	return nil, nil
}

//Signal to shut down all goroutines, stop sockets, flush log and close it, cancel timers.
func Shutdown(server cluster.Server) {
	server.Close()
}

//Client's message to Raft node
func (server RaftMachine) Append(cmdReq []byte) {
	reqApp := Append{Data: cmdReq}
	server.SM.CommMedium.clientCh <- reqApp
}

//Process incoming events from StateMachine.
func processEvents(server cluster.Server, sm *State_Machine, myConf *Config) {
	var incm incomming
	for {
		select {
		case incm := <-sm.CommMedium.actionCh:
			switch incm.(type) {
			case Send:
				msg := incm.(Send)
				//fmt.Println("<-Sending: ", msg)
				processOutbox(server, sm, msg)

			case Alarm:
				al := incm.(Alarm)
				if al.T == LTO {
					myConf.DoTO = time.Now().Add(time.Duration(LTIME) * time.Millisecond)
					myConf.ToFlag = true
				}
				if al.T == CTO {
					myConf.ElectionTimeout = (CTIME + rand.Intn(RANGE))
					myConf.DoTO = time.Now().Add(time.Duration(myConf.ElectionTimeout) * time.Second)
					myConf.ToFlag = true
				}
				if al.T == FTO {
					myConf.ToFlag = true
					myConf.ElectionTimeout = (FTIME + rand.Intn(RANGE))
					myConf.DoTO = time.Now().Add(time.Duration(myConf.ElectionTimeout) * time.Second)
					myConf.ToFlag = true
				}

				/*myConf.DoTO.Stop()
				al := incm.(Alarm)
				if al.T == LTO {
					fmt.Println("  -SET-  ", sm.id, time.Now())

					myConf.DoTO = time.AfterFunc(time.Duration(LTIME)*time.Second, func() {
						myConf.DoTO.Stop()
						fmt.Println("  -TIMEOUT-  ", sm.id, time.Now())

						sm.CommMedium.timeoutCh <- nil
					})
				}
				if al.T == CTO {
					myConf.ElectionTimeout = (CTIME + rand.Intn(RANGE))
					fmt.Println("  -SET-  ", sm.id, time.Now())

					myConf.DoTO = time.AfterFunc(time.Duration(myConf.ElectionTimeout)*time.Second, func() {
						myConf.DoTO.Stop()
						fmt.Println("  -TIMEOUT-  ", sm.id, time.Now())

						sm.CommMedium.timeoutCh <- nil
					})
				}
				if al.T == FTO {
					myConf.ElectionTimeout = (FTIME + rand.Intn(RANGE))
					fmt.Println("  -SET-  ", sm.id, time.Now())

					myConf.DoTO = time.AfterFunc(time.Duration(myConf.ElectionTimeout)*time.Second, func() {
						myConf.DoTO.Stop()
						fmt.Println("  -TIMEOUT-  ", sm.id, time.Now())

						sm.CommMedium.timeoutCh <- nil
					})
				}
				*/
			case Commit:

			case LoggStore:
				msg := incm.(LoggStore)
				//fmt.Println("---> ", msg)
				storeDate(msg.Data, myConf)

			case StateStore:

			}
		}
	}
	fmt.Println("Bye", incm)
}

//Process to listen incomming packets from other Servers.
func processInbox(server cluster.Server, sm *State_Machine) {
	for {
		env := <-server.Inbox()
		//fmt.Printf("[From: %d MsgId:%d] ", env.Pid, env.MsgId)
		//fmt.Println(env.Msg)
		//msg := env.Msg.(MyStructure)
		switch env.Msg.(type) {
		case VoteReq:
			sm.CommMedium.netCh <- env.Msg
		case VoteResp:
			sm.CommMedium.netCh <- env.Msg
		case AppEntrReq:
			sm.CommMedium.netCh <- env.Msg
		case AppEntrResp:
			sm.CommMedium.netCh <- env.Msg
		}
	}
}

//Process to send packets to other Servers.
func processOutbox(server cluster.Server, sm *State_Machine, msg Send) {
	//fmt.Println("*IN*", msg.PeerId)
	//var tmp MyStructure
	if msg.PeerId == 0 {
		switch msg.Event.(type) {
		case AppEntrReq:
			AppReq := msg.Event.(AppEntrReq)
			server.Outbox() <- &cluster.Envelope{Pid: cluster.BROADCAST, MsgId: 11, Msg: AppReq}
		case VoteReq:
			VotReq := msg.Event.(VoteReq)
			server.Outbox() <- &cluster.Envelope{Pid: cluster.BROADCAST, MsgId: 1, Msg: VotReq}
		}
	} else {
		switch msg.Event.(type) {
		case AppEntrReq:
			AppReq := msg.Event.(AppEntrReq)
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), MsgId: 11, Msg: AppReq}
		case AppEntrResp:
			AppResp := msg.Event.(AppEntrResp)
			AppResp.Peer = int32(server.Pid())
			//fmt.Println(":::---", msg.PeerId, AppResp)
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), MsgId: 22, Msg: AppResp}
		case VoteResp:
			VotResp := msg.Event.(VoteResp)
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), MsgId: 2, Msg: VotResp}
		}
	}
}

//Storing log entries in database.
func storeDate(data []MyLogg, myConf *Config) {
	for i := 0; i < len(data); i++ {
		//fmt.Println("--:", data[i])
		err := myConf.lg.Append(data[i])
		if err != nil {
			fmt.Println("error:", err)
		}
	}
	//fmt.Println("#--:", myConf.lg.GetLastIndex())
}

//Configuration of Log and Node.
func logConfig(myid int, myConf *Config) {
	var conf MyConfig
	file, _ := os.Open("log_config.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&conf)
	if err != nil {
		fmt.Println("error:", err)
	}

	foundMyId := false
	for _, srv := range conf.Details {
		if srv.Id == myid {
			foundMyId = true
			myConf.Id = myid
			myConf.LogDir = srv.LogDir
			myConf.ElectionTimeout = srv.ElectionTimeout
			myConf.HeartbeatTimeout = srv.HeartbeatTimeout
		}
	}
	if !foundMyId {
		fmt.Println("Expected this server's id (\"%d\") to be present in the configuration", myid)
	}
}

//Craetes node, statemachine & Initializes the node.
func initNode(id int, myConf *Config, sm *State_Machine) cluster.Server {
	//Register a struct name by giving it a dummy object of that name.
	gob.Register(AppEntrReq{})
	gob.Register(AppEntrResp{})
	gob.Register(VoteReq{})
	gob.Register(VoteResp{})
	gob.Register(StateStore{})
	gob.Register(LoggStore{})
	gob.Register(CommitInfo{})
	gob.Register(MyLogg{})

	//Channel initialization.
	sm.CommMedium.clientCh = make(chan interface{})
	sm.CommMedium.netCh = make(chan interface{})
	sm.CommMedium.timeoutCh = make(chan interface{})
	sm.CommMedium.actionCh = make(chan interface{})
	sm.CommMedium.CommitCh = make(chan interface{})

	rand.Seed(time.Now().UTC().UnixNano() * int64(id))
	//Set up details about cluster nodes form json file.
	server, err := cluster.New(id, "cluster_config.json")
	if err != nil {
		panic(err)
	}
	//Initialize the Log and Node configuration.
	logConfig(id, myConf)
	myConf.lg, err = log.Open(myConf.LogDir)
	if err != nil {
		panic(err)
	}
	//fmt.Printf("---: %T \n", myConf.lg)
	myConf.DoTO = time.Now().Add(time.Duration(FTIME) * time.Minute)

	//	myConf.DoTO = time.AfterFunc(10, func() {})

	return server
}

//Trigger Timeouts for State.
func processTO(myConf *Config, sm *State_Machine) {
	for {
		fmt.Print("")
		//Send timeout to State after dur milliseconds.
		if myConf.ToFlag {
			for {
				if myConf.DoTO.Before(time.Now()) {
					//fmt.Println("--->>", myConf.Id)

					//fmt.Println("Timeout Called")

					sm.CommMedium.timeoutCh <- nil
					//fmt.Println("##***>>", sm.id, sm.CommMedium)
					myConf.ToFlag = false

					break
				}
			}
		}
	}
}

func (myRaft Raft) startNode(myConf *Config, server cluster.Server, sm *State_Machine) {
	go processTO(myConf, sm)
	//Start backaground process to listen incomming packets from other servers.
	go processInbox(server, sm)
	//Start StateMachine in follower state.
	go sm.FollSys()
	//Raft node Processing.
	processEvents(server, sm, myConf)
}

func (myRaft Raft) makeRafts() Raft {
	//myRaft.CommitInfo = make(chan interface{})
	for id := 1; id <= PEERS; id++ {
		//fmt.Println(id)
		myNode := new(RaftMachine)
		sm := new(State_Machine)
		myConf := new(Config)
		server := initNode(id, myConf, sm)
		sm.id = int32(id)
		myNode.Node = server
		myNode.SM = sm
		myNode.Conf = myConf
		myRaft.Cluster = append(myRaft.Cluster, myNode)
		go myRaft.startNode(myRaft.Cluster[id-1].Conf, myRaft.Cluster[id-1].Node, myRaft.Cluster[id-1].SM)
	}
	return myRaft
}

func initMockNode(id int, myConf *Config, sm *State_Machine, cl *mock.MockCluster) cluster.Server {
	//Register a struct name by giving it a dummy object of that name.
	gob.Register(AppEntrReq{})
	gob.Register(AppEntrResp{})
	gob.Register(VoteReq{})
	gob.Register(VoteResp{})
	gob.Register(StateStore{})
	gob.Register(LoggStore{})
	gob.Register(CommitInfo{})
	gob.Register(MyLogg{})

	//Channel initialization.
	sm.CommMedium.clientCh = make(chan interface{}, 50)
	sm.CommMedium.netCh = make(chan interface{}, 50)
	sm.CommMedium.timeoutCh = make(chan interface{}, 50)
	sm.CommMedium.actionCh = make(chan interface{}, 50)
	sm.CommMedium.CommitCh = make(chan interface{}, 50)

	rand.Seed(time.Now().UTC().UnixNano() * int64(id))

	// Give each raftNode its own "Server" from the cluster.
	server, err := cl.AddServer(id)
	if err != nil {
		panic(err)
	}

	//Initialize the Log and Node configuration.
	//var err error
	logConfig(id, myConf)
	myConf.lg, err = log.Open(myConf.LogDir)
	if err != nil {
		panic(err)
	}
	myConf.DoTO = time.Now().Add(time.Duration(FTIME) * time.Minute)

	//myConf.DoTO = time.AfterFunc(10, func() {})
	return server
}

func (myRaft Raft) makeMockRafts() (Raft, *mock.MockCluster) {
	//cl := new(mock.MockCluster)
	clconfig := cluster.Config{Peers: nil}
	//fmt.Println("-->", clconfig)
	cl, err := mock.NewCluster(clconfig)
	if err != nil {
		panic(err)
	}
	//myRaft.CommitInfo = make(chan interface{})
	//fmt.Printf("---> %T \n", cl)
	for id := 1; id <= PEERS; id++ {
		myNode := new(RaftMachine)
		sm := new(State_Machine)
		myConf := new(Config)

		server := initMockNode(id, myConf, sm, cl)
		//fmt.Println("---->", server)

		sm.id = int32(id)
		myNode.Node = server
		myNode.SM = sm
		myNode.Conf = myConf
		myRaft.Cluster = append(myRaft.Cluster, myNode)
		go myRaft.startNode(myRaft.Cluster[id-1].Conf, myRaft.Cluster[id-1].Node, myRaft.Cluster[id-1].SM)
	}
	return myRaft, cl
}

func main() {
	myRaft := new(Raft)
	myNode := new(RaftMachine)
	sm := new(State_Machine)
	myConf := new(Config)
	flag.Parse()
	//Get Server id from command line.
	myId, _ := strconv.Atoi(flag.Args()[0])
	//Start Node.
	server := initNode(myId, myConf, sm)
	sm.id = int32(myId)
	myNode.Node = server
	myNode.SM = sm
	myNode.Conf = myConf
	myRaft.Cluster = append(myRaft.Cluster, myNode)
	myRaft.startNode(myRaft.Cluster[0].Conf, myRaft.Cluster[0].Node, myRaft.Cluster[0].SM)
}
