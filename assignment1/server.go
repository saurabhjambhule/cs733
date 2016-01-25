package main

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"
	"strconv"
	"strings"
	"time"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

//Structre : DBData
//Contains metadata and contents of the file .
type DBData struct {
	Vers  int
	Cont  string
	Life  time.Time
	TFlag int
}

var mutex = &sync.Mutex{}

//Function: serverMain
//Creates socket, listens to requets and serve them.
func serverMain() {

	service := ":8080"
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	checkError(err)

	//Enable servers to listen for incomming connections.
	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)

	//Create database connection.
	fileDB, err := leveldb.OpenFile("./DB", nil)
	checkError(err)
	defer fileDB.Close()

	//Infinite loop, so serevr can run continuously till we terminate.
	for {

		//Accepts incomming client's connection. 
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		//Create new thread to SERVE client. 
		go handleServer(conn, fileDB)
	}
}

//Function: handleServer
//Serves clients various requets.
//Parameters:
//    conn - client connection
//    fileDB - database connection
func handleServer(conn net.Conn, fileDB *leveldb.DB) {

	var incMsg, outMsg string
	var last string = ""
	incMsgT := make([]byte, 1024)

	for {
		//Read client's send message
		in, err := bufio.NewReader(conn).Read(incMsgT)
		if err != nil {
			return
		}

		//Replacing \r\n with ## for splitting multiple commands.
		for i := 0; i < len(incMsgT); i++ {
			if (incMsgT[i] == uint8(13)) && (incMsgT[i+1] == uint8(10)) {
				incMsgT[i] = uint8('#')
				incMsgT[i+1] = uint8('#')
			}
		}

		//convert bytes into string and remove last \r\n
		incMsg = string(incMsgT)
		txt := incMsg[0 : in-2]

		//If the contents are in two different incomming message, then merge them.
		if len(last) != 0 {

			incMsg = last + txt
		} else {
			incMsg = txt
		}

		//Getting various commands from incomming messages.
		cmd := strings.Split(incMsg, "##")

		//Proccessing the requested command.
		for i := 0; i < len(cmd); {

			outMsg, i, last = cmdEval(cmd, i, conn, fileDB)

			//Write result to client. 
			conn.Write([]byte(outMsg + "\n"))

		}
	}

}

//Function: isExpireT
//Checkes whether the file is expired or not. If expired delete the file.
//Parameters:
//    fileName - file name
//    iterT - pointer database location
// Return:
//	  true - file expired
//	  false - file alive
func isExpireT(fileName string, iterT iterator.Iterator) bool {

	var expFlag int = 0
	var flag int = 0
	var f3 DBData

	key1 := iterT.Key()

	if string(key1) == fileName {
		flag = 1
	}

	if flag == 1 {
		tmp := iterT.Value()
		json.Unmarshal(tmp, &f3)
		var flagT1 int = f3.TFlag

		if flagT1 == 1 {
			timeT := f3.Life
			now := time.Now()
			tmp := timeT.Before(now)
			if tmp {
				expFlag = 1
			}
		} else {
			expFlag = 0
		}
	}

	if expFlag == 1 {
		return true
	} else {
		return false
	}
}


//Function: checkError
//Prints error
//Parameters:
//    err - error that encounterd
func checkError(err error) {

	if err != nil {
		return
	}
}

//Function: isCmd
//Validation of client's provided COMMAND
//Parameters:
//    cmdT - command
// Return:
//	  1 - command ok
//	  0 - command not ok
func isCmd(cmdT []string) int {

	cmdTyp := string(cmdT[0])
	cmdLen := len(cmdT)

	switch cmdTyp {

	case "write", "append":
		if !(cmdLen == 3 || cmdLen == 4) {
			return 0
		} else {
			if _, err := strconv.Atoi(string(cmdT[2])); err != nil {
				return 0
			} else if len([]byte(cmdT[1])) > 250 {
				return 0
			} else if cmdLen == 4 {
				if _, err := strconv.Atoi(string(cmdT[3])); err != nil {
					return 0
				}
			}
		}
		return 1

	case "read", "delete":
		if cmdLen > 2 || cmdLen < 2 {
			return 0
		} else if len([]byte(cmdT[1])) > 250 {
			return 0
		}
		return 1

	case "cas":
		if !(cmdLen == 4 || cmdLen == 5) {
			return 0
		} else {
			if _, err := strconv.Atoi(string(cmdT[3])); err != nil {
				return 0
			} else if len([]byte(cmdT[1])) > 250 {
				return 0
			} else if cmdLen == 4 {
				if _, err := strconv.Atoi(string(cmdT[2])); err != nil {
					return 0
				}
			}
		}
		return 1

	default:
		return 2
	}
	return 0
}


