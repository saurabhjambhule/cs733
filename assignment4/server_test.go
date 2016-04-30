package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/saurabhjambhule/cs733/assignment4/raft"
)

const (
	PEER = 5
)

type Msg struct {
	// Kind = the first character of the command. For errors, it
	// is the first letter after "ERR_", ('V' for ERR_VERSION, for
	// example), except for "ERR_CMD_ERR", for which the kind is 'M'
	Kind     byte
	Filename string
	Contents []byte
	Numbytes int
	Exptime  int // expiry time in seconds
	Version  int
}

type Client struct {
	conn   *net.TCPConn
	reader *bufio.Reader // a bufio Reader wrapper over conn
}

var errNoConn = errors.New("Connection is closed")

var handlr MyHandler
var node raft.Raft
var leaderId int
var totEntr int
var cmds []*exec.Cmd

//var mutex = &sync.Mutex{}
//var wg sync.WaitGroup

func Test_StartCluster(t *testing.T) {
	cleanDB()
	/*
		for i := 1; i <= PEER; i++ {
			tempH, tempR := serverMainTest(i)
			handlr.Servers = append(handlr.Servers, tempH)
			node.Cluster = append(node.Cluster, tempR)
			//fmt.Println(handlr.Servers[i-1].ClientMap)
		}
		time.Sleep(2 * time.Second)
	*/

	cmds = make([]*exec.Cmd, 6)
	//cmds[0] = exec.Command("go build *.go")
	for i := 1; i <= 5; i++ {
		cmds[i] = exec.Command("./server", strconv.Itoa(i))
		err := cmds[i].Start()
		cmds[i].Stdout = os.Stdout
		cmds[i].Stderr = os.Stderr
		if err != nil {
			panic(err)
		}
	}

	//fmt.Println(cmds)
}

func TestLeaderElection(t *testing.T) {
	leaderId = -1
	time.Sleep(2 * time.Second)

	for i := 0; i < 5; i++ {
		cl := mkClient(t, i)
		defer cl.close()
		data := "Get Leader"
		m, err := cl.write("leader", data, 0)
		if err == nil && m.Kind == 'R' {
			leaderId = m.Numbytes
			break
		}
		if err == nil && m.Kind == 'O' {
			leaderId = i
			break
		}
	}
}

func TestRPC_BasicSequential(t *testing.T) {

	//leaderId = 0
	cl := mkClient(t, leaderId)
	defer cl.close()
	//fmt.Println("...")
	// Read non-existent file cs733net
	m, err := cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)
	// Read non-existent file cs733net
	m, err = cl.delete("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	// Write file cs733net
	data := "Cloud fun"
	m, err = cl.write("cs733net", data, 0)
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	// Expect to read it back
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data)}, "read my write", err)

	// CAS in new value
	version1 := m.Version
	data2 := "Cloud fun 2"
	// Cas new value
	m, err = cl.cas("cs733net", version1, data2, 0)
	expect(t, m, &Msg{Kind: 'O'}, "cas success", err)

	// Expect to read it back
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data2)}, "read my cas", err)

	// Expect Cas to fail with old version
	m, err = cl.cas("cs733net", version1, data, 0)
	expect(t, m, &Msg{Kind: 'V'}, "cas version mismatch", err)

	// Expect a failed cas to not have succeeded. Read should return data2.
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data2)}, "failed cas to not have succeeded", err)

	// delete
	m, err = cl.delete("cs733net")
	expect(t, m, &Msg{Kind: 'O'}, "delete success", err)

	// Expect to not find the file
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	//printDB(node, 5)
}

func TestRPC_Binary(t *testing.T) {
	//leaderId = CurrLeader(node.Cluster)

	cl := mkClient(t, leaderId)
	defer cl.close()

	// Write binary contents
	data := "\x00\x01\r\n\x03" // some non-ascii, some crlf chars
	m, err := cl.write("binfile", data, 0)
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	// Expect to read it back
	m, err = cl.read("binfile")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data)}, "read my write", err)

}

