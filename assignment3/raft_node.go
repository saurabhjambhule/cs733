package main

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/cs733-iitb/cluster"
	//"github.com/cs733-iitb/log"
)

type Config struct {
	Id               int    // this node's id. One of the cluster's entries should match.
	LogDir           string // Log file directory for this node
	ElectionTimeout  int
	HeartbeatTimeout int
	DoTO             time.Time
	toFlag           bool
}

type MyConfig struct {
	Details []Config
}

type incomming interface {
}

//Process incoming events from StateMachine.
func processEvents(server cluster.Server, sm *State_Machine, myConf *Config) {
	var incm incomming
	/*lg, err := log.Open(myConf.LogDir)
	lg.RegisterSampleEntry(MyLogg{})
	assert(err == nil)
	defer lg.Close()*/

	for {
		//fmt.Println("In eventProcess Node")
		select {
		case incm := <-actionCh:
			switch incm.(type) {
			case Send:
				msg := incm.(Send)
				//fmt.Println("<-Sending: ", msg)
				processOutbox(server, msg)

			case Alarm:
				//fmt.Println("Timeout Init")
				al := incm.(Alarm)
				now := time.Now()
				if al.T == LTO {
					myConf.DoTO = now.Add(time.Duration(myConf.HeartbeatTimeout) * time.Millisecond)
					myConf.toFlag = true
				} else {
					myConf.ElectionTimeout = random()
					myConf.DoTO = now.Add(time.Duration(myConf.ElectionTimeout) * time.Millisecond)
					myConf.toFlag = true
				}

			case Commit:

			case LoggStore:
				//msg := incm.(LoggStore)
				//storeDate(msg.Index, msg.Data)

			case StateStore:

			}
		}
	}
	fmt.Println("Bye", incm)
}

//Process to listen incomming packets from other Servers.
func processInbox(server cluster.Server) {
	for {
		env := <-server.Inbox()
		//fmt.Printf("[From: %d MsgId:%d] ", env.Pid, env.MsgId)
		//fmt.Println(env.Msg)
		//msg := env.Msg.(MyStructure)
		switch env.Msg.(type) {
		case VoteReq:
			netCh <- env.Msg
		case VoteResp:
			netCh <- env.Msg
		case AppEntrReq:
			netCh <- env.Msg
		case AppEntrResp:
			netCh <- env.Msg
		}
	}
}

//Process to send packets to other Servers.
func processOutbox(server cluster.Server, msg Send) {
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
			AppResp.Peer = int32(myId)
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), MsgId: 22, Msg: AppResp}
		case VoteResp:
			VotResp := msg.Event.(VoteResp)
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), MsgId: 2, Msg: VotResp}
		}
	}
}

//Trigger Timeouts for State.
func processTO(myConf *Config) {
	for {
		//Send timeout to State after dur milliseconds.
		if myConf.toFlag && myConf.DoTO.Before(time.Now()) {
			myConf.toFlag = false
			//fmt.Println("Timeout Called")
			timeoutCh <- nil
		}
	}
}

/*func storeDate(ind int, data Logg) {
	for i := ind; i < len(data); i++ {
		err = lg.Append(string(data[i]))
		assert(err == nil)
	}
}*/

//Random function to select random time for election timeout.
//Not covered in test cases as output will be non deTerministic.
func random() int {
	min := 1500
	max := 3000
	return min + rand.Intn(max-min)
}

//Configuration of Log and Node.
func (conf MyConfig) logConfig(myid int) Config {
	file, _ := os.Open("log_config.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&conf)
	if err != nil {
		fmt.Println("error:", err)
	}

	var myConf Config
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
		log.Fatalf("Expected this server's id (\"%d\") to be present in the configuration", myid)
	}
	return myConf
}

//Channel for passing timeout duration to the processTO background process.
var processTOCh = make(chan int)
var myId int

func main() {
	//Register a struct name by giving it a dummy object of that name.
	//gob.Register(MyStructure{})
	gob.Register(AppEntrReq{})
	gob.Register(AppEntrResp{})
	gob.Register(VoteReq{})
	gob.Register(VoteResp{})
	gob.Register(StateStore{})
	gob.Register(LoggStore{})

	var sm State_Machine
	var conf MyConfig
	flag.Parse()
	//Get Server id from command line.
	myId, _ = strconv.Atoi(flag.Args()[0])
	sm.id = int32(myId)
	rand.Seed(time.Now().UTC().UnixNano() * int64(myId))

	//Set up details about cluster nodes form json file.
	server, err := cluster.New(myId, "cluster_config.json")
	if err != nil {
		panic(err)
	}

	//Initialize the Log and Node configuration.
	myConf := conf.logConfig(myId)
	myConf.toFlag = false
	now := time.Now()
	myConf.DoTO = now.Add(time.Duration(10) * time.Minute)

	//Start backaground process to listen incomming packets from other servers.
	go processInbox(server)

	//Start background process to trigger timeout events.
	go processTO(&myConf)

	//Start StateMachine to process incomming packets in background.
	go sm.FollSys()

	//Raft node Processing.
	processEvents(server, &sm, &myConf)
}