//Function: cmdEval
//Evalutes result of client's command
//Parameters:
//    cmdTmp - client's command
//    conn - client's connection
//	  fileDB - database connection
//Return:
//	  Result after successfull command execution, otherwise error msg
func cmdEval(cmdTmp []string, i int, conn net.Conn, fileDB *leveldb.DB) (string, int, string) {

	cmd := strings.Fields(cmdTmp[i])
	cmdTyp := string(cmd[0])
	cmdLen := len(cmd)

	switch cmdTyp {
	case "write":
		tt, ii, last := writeFile(cmd, cmdTmp, i, cmdLen, conn, fileDB)
		return tt, ii, last
	case "read":
		tt, ii := readFile(cmd, i, cmdLen, fileDB)
		return tt, ii, ""
	case "cas":
		tt, ii, last := casFile(cmd, cmdTmp, i, cmdLen, conn, fileDB)
		return tt, ii, last
	case "append":
		tt, ii, last := appendFile(cmd, cmdTmp, i, cmdLen, conn, fileDB)
		return tt, ii, last
	case "delete":
		tt, ii := deleteFile(cmd, i, cmdLen, fileDB)
		return tt, ii, ""
	default:
		i++
		return "", i, ""
	}
	i++
	return "ERR_CMD_ERR", i, ""
}


//Function: getContent
//Collect the content of the file.
//Return:
//	  flagCont - true, if all content of the file recieved
//	  fileCont - flagCont:true- content of the file, flagCont:false- partial content waits for remaining 
//    flagErr -  if file is oversize than stated size in command
func getContent(cmd []string, cmd1 []string, i int, sz int, cmdLen int)(bool, int, string, bool){


	var noByt int = 0
	var last string = ""
	var fileCont string = ""

	last = last + cmd1[i] + " "

	last = last + "##"

	if len(cmd1) == i+1 {
		i++
		return false, i, last, false
	}

	flagff := true

	for {

		i++

		last = last + cmd1[i] + "\n"

		cmddd := strings.Fields(cmd1[i])

		if isCmd(cmddd) != 2 {
			break
		}

		if len(cmd1[i]) == 0 {
			fileCont = fileCont + "\n"
			continue
		}

		if flagff {
			fileCont = cmd1[i]
			flagff = false
		} else {
			fileCont = fileCont + "\n" + cmd1[i]
		}


		noByt = noByt + len(cmd1[i])

		if noByt > sz {
			i++
			return false, i, "ERR_INTERNAL", true
		} else if noByt == sz {
			break
		}

		if len(cmd1) == i+1 {

			last = last + "##"
			i++
			return false, i, last, false
		}

	}

	return true, i, fileCont, false

}

//Function: getFile
//Checkes whether file exists or not
//Parameters:
//    fileName - file name
//    fileDB - database connection
//Return:
//	  true - file found
//	  false - file not found
//	  file itself
func getFile(fileName string, fileDB *leveldb.DB) (bool, []byte, iterator.Iterator) {

	var flag int = 0
	var val []byte

	iter := fileDB.NewIterator(nil, nil)

	//doOperation([]byte(fileNm), nil, "delete", fileDB, wo)

	for iter.Next() {
		key := iter.Key()
		if string(key) == fileName {
			val = iter.Value()
			flag = 1
			break
		}
	}

	if flag == 1 {
		return true, val, iter
	} else {
		return false, nil, iter
	}

}

func doOperation(fileNm []byte, finalByt []byte, cmdTyp string, fileDB *leveldb.DB, wo *opt.WriteOptions) {

	var err error

	mutex.Lock()
	switch cmdTyp {
		case "write":	err = fileDB.Put(fileNm, finalByt, nil)
		case "delete": 	err = fileDB.Delete([]byte(fileNm), wo)
	}
	mutex.Unlock()
	checkError(err)
}