func TestRPC_Chunks(t *testing.T) {
	//leaderId = CurrLeader(node.Cluster)

	// Should be able to accept a few bytes at a time
	cl := mkClient(t, leaderId)
	defer cl.close()
	var err error
	snd := func(chunk string) {
		if err == nil {
			err = cl.send(chunk)
		}
	}

	// Send the command "write teststream 10\r\nabcdefghij\r\n" in multiple chunks
	// Nagle's aLgorithm is disabled on a write, so the server should get these in separate TCP packets.
	snd("wr")
	time.Sleep(10 * time.Millisecond)
	snd("ite test")
	time.Sleep(10 * time.Millisecond)
	snd("stream 1")
	time.Sleep(10 * time.Millisecond)
	snd("0\r\nabcdefghij\r")
	time.Sleep(10 * time.Millisecond)
	snd("\n")
	var m *Msg
	m, err = cl.rcv()
	expect(t, m, &Msg{Kind: 'O'}, "writing in chunks should work", err)
}

func TestRPC_Batch(t *testing.T) {
	//fmt.Println("--------")

	// Send multiple commands in one batch, expect multiple responses
	cl := mkClient(t, leaderId)
	defer cl.close()
	cmds := "write batch1 3\r\nabc\r\n" +
		"write batch2 4\r\ndefg\r\n"
	//+ "read batch1\r\n"

	cl.send(cmds)
	m, err := cl.rcv()

	expect(t, m, &Msg{Kind: 'O'}, "write batch1 success", err)
	m, err = cl.rcv()

	expect(t, m, &Msg{Kind: 'O'}, "write batch2 success", err)
	//m, err = cl.rcv()
	//expect(t, m, &Msg{Kind: 'C', Contents: []byte("abc")}, "read batch1", err)
}

func TestRPC_BasicTimer(t *testing.T) {
	//leaderId = CurrLeader(node.Cluster)

	cl := mkClient(t, leaderId)
	defer cl.close()

	// Write file cs733, with expiry time of 2 seconds
	str := "Cloud fun"
	m, err := cl.write("cs733", str, 2)
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	// Expect to read it back immediately.
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(str)}, "read my cas", err)

	time.Sleep(3 * time.Second)

	// Expect to not find the file after expiry
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	// Recreate the file with expiry time of 1 second
	m, err = cl.write("cs733", str, 1)
	expect(t, m, &Msg{Kind: 'O'}, "file recreated", err)

	// Overwrite the file with expiry time of 4. This should be the new time.
	m, err = cl.write("cs733", str, 3)
	expect(t, m, &Msg{Kind: 'O'}, "file overwriten with exptime=4", err)

	// The last expiry time was 3 seconds. We should expect the file to still be around 2 seconds later
	time.Sleep(2 * time.Second)

	// Expect the file to not have expired.
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(str)}, "file to not expire until 4 sec", err)

	time.Sleep(3 * time.Second)
	// 5 seconds since the last write. Expect the file to have expired
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'F'}, "file not found after 4 sec", err)

	// Create the file with an expiry time of 1 sec. We're going to delete it
	// then immediately create it. The new file better not get deleted.
	m, err = cl.write("cs733", str, 1)
	expect(t, m, &Msg{Kind: 'O'}, "file created for delete", err)

	m, err = cl.delete("cs733")
	expect(t, m, &Msg{Kind: 'O'}, "deleted ok", err)

	m, err = cl.write("cs733", str, 0) // No expiry
	expect(t, m, &Msg{Kind: 'O'}, "file recreated", err)

	time.Sleep(1100 * time.Millisecond) // A little more than 1 sec
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C'}, "file should not be deleted", err)

}

