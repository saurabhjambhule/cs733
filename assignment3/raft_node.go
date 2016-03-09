package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/cs733-iitb/cluster"
)

type incomming interface {
}

//Process incoming events from StateMachine.
func processEvents(server cluster.Server) {
	var incm incomming
	for {
		fmt.Println("In eventProcess Node")
		select {
		case incm := <-actionCh:
			switch incm.(type) {
			case Send:
				msg := incm.(Send)
				fmt.Println("<-Recieved", msg)
				processOutbox(server, msg)

			case Alarm:
				al := incm.(Alarm)
				toFlag = true
				fmt.Println("<-Recieved", al)
				toDur = al.T

			case Commit:

			}
		}
	}
	fmt.Println("Bye", incm)
}

//Process to listen incomming packets from other Servers.
func processInbox(server cluster.Server) {
	var incPackt1 VoteReq
	var incPackt2 VoteResp
	var incPackt11 AppEntrReq
	var incPackt22 AppEntrResp
	for {
		env := <-server.Inbox()
		fmt.Printf("[From: %d MsgId:%d]\n", env.Pid, env.MsgId)
		tmp := env.Msg.([]byte)
		switch env.MsgId {
		case 1:
			json.Unmarshal(tmp, &incPackt1)
			netCh <- incPackt1

		case 2:
			json.Unmarshal(tmp, &incPackt2)
			netCh <- incPackt2

		case 11:
			json.Unmarshal(tmp, &incPackt11)
			netCh <- incPackt11

		case 22:
			json.Unmarshal(tmp, &incPackt22)
			netCh <- incPackt22
		}
	}
}

//Process to send packets to other Servers.
func processOutbox(server cluster.Server, msg Send) {
	final, _ := json.Marshal(msg.Event)
	if msg.PeerId == 0 {
		switch msg.Event.(type) {
		case AppEntrReq:
			server.Outbox() <- &cluster.Envelope{Pid: cluster.BROADCAST, MsgId: 11, Msg: final}
		case VoteReq:
			server.Outbox() <- &cluster.Envelope{Pid: cluster.BROADCAST, MsgId: 1, Msg: final}
		}
	} else {
		switch msg.Event.(type) {
		case AppEntrReq:
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), MsgId: 11, Msg: final}
		case AppEntrResp:
			TmpResp := msg.Event.(AppEntrResp)
			TmpResp.Peer = int32(myId)
			finalT, _ := json.Marshal(TmpResp)
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), MsgId: 22, Msg: finalT}
		case VoteResp:
			server.Outbox() <- &cluster.Envelope{Pid: int(msg.PeerId), MsgId: 2, Msg: final}
		}
	}
}

//Trigger Timeouts for State.
func processTO() {
	for {
		//Send timeout to State after dur milliseconds.
		if toFlag == true {
			//dur := time.Duration(<-processTOCh)
			fmt.Println("Alarm Called", toDur)
			time.Sleep(time.Second * time.Duration(toDur))
			fmt.Println("Timeout Called")
			timeoutCh <- nil
			toFlag = false
		}
	}
}

//Channel for passing timeout duration to the processTO background process.
var processTOCh = make(chan int)
var toDur int
var toFlag bool
var myId int

func main() {
	var sm State_Machine
	flag.Parse()
	//Get Server id from command line.
	myId, _ = strconv.Atoi(flag.Args()[0])

	//Set up details about cluster nodes form json file.
	server, err := cluster.New(myId, "cluster_config.json")
	if err != nil {
		panic(err)
	}

	//Start backaground process to listen incomming packets from other servers.
	go processInbox(server)

	//Start background process to trigger timeout events.
	go processTO()

	//Start StateMachine to process incomming packets in background.
	go sm.FollSys()

	processEvents(server)
}