//Function: writeFile
//Insert given data to database.
//Return:
//	  OK <version> - successfull
//	  ERR_CMD_ERR - invalid command
//	  ERR_INTERNAL - content exceeds given limit
func writeFile(cmd []string, cmd1 []string, i int, cmdLen int, conn net.Conn, fileDB *leveldb.DB) (string, int, string) {
	
	var fileCont string = ""
	var incMsg []byte
	var vStr string
	var f1, f2 DBData
	var flagCont, flagErr bool
	
	sz, _ := strconv.Atoi(cmd[2])
	if isCmd(cmd) == 0 {
		i++
		return "ERR_CMD_ERR", i, ""
	}

	fileNm := string(cmd[1])

	flagCont, i, fileCont, flagErr = getContent(cmd,cmd1, i, sz,cmdLen)
	if flagErr{
		return fileCont, i, ""
	}
	if !flagCont{
		return "", i, fileCont
	}

	flag, val, _ := getFile(fileNm, fileDB)

	if flag {

		json.Unmarshal(val, &f1)

		var versT1 int = f1.Vers
		versT1 = versT1 + 1
		vStr = strconv.Itoa(versT1)

		f2.Vers = versT1
		f2.Cont = fileCont
		f2.TFlag = 0

		timeT, _ := strconv.Atoi(string(cmd[3]))
		if cmdLen == 4 && timeT != 0 {

			f2.TFlag = 1
			now := time.Now()
			newT := now.Add(time.Duration(timeT) * time.Second)
			f2.Life = newT
		}

		final, _ := json.Marshal(f2)

		doOperation([]byte(fileNm), []byte(final), "write", fileDB, nil)

		i++

		return "OK " + vStr, i, ""

	} else {

		versT1 := 1001

		vStr = string(versT1)

		f2.Vers = versT1
		f2.Cont = fileCont
		f2.TFlag = 0

		if cmdLen == 4 {

			f2.TFlag = 1
			now := time.Now()
			timeT, _ := strconv.Atoi(string(cmd[3]))

			newT := now.Add(time.Duration(timeT) * time.Second)
			f2.Life = newT
		}

		final, _ := json.Marshal(f2)
		
		doOperation([]byte(fileNm), []byte(final), "write", fileDB, nil)
		i++

		return "OK " + "1001", i, ""
	}
	_ = incMsg
	return "OK", i, ""
}

//Function: readFile
//Read given file data from database.
//Return:
//	  CONTENTS <version> <numbytes> <exptime> \r\n <content bytes> - file contents on success
//	  ERR_CMD_ERR - invalid command
//	  ERR_FILE_NOT_FOUND - give file doesnt exist
func readFile(cmd []string, i int, cmdLen int, fileDB *leveldb.DB) (string, int) {

	var flagT bool = false
	var retStr string
	var cont string
	var timeT string
	var vStr string
	var f1 DBData

	if isCmd(cmd) == 0 {
		i++
		return "ERR_CMD_ERR", i
	}

	fileNm := string(cmd[1])

	flag, val, iter1 := getFile(fileNm, fileDB)

	if flag {
		if !isExpireT(fileNm, iter1) {
			flagT = true
		}else{
				var wo *opt.WriteOptions
				doOperation([]byte(fileNm), nil, "delete", fileDB, wo)
		}
	}

	if flagT {

		json.Unmarshal(val, &f1)

		var versT1 int = f1.Vers
		vStr = strconv.Itoa(versT1)

		cont = f1.Cont
		contT := []byte(cont)
		lenT := len(contT)
		lenT1 := strconv.Itoa(lenT)

		var flagT1 int = f1.TFlag

		if flagT1 == 1 {
			timeT = f1.Life.String()
			retStr = "CONTENTS " + vStr + " " + lenT1 + " " + timeT + "\n" + cont
			i++
			return retStr, i
		} else {
			retStr = "CONTENTS " + vStr + " " + lenT1 + "\n" + cont
			i++
			return retStr, i
		}
		iter1.Release()
	} else {
		i++
		return "ERR_FILE_NOT_FOUND", i
	}
	return "OK", i
}