// nclients write to the same file. At the end the file should be
// any one clients' last write
func TestRPC_ConcurrentWrites(t *testing.T) {
	//leaderId = CurrLeader(node.Cluster)

	nclients := 50
	niters := 10
	clients := make([]*Client, nclients)
	for i := 0; i < nclients; i++ {
		cl := mkClient(t, leaderId)
		if cl == nil {
			t.Fatalf("Unable to create client #%d", i)
		}
		defer cl.close()
		clients[i] = cl
	}

	errCh := make(chan error, nclients)
	var sem sync.WaitGroup // Used as a semaphore to coordinate goroutines to begin concurrently
	sem.Add(1)
	ch := make(chan *Msg, nclients*niters) // channel for all replies
	for i := 0; i < nclients; i++ {
		go func(i int, cl *Client) {
			sem.Wait()
			for j := 0; j < niters; j++ {
				str := fmt.Sprintf("cl %d %d", i, j)
				m, err := cl.write("concWrite", str, 0)
				if err != nil {
					errCh <- err
					break
				} else {
					ch <- m
				}
			}
		}(i, clients[i])
	}
	time.Sleep(100 * time.Millisecond) // give goroutines a chance
	sem.Done()                         // Go!

	// There should be no errors
	for i := 0; i < nclients*niters; i++ {
		select {
		case m := <-ch:
			if m.Kind != 'O' {
				t.Fatalf("Concurrent write failed with kind=%c", m.Kind)
			}
		case err := <-errCh:
			t.Fatal(err)
		}
	}
	time.Sleep(3000 * time.Millisecond)
	m, _ := clients[0].read("concWrite")
	//fmt.Println(string(m.Kind), "-", string(m.Contents))
	// Ensure the contents are of the form "cl <i> 9"
	// The last write of any client ends with " 9"
	totEntr += niters * nclients
	str := " " + strconv.Itoa(niters-1)
	if !(m.Kind == 'C' && strings.HasSuffix(string(m.Contents), str)) {
		t.Fatalf("Expected to be able to read after 1000 writes. Got msg = %v", m)
	}
}

// nclients cas to the same file. At the end the file should be any one clients' last write.
// The only difference between this test and the ConcurrentWrite test above is that each
// client loops around until each CAS succeeds. The number of concurrent clients has been
// reduced to keep the testing time within limits.
func TestRPC_ConcurrentCas(t *testing.T) {

	nclients := 5
	niters := 5

	clients := make([]*Client, nclients)
	for i := 0; i < nclients; i++ {
		cl := mkClient(t, leaderId)
		if cl == nil {
			t.Fatalf("Unable to create client #%d", i)
		}
		defer cl.close()
		clients[i] = cl
	}

	var sem sync.WaitGroup // Used as a semaphore to coordinate goroutines to *begin* concurrently
	sem.Add(1)

	m, _ := clients[0].write("concCas", "first", 0)
	ver := m.Version
	if m.Kind != 'O' || ver == 0 {
		t.Fatalf("Expected write to succeed and return version")
	}

	var wg sync.WaitGroup
	wg.Add(nclients)

	errorCh := make(chan error, nclients)

	for i := 0; i < nclients; i++ {
		go func(i int, ver int, cl *Client) {
			sem.Wait()
			defer wg.Done()
			for j := 0; j < niters; j++ {
				//fmt.Println(i, ":", j)
				str := fmt.Sprintf("cl %d %d", i, j)
				for {
					m, err := cl.cas("concCas", ver, str, 0)
					if err != nil {
						errorCh <- err
						return
					} else if m.Kind == 'O' {
						break
					} else if m.Kind != 'V' {
						errorCh <- errors.New(fmt.Sprintf("Expected 'V' msg, got %c", m.Kind))
						return
					}
					ver = m.Version // retry with latest version
				}
			}
		}(i, ver, clients[i])
	}

	time.Sleep(1000 * time.Millisecond) // give goroutines a chance
	sem.Done()                          // Start goroutines
	wg.Wait()                           // Wait for them to finish

	select {
	case e := <-errorCh:
		t.Fatalf("Error received while doing cas: %v", e)
	default: // no errors
	}

	totEntr += niters * nclients
	m, _ = clients[0].read("concCas")
	str := " " + strconv.Itoa(niters-1)
	if !(m.Kind == 'C' && strings.HasSuffix(string(m.Contents), str)) {
		t.Fatalf("Expected to be able to read after 1000 writes. Got msg.Kind = %d, msg.Contents=%s", m.Kind, m.Contents)
	}
}

