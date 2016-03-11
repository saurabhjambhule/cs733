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
)

type MyStructure struct {
	AppReq  AppEntrReq
	AppResp AppEntrResp
	VotReq  VoteReq
	VotResp VoteResp
}

type Config struct {
	Id               int    // this node's id. One of the cluster's entries should match.
	LogDir           string // Log file directory for this node
	ElectionTimeout  int
	HeartbeatTimeout int
}

type MyConfig struct {
	Details []Config
}

type incomming interface {
}

//Process incoming events from StateMachine.
func processEvents(server cluster.Server, sm *State_Machine, myConf *Config) {
	var incm incomming
	for {
		//fmt.Println("In eventProcess Node")
		select {
		case incm := <-actionCh:
			switch incm.(type) {
			case Send:
				msg := incm.(Send)
				fmt.Println("<-Recieved", msg, " Count - ", sm.VoteGrant[1], " Term -", sm.currTerm)
				processOutbox(server, msg)

			case Alarm:
				al := incm.(Alarm)
				toFlag = true
				if al.T == myConf.HeartbeatTimeout {
					toDur = myConf.HeartbeatTimeout * 100
				} else {
					toDur = myConf.ElectionTimeout * 100
				}

			case Commit:

			case LoggStore:

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
		fmt.Printf("[From: %d MsgId:%d]\n", env.Pid, env.MsgId)
		msg := env.Msg.(MyStructure)
		switch env.MsgId {
		case 1:
			netCh <- msg.VotReq
		case 2:
			//fmt.Println("IN<<<<<")
			netCh <- msg.VotResp
		case 11:
			netCh <- msg.AppReq
		case 22:
			netCh <- msg.AppResp
		}
	}
}

//Process to send packets to other Servers.
func processOutbox(server cluster.Server, msg Send) {
	fmt.Printf("******IN<<< %T\n", msg.Event)
	//fmt.Println("*IN*", msg.PeerId)
	var tmp MyStructure
	if msg.PeerId == 0 {
		switch msg.Event.(type) {
		case AppEntrReq:
			tmp.AppReq = msg.Event.(AppEntrReq)
			server.Outbox() <- &cluster.Envelope{Pid: cluster.BROADCAST, MsgId: 11, Msg: tmp}
		case VoteReq:
			tmp.VotReq = msg.Event.(VoteReq)
			server.Outbox() <- &cluster.Envelope{Pid: cluster.BROADCAST, MsgId: 1, Msg: tmp}
		}
	} else {
		switch msg.Event.(type) {
		case AppEntrReq:
			tmp.AppReq = msg.Event.(AppEntrReq)
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), MsgId: 11, Msg: tmp}
		case AppEntrResp:
			tmp.AppResp = msg.Event.(AppEntrResp)
			tmp.AppResp.Peer = int32(myId)
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), MsgId: 22, Msg: tmp}
		case VoteResp:
			tmp.VotResp = msg.Event.(VoteResp)
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), MsgId: 2, Msg: tmp}
		}
	}
}

//Trigger Timeouts for State.
func processTO() {
	for {
		//Send timeout to State after dur milliseconds.
		if toFlag == true {
			fmt.Println("Alarm Called", toDur)
			time.Sleep(time.Millisecond * time.Duration(toDur))
			fmt.Println("Timeout Called")
			timeoutCh <- nil
			toFlag = false
		}
	}
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
var toDur int
var toFlag bool
var myId int

func main() {
	//Register a struct name by giving it a dummy object of that name.
	gob.Register(MyStructure{})
	gob.Register(AppEntrReq{})
	gob.Register(AppEntrResp{})
	gob.Register(VoteReq{})
	gob.Register(VoteResp{})

	rand.Seed(time.Now().UTC().UnixNano())
	var sm State_Machine
	var conf MyConfig
	flag.Parse()
	//Get Server id from command line.
	myId, _ = strconv.Atoi(flag.Args()[0])
	sm.id = int32(myId)

	//Set up details about cluster nodes form json file.
	server, err := cluster.New(myId, "cluster_config.json")
	if err != nil {
		panic(err)
	}

	//Get Log configuration.
	myConf := conf.logConfig(myId)

	//Start backaground process to listen incomming packets from other servers.
	go processInbox(server)

	//Start background process to trigger timeout events.
	go processTO()

	//Start StateMachine to process incomming packets in background.
	go sm.FollSys()

	//Raft node Processing.
	processEvents(server, &sm, &myConf)
}
