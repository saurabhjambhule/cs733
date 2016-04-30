package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/saurabhjambhule/cs733/assignment4/fs"
	"github.com/saurabhjambhule/cs733/assignment4/raft"
	"github.com/saurabhjambhule/cs733/assignment4/raft/sm"
)

const (
	SIZE = 5
)

var mutex sync.Mutex

//Contains client handler config dada.
type Handler struct {
	Id        int    //this node's Id. One of the cluster's entries should match
	Address   string //address for the client handler
	ClientMap map[string]*net.TCPConn
	sync.RWMutex
}

//Contains client handler of all nodes in cluster.
type MyHandler struct {
	Servers []*Handler
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
	case 'R':
		resp = "ERR_REDIRECT " + string(msg.Contents)
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

func serve(clientId string, myNode *raft.RaftMachine, clHandl *Handler) {
	clHandl.RLock()
	conn := clHandl.ClientMap[clientId]
	clHandl.RUnlock()
	//fmt.Println("***", conn)
	reader := bufio.NewReader(conn)
	for {
		msg, msgerr, fatalerr := fs.GetMsg(reader)
		if fatalerr != nil || msgerr != nil {
			reply(conn, &fs.Msg{Kind: 'M'})
			conn.Close()
			clHandl.Lock()
			delete(clHandl.ClientMap, clientId)
			clHandl.Unlock()
			break
		}

		if msgerr != nil {
			if (!reply(conn, &fs.Msg{Kind: 'M'})) {
				conn.Close()
				clHandl.Lock()
				delete(clHandl.ClientMap, clientId)
				clHandl.Unlock()
				break
			}
		}
		//fmt.Println("---:", msg)

		if string(msg.Kind) != "r" {
			//fmt.Println(msg)
			msgByt, _ := json.Marshal(msg)
			raft.ClientAppend(myNode, clientId, msgByt)
			//fmt.Println("---->")
		} else {
			time.Sleep(1 * time.Second)
			response := fs.ProcessMsg(msg)
			if !reply(conn, response) {
				conn.Close()
				clHandl.Lock()
				delete(clHandl.ClientMap, clientId)
				clHandl.Unlock()
				break
			}
		}
	}
}

func responseReq(myNode *raft.RaftMachine, clHandl *Handler) {
	var msg *fs.Msg
	for {
		select {
		case commData := <-myNode.SM.CommMedium.CommitCh:
			cmData := (commData).(sm.Commit)
			//fmt.Println("@@@", clHandl.ClientMap)

			clHandl.RLock()
			conn := clHandl.ClientMap[cmData.Data.Id]
			clHandl.RUnlock()
			//fmt.Println("@@@", cmData.Data.Id)

			redir := strings.Fields(string(cmData.Err))

			if cmData.Err != nil {
				if redir[0] == "ERR_REDIRECT" {
					conn = clHandl.ClientMap[redir[1]]
					reply(conn, &fs.Msg{Kind: 'R', Contents: []byte(strconv.Itoa(int(cmData.Index)))})
					conn.Close()
					clHandl.Lock()
					delete(clHandl.ClientMap, cmData.Data.Id)
					clHandl.Unlock()
					break
				}
				//	fmt.Println("---:>")
				reply(conn, &fs.Msg{Kind: 'M'})
				conn.Close()
				clHandl.Lock()
				delete(clHandl.ClientMap, cmData.Data.Id)
				clHandl.Unlock()
				break
			}

			data := []byte(cmData.Data.Logg)
			//fmt.Println("<---", bytes.Compare([]byte(msgByt), data1))
			json.Unmarshal(data, &msg)
			//fmt.Println("---:::>", msg)
			//fmt.Println("<---", msg)
			response := fs.ProcessMsg(msg)
			//fmt.Println("<:::---", response)

			if !reply(conn, response) {
				conn.Close()
				clHandl.Lock()
				delete(clHandl.ClientMap, cmData.Data.Id)
				clHandl.Unlock()
				break
			}

		case commData := <-myNode.SM.CommMedium.CommitInfoCh:
			cmData := (commData).(sm.CommitInfo)
			data := []byte(cmData.Data.Logg)
			json.Unmarshal(data, &msg)
			response := fs.ProcessMsg(msg)
			_ = response
		}
	}
}

//Client handler cnfiguration.
func handlerConfig(myId int) *Handler {
	var handl MyHandler
	cliHandl := new(Handler)
	file, _ := os.Open("config/handler_config.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&handl)
	if err != nil {
		fmt.Println("--error:", err)
	}

	//initializing map for handler.
	//m.hm = make(map[string]string)
	cliHandl.ClientMap = make(map[string]*net.TCPConn)

	//initializing config structure from jason file.
	foundMyId := false
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

func serverMain(myId int) {
	cliHandl := handlerConfig(myId)
	node := raft.StartRaft(myId)
	tcpaddr, err := net.ResolveTCPAddr("tcp", cliHandl.Address)
	check(err)
	tcp_acceptor, err := net.ListenTCP("tcp", tcpaddr)
	check(err)
	for {
		tcp_conn, err := tcp_acceptor.AcceptTCP()
		check(err)
		clientId := fmt.Sprintf("%s", tcp_conn.RemoteAddr())
		cliHandl.Lock()
		cliHandl.ClientMap[clientId] = tcp_conn
		cliHandl.Unlock()
		fmt.Println("conn-", clientId, cliHandl.ClientMap[clientId])
		go serve(clientId, node, cliHandl)
		go responseReq(node, cliHandl)
	}
}

func serverMainTest(myId int) (*Handler, *raft.RaftMachine) {
	cliHandl := handlerConfig(myId)
	node := raft.StartRaft(myId)
	tcpaddr, err := net.ResolveTCPAddr("tcp", cliHandl.Address)
	check(err)
	tcp_acceptor, err := net.ListenTCP("tcp", tcpaddr)
	check(err)
	go func(node *raft.RaftMachine, cliHandl *Handler) {
		for {
			tcp_conn, err := tcp_acceptor.AcceptTCP()
			check(err)
			clientId := fmt.Sprintf("%s", tcp_conn.RemoteAddr())
			cliHandl.Lock()
			cliHandl.ClientMap[clientId] = tcp_conn
			cliHandl.Unlock()
			//fmt.Println("conn-", clientId, cliHandl.ClientMap[clientId])
			go serve(clientId, node, cliHandl)
			go responseReq(node, cliHandl)
		}
	}(node, cliHandl)
	return cliHandl, node
}

func main() {
	flag.Parse()
	myid, _ := strconv.Atoi(flag.Args()[0])
	serverMain(myid)
}