func Test_Shutdown(t *testing.T) {
	cleanDB()
	//Appending some entires.
	cl := mkClient(t, leaderId)
	defer cl.close()
	nclients := 1
	niters := 10
	errCh := make(chan error, nclients)
	ch := make(chan *Msg, nclients*niters) // channel for all replies
	for i := 0; i < 1; i++ {
		for j := 0; j < niters; j++ {
			str := fmt.Sprintf("cl %d %d", i, j)
			m, err := cl.write("concWrite", str, 0)
			if err != nil {
				errCh <- err
				break
			} else {
				ch <- m
			}
		}
	}
	time.Sleep(100 * time.Millisecond) // give goroutines a chance

	// There should be no errors
	for i := 0; i < nclients*niters; i++ {
		select {
		case m := <-ch:
			if m.Kind != 'O' {
				t.Fatalf("Concurrent write failed with kind=%c", m.Kind)
			}
		case err := <-errCh:
			t.Fatal(err)
		}
	}
	time.Sleep(3000 * time.Millisecond)
	m, _ := cl.read("concWrite")
	totEntr += niters * nclients
	str := " " + strconv.Itoa(niters-1)
	if !(m.Kind == 'C' && strings.HasSuffix(string(m.Contents), str)) {
		t.Fatalf("Expected to be able to read after 1000 writes. Got msg = %v", m)
	}

	cmds[leaderId+1].Process.Kill()
	time.Sleep(10 * time.Second)

	data := "chk alive"
	m, err := cl.write("alive", data, 0)
	if err == nil && m.Kind == 'R' {
		fmt.Println("...")
	}
	if err == nil && m.Kind == 'O' {
		t.Fatalf("Shudown fail")
	}
}

func Test_Restore(t *testing.T) {
	cmds[leaderId+1] = exec.Command("./server", strconv.Itoa(leaderId+1))
	err := cmds[leaderId+1].Start()
	cmds[leaderId+1].Stdout = os.Stdout
	cmds[leaderId+1].Stderr = os.Stderr
	if err != nil {
		t.Fatalf("Failed to restart server")
	}
	time.Sleep(10 * time.Second)
}

/*--------------------------------|Utility functions|--------------------------------------*/

func (cl *Client) read(filename string) (*Msg, error) {
	cmd := "read " + filename + "\r\n"
	return cl.sendRcv(cmd)
}

func (cl *Client) write(filename string, contents string, exptime int) (*Msg, error) {
	var cmd string
	if exptime == 0 {
		cmd = fmt.Sprintf("write %s %d\r\n", filename, len(contents))
	} else {
		cmd = fmt.Sprintf("write %s %d %d\r\n", filename, len(contents), exptime)
	}
	cmd += contents + "\r\n"
	return cl.sendRcv(cmd)
}

func (cl *Client) cas(filename string, version int, contents string, exptime int) (*Msg, error) {
	var cmd string
	if exptime == 0 {
		cmd = fmt.Sprintf("cas %s %d %d\r\n", filename, version, len(contents))
	} else {
		cmd = fmt.Sprintf("cas %s %d %d %d\r\n", filename, version, len(contents), exptime)
	}
	cmd += contents + "\r\n"
	return cl.sendRcv(cmd)
}

func (cl *Client) delete(filename string) (*Msg, error) {
	cmd := "delete " + filename + "\r\n"
	return cl.sendRcv(cmd)
}

func (cl *Client) send(str string) error {
	if cl.conn == nil {
		return errNoConn
	}
	_, err := cl.conn.Write([]byte(str))
	if err != nil {
		err = fmt.Errorf("Write error in SendRaw: %v", err)
		cl.conn.Close()
		cl.conn = nil
	}
	return err
}

func (cl *Client) sendRcv(str string) (msg *Msg, err error) {
	if cl.conn == nil {
		return nil, errNoConn
	}
	err = cl.send(str)
	if err == nil {
		msg, err = cl.rcv()
	}
	return msg, err
}

func (cl *Client) close() {
	if cl != nil && cl.conn != nil {
		cl.conn.Close()
		cl.conn = nil
	}
}

func (cl *Client) rcv() (msg *Msg, err error) {
	// we will assume no errors in server side formatting
	line, err := cl.reader.ReadString('\n')
	if err == nil {
		msg, err = parseFirst(line)
		if err != nil {
			return nil, err
		}
		if msg.Kind == 'C' {
			contents := make([]byte, msg.Numbytes)
			var c byte
			for i := 0; i < msg.Numbytes; i++ {
				if c, err = cl.reader.ReadByte(); err != nil {
					break
				}
				contents[i] = c
			}
			if err == nil {
				msg.Contents = contents
				cl.reader.ReadByte() // \r
				cl.reader.ReadByte() // \n
			}
		}
	}
	if err != nil {
		cl.close()
	}
	return msg, err
}