//Function: writeFile
//Insert given data to database.
//Return:
//	  OK <version> - successfull
//	  ERR_CMD_ERR - invalid command
//	  ERR_INTERNAL - content exceeds given limit
func casFile(cmd []string, cmd1 []string, i int, cmdLen int, conn net.Conn, fileDB *leveldb.DB) (string, int, string) {

	var flagCont, flagErr bool
	var flag bool = false
	var fileCont string = ""
	var incMsg []byte
	var vStr string
	var vStrT string
	var versT1 int
	var f1, f2 DBData

	sz, _ := strconv.Atoi(cmd[3])
	if isCmd(cmd) == 0 {
		i++
		return "ERR_CMD_ERR", i, ""
	}

	fileNm := string(cmd[1])
	vStr = string(cmd[2])

	flagT, val, _ := getFile(fileNm, fileDB)

	if flagT {

		json.Unmarshal(val, &f1)

		versT1 = f1.Vers
		vStrT = strconv.Itoa(versT1)
		if vStr == vStrT {
			flag = true
		} else {
			i++
			return "ERR_VERSION " + vStrT, i, ""
		}
	}

	if flag {

		flagCont, i, fileCont, flagErr = getContent(cmd, cmd1, i, sz, cmdLen)
		if flagErr{
			return fileCont, i, ""
		}
		if !flagCont{
			return "", i, fileCont
		}

		versT1 = versT1 + 1
		vStrT = strconv.Itoa(versT1)

		f2.Cont = fileCont
		f2.TFlag = 0
		f2.Vers = versT1

		if cmdLen == 5 {

			timeT, _ := strconv.Atoi(string(cmd[4]))

			if timeT != 0 {

				f2.TFlag = 1
				now := time.Now()
				newT := now.Add(time.Duration(timeT) * time.Second)
				f2.Life = newT
			}
		}

		final, _ := json.Marshal(f2)

		doOperation([]byte(fileNm), []byte(final), "write", fileDB, nil)

		i++
		return "OK " + vStrT, i, ""

	} else {
		i++
		return "ERR_FILE_NOT_FOUND", i, ""
	}
	_ = incMsg
	return "OK", i, ""
}

func appendFile(cmd []string, cmd1 []string, i int, cmdLen int, conn net.Conn, fileDB *leveldb.DB) (string, int, string) {

	var flagCont, flagErr bool
	var fileCont string = ""
	var incMsg []byte
	var vStr string
	var f1, f2 DBData

	sz, _ := strconv.Atoi(cmd[2])
	if isCmd(cmd) == 0 {
		i++
		return "ERR_CMD_ERR", i, ""
	}

	fileNm := string(cmd[1])
	flagCont, i, fileCont, flagErr = getContent(cmd,cmd1, i, sz,cmdLen)
	if flagErr{
		return fileCont, i, ""
	}
	if !flagCont{
		return "", i, fileCont
	}

	flag, val, _ := getFile(fileNm, fileDB)

	if flag {

		json.Unmarshal(val, &f1)

		var versT1 int = f1.Vers
		versT1 = versT1 + 1
		vStr = strconv.Itoa(versT1)

		var contT1 string = f1.Cont

		f2.Vers = versT1
		f2.Cont = contT1 + fileCont
		f2.TFlag = 0

		if cmdLen == 4 {

			timeT, _ := strconv.Atoi(string(cmd[3]))

			if timeT != 0 {
				f2.TFlag = 1
				now := time.Now()
				newT := now.Add(time.Duration(timeT) * time.Second)
				f2.Life = newT

			}

		}

		final, _ := json.Marshal(f2)
		doOperation([]byte(fileNm), []byte(final), "write", fileDB, nil)
		i++

		return "OK " + vStr, i, ""

	} else {
		i++
		return "ERR_FILE_NOT_FOUND", i, ""
	}
	_ = incMsg
	return "OK", i, ""
}

func deleteFile(cmd []string, i int, cmdLen int, fileDB *leveldb.DB) (string, int) {

	var flagT bool = false

	if isCmd(cmd) == 0 {
		i++
		return "ERR_CMD_ERR", i
	}

	fileNm := string(cmd[1])

	flag, _, iter1 := getFile(fileNm, fileDB)

	if flag {
		if !isExpireT(fileNm, iter1) {
			flagT = true
		}
	}

	if flagT {

		var wo *opt.WriteOptions

		doOperation([]byte(fileNm), nil, "delete", fileDB, wo)
		i++
		return "OK", i
	} else {
		i++
		return "ERR_FILE_NOT_FOUND", i
	}
	return "OK", i
}

func main() {

	serverMain()
}
