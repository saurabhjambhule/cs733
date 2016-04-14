package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/saurabhjambhule/cs733/assignment4/fs"
	"github.com/saurabhjambhule/cs733/assignment4/raft"
	"github.com/saurabhjambhule/cs733/assignment4/raft/sm"
)

const (
	SIZE = 5
)

//Contains client handler config dada.
type Handler struct {
	Id      int    //this node's Id. One of the cluster's entries should match
	Address string //address for the client handler
}

//Contains client handler of all nodes in cluster.
type MyHandler struct {
	Servers []Handler
}

var crlf = []byte{'\r', '\n'}

func check(obj interface{}) {
	if obj != nil {
		fmt.Println(obj)
		os.Exit(1)
	}
}

func reply(conn *net.TCPConn, msg *fs.Msg) bool {
	var err error
	write := func(data []byte) {
		if err != nil {
			return
		}
		_, err = conn.Write(data)
	}
	var resp string
	switch msg.Kind {
	case 'C': // read response
		resp = fmt.Sprintf("CONTENTS %d %d %d", msg.Version, msg.Numbytes, msg.Exptime)
	case 'O':
		resp = "OK "
		if msg.Version > 0 {
			resp += strconv.Itoa(msg.Version)
		}
	case 'F':
		resp = "ERR_FILE_NOT_FOUND"
	case 'V':
		resp = "ERR_VERSION " + strconv.Itoa(msg.Version)
	case 'M':
		resp = "ERR_CMD_ERR"
	case 'I':
		resp = "ERR_INTERNAL"
	default:
		fmt.Printf("Unknown response kind '%c'", msg.Kind)
		return false
	}
	resp += "\r\n"
	write([]byte(resp))
	if msg.Kind == 'C' {
		write(msg.Contents)
		write(crlf)
	}
	return err == nil
}

func serve(conn *net.TCPConn, SM *sm.State_Machine) {
	reader := bufio.NewReader(conn)
	for {
		msg, msgerr, fatalerr := fs.GetMsg(reader)
		if fatalerr != nil || msgerr != nil {
			reply(conn, &fs.Msg{Kind: 'M'})
			conn.Close()
			break
		}

		if msgerr != nil {
			if (!reply(conn, &fs.Msg{Kind: 'M'})) {
				conn.Close()
				break
			}
		}
		//fmt.Println("***", string(msg.Kind))
		if string(msg.Kind) != "r" {
			//fmt.Println(msg)
			msgByt, _ := json.Marshal(msg)
			msgToLog := sm.Append{Data: []byte(msgByt)}
			//fmt.Println("???>>", msgToLog)

			SM.CommMedium.ClientCh <- msgToLog
			commData := <-SM.CommMedium.CommitCh
			data := (commData).(sm.Commit)
			//data := (data.Data).(sm.MyLogg)
			//fmt.Println("???", data)
			_ = data
		}

		response := fs.ProcessMsg(msg)
		if !reply(conn, response) {
			conn.Close()
			break
		}
	}
}

//Client handler cnfiguration.
func handlerConfig(myId int) Handler {
	var handl MyHandler
	var cliHandl Handler
	file, _ := os.Open("config/handler_config.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&handl)
	if err != nil {
		fmt.Println("--error:", err)
	}
	foundMyId := false
	//initializing config structure from jason file.
	for _, srv := range handl.Servers {
		if srv.Id == myId {
			foundMyId = true
			cliHandl.Id = myId
			cliHandl.Address = srv.Address
		}
	}
	if !foundMyId {
		fmt.Println("--Expected this server's Id (\"%d\") to be present in the configuration", myId)
	}
	return cliHandl
}

func serverMain(myId int) (Handler, *raft.RaftMachine) {
	cliHandl := handlerConfig(myId)
	node := raft.StartRaft(myId)
	tcpaddr, err := net.ResolveTCPAddr("tcp", cliHandl.Address)
	check(err)
	tcp_acceptor, err := net.ListenTCP("tcp", tcpaddr)
	check(err)

	go func(SM *sm.State_Machine) {
		for {
			tcp_conn, err := tcp_acceptor.AcceptTCP()
			check(err)
			go serve(tcp_conn, SM)
		}
	}(node.SM)

	return cliHandl, node
}

func main() {
	flag.Parse()
	myid, _ := strconv.Atoi(flag.Args()[0])
	serverMain(myid)
}