func parseFirst(line string) (msg *Msg, err error) {
	fields := strings.Fields(line)
	msg = &Msg{}

	// Utility function fieldNum to int
	toInt := func(fieldNum int) int {
		var i int
		if err == nil {
			if fieldNum >= len(fields) {
				err = errors.New(fmt.Sprintf("Not enough fields. Expected field #%d in %s\n", fieldNum, line))
				return 0
			}
			i, err = strconv.Atoi(fields[fieldNum])
		}
		return i
	}

	if len(fields) == 0 {
		return nil, errors.New("Empty line. The previous command is likely at fault")
	}
	switch fields[0] {
	case "OK": // OK [version]
		msg.Kind = 'O'
		if len(fields) > 1 {
			msg.Version = toInt(1)
		}
	case "CONTENTS": // CONTENTS <version> <numbytes> <exptime> \r\n
		msg.Kind = 'C'
		msg.Version = toInt(1)
		msg.Numbytes = toInt(2)
		msg.Exptime = toInt(3)
	case "ERR_VERSION":
		msg.Kind = 'V'
		msg.Version = toInt(1)
	case "ERR_FILE_NOT_FOUND":
		msg.Kind = 'F'
	case "ERR_CMD_ERR":
		msg.Kind = 'M'
	case "ERR_INTERNAL":
		msg.Kind = 'I'
	case "ERR_REDIRECT":
		msg.Kind = 'R'
		id, _ := strconv.Atoi(fields[0])
		msg.Numbytes = id
	default:
		err = errors.New("Unknown response " + fields[0])
	}
	if err != nil {
		return nil, err
	} else {
		return msg, nil
	}
}

func CurrLeader(Cluster []*raft.RaftMachine) int {
	for {
		for i := 0; i < PEER; i++ {
			if Cluster[i].SM.Status == "leader" {
				return i
			}
		}
	}
}

func printDB(myRaft raft.Raft, len int64) {
	for j := int64(0); j < len; j++ {
		res, err1 := myRaft.Cluster[0].Conf.Lg.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = myRaft.Cluster[1].Conf.Lg.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = myRaft.Cluster[2].Conf.Lg.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = myRaft.Cluster[3].Conf.Lg.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}

		res, err1 = myRaft.Cluster[4].Conf.Lg.Get(j)
		if err1 != nil {
			fmt.Print("\t")
		} else {
			fmt.Print(res)
			fmt.Print("\t")
		}
		fmt.Println("")
	}
}

func cleanDB() {
	os.RemoveAll("./log")
	os.RemoveAll("./state")

}

func getAddress(myId int) string {
	var handl MyHandler
	var port string
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
			port = srv.Address
		}
	}
	if !foundMyId {
		fmt.Println("--Expected this server's Id (\"%d\") to be present in the configuration", myId)
	}
	return port
}

/*
func mkClient(t *testing.T, id int) *Client {
	var client *Client
	raddr, err := net.ResolveTCPAddr("tcp", handlr.Servers[id].Address)
	if err == nil {
		conn, err := net.DialTCP("tcp", nil, raddr)
		if err == nil {
			client = &Client{conn: conn, reader: bufio.NewReader(conn)}
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	return client
}
*/

func mkClient(t *testing.T, id int) *Client {
	var client *Client
	//fmt.Println(getAddress(id + 1))
	raddr, err := net.ResolveTCPAddr("tcp", getAddress(id+1))
	if err == nil {
		conn, err := net.DialTCP("tcp", nil, raddr)
		//fmt.Println(err)
		if err == nil {
			client = &Client{conn: conn, reader: bufio.NewReader(conn)}
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func expect(t *testing.T, response *Msg, expected *Msg, errstr string, err error) {
	//fmt.Println("$$$$$$$$$$")
	if err != nil {
		t.Fatal("Unexpected error: " + err.Error())
	}
	ok := true
	if response.Kind != expected.Kind {
		ok = false
		errstr += fmt.Sprintf(" Got kind='%c', expected '%c'", response.Kind, expected.Kind)
	}
	if expected.Version > 0 && expected.Version != response.Version {
		ok = false
		errstr += " Version mismatch"
	}
	if response.Kind == 'C' {
		if expected.Contents != nil &&
			bytes.Compare(response.Contents, expected.Contents) != 0 {
			ok = false
		}
	}
	if !ok {
		t.Fatal("Expected " + errstr)
	}
}