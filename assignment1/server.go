package main

import (
    "fmt"
    "net"
    "os"
    "bufio"
    "strings"
    "strconv"
    "github.com/syndtr/goleveldb/leveldb"
)

func singleServer() {

    service := ":8080"
    tcpAddr, err := net.ResolveTCPAddr("tcp", service)
    checkError(err)

    listener, err := net.ListenTCP("tcp", tcpAddr)
    checkError(err)

    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        singleClient(conn)
        conn.Close() 
    }
}


func singleClient(conn net.Conn) {

    for{
        incMsg, err := bufio.NewReader(conn).ReadString('\n')
        if err != nil {
            return
        }
        fmt.Print("Received:", string(incMsg))
        outMsg := cmdEval(incMsg, conn)
        //outMsg := strings.ToUpper(incMsg)
        conn.Write([]byte(outMsg + "\n"))
        fmt.Print("Reply:", string(outMsg))
    }
}


func multiServer() {

    service := ":8080"
    tcpAddr, err := net.ResolveTCPAddr("tcp", service)
    checkError(err)

    listener, err := net.ListenTCP("tcp", tcpAddr)
    checkError(err)

    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        // run as a goroutine
        go multiClient(conn)
    }
}


func multiClient(conn net.Conn) {

    // close connection on exit
    defer conn.Close()

     for{
        incMsg, err := bufio.NewReader(conn).ReadString('\n')
        if err != nil {
            return
        }
        fmt.Print("Received:", string(incMsg))
        outMsg := cmdEval(incMsg, conn)
        //outMsg := strings.ToUpper(incMsg)
        conn.Write([]byte(outMsg + "\n"))
        fmt.Print("Reply:", string(outMsg))
    }
}


func checkError(err error) {

    if err != nil {
        fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
        os.Exit(1)
    }
}


func cmdEval(cmdTmp string, conn net.Conn) (output string) {

    cmd := strings.Fields(cmdTmp)
    cmdTyp := cmd[0]
    cmdLen := len(cmd)
    
    ch1 := "write"
    ch2 := "read"
    ch3 := "cas"
    ch4 := "append"
    ch5 := "rename"
    ch6 := "delete"

    switch cmdTyp{
        case ch1:   return writeFile(cmd, cmdLen, conn)
        case ch2:   return readFile(cmd, cmdLen)
        case ch3:   return casFile(cmd, cmdLen, conn)
        case ch4:   return appendFile(cmd, cmdLen, conn)
        case ch5:   return renameFile(cmd, cmdLen)
        case ch6:   return deleteFile(cmd, cmdLen)
        default: fmt.Println("Invalid Choice!")
    }
    return cmdTyp
}

func writeFile(cmd []string, cmdLen int, conn net.Conn) (resp string){
    
    var flag int = 0
    var flagT int = 0
    //var vStr string

    if !(cmdLen == 3 || cmdLen == 4){
        return "ERR_CMD_ERR"
    }

    fileNm := string(cmd[1])

    conn.Write([]byte("\n"))
    incMsg, err := bufio.NewReader(conn).ReadString('\n')
    if err != nil {
        return
    }
    fmt.Print("Received:", string(incMsg))


    fileDB, err := leveldb.OpenFile("./fileDB", nil)
    versDB, err := leveldb.OpenFile("./versDB", nil)
    timeDB, err := leveldb.OpenFile("./timeDB", nil)
    if err != nil {
        fmt.Println("DB Error!")
    }
    defer fileDB.Close()
    defer versDB.Close()
    defer timeDB.Close()

    iterTmp := fileDB.NewIterator(nil, nil)
    //fmt.Println("DB")
    for iterTmp.Next() {   
        keyT := string(iterTmp.Key())
        if keyT == "tmp"{
            flag = 1
            break
        }
    }
    if flagT == 0{
        err = fileDB.Put([]byte("tmp"), []byte("tmp"), nil)
        err = versDB.Put([]byte("tmp"), []byte("tmp"), nil)
        err = timeDB.Put([]byte("tmp"), []byte("tmp"), nil)
    }

    //sz, _ := strconv.Atoi(cmd[2])
    //fmt.Println(sz)

    iter := versDB.NewIterator(nil, nil)
    //fmt.Println("DB")
    for iter.Next() {
            
        key := string(iter.Key())
        fmt.Println(key)
        if key == fileNm{
            flag = 1
            break
        }
    }

    if flag == 1{
        val := string(iter.Value())
        vers, _ := strconv.Atoi(val)
        vers = vers + 1
        vStr := strconv.Itoa(vers)
        err = versDB.Put([]byte(cmd[1]), []byte(vStr), nil)
        fmt.Println(vStr)
        return "OK "+cmd[1]+vStr
        
    }else{
        //fmt.Println("DB")
        err = fileDB.Put([]byte(cmd[1]), []byte(incMsg), nil)
        err = versDB.Put([]byte(cmd[1]), []byte("1001"), nil)
        if cmdLen == 4 {
            err = timeDB.Put([]byte(cmd[1]), []byte(cmd[3]), nil)
        }
        return "OK "+cmd[1]+"1001"
    }
    return "OK"
}


func readFile(cmd []string, cmdLen int) (resp string){
    
    var flag int = 0
    var retStr string
    var fStr string
    var vStr string
    var tStr string

    fileNm := string(cmd[1])

    if cmdLen > 2 || cmdLen < 2{
        return "ERR_CMD_ERR"
    }

    fileDB, err := leveldb.OpenFile("./fileDB", nil)
    versDB, err := leveldb.OpenFile("./versDB", nil)
    timeDB, err := leveldb.OpenFile("./timeDB", nil)
    if err != nil {
        fmt.Println("DB Error!")
    }
    defer fileDB.Close()
    defer versDB.Close()
    defer timeDB.Close()

    iter := fileDB.NewIterator(nil, nil)
    for iter.Next() {
            
        key1 := string(iter.Key())

        if key1 == fileNm{
            flag = 1
            break
        }
    }

    if flag == 1{
        vIter := versDB.NewIterator(nil, nil)
        for vIter.Next() {
            
            key2 := string(vIter.Key())
            if key2 == fileNm{
                vStr = string(vIter.Value())
                retStr = "CONTENTS "+cmd[1]+vStr
                vIter.Release()
                break
            }
        }

        tIter := timeDB.NewIterator(nil, nil)
        for tIter.Next() {
            
            key3 := string(tIter.Key())
            if key3 == fileNm{
                tStr = string(vIter.Value())
                retStr = " "+tStr

                tIter.Release()
                break
            }
        }

        fStr = string(iter.Value())
        fmt.Println(fStr)
        retStr = retStr+"\t"+fStr
        return retStr
        
    }else{
        return "ERR_FILE_NOT_FOUND"
    }
    return "OK"
}

func casFile(cmd []string, cmdLen int, conn net.Conn) (resp string){
    
    var flag int = 0
    var vStr string

    if !(cmdLen == 3 || cmdLen == 4){
        return "ERR_CMD_ERR"
    }

    fileNm := string(cmd[1])
    versNm := string(cmd[2])

    conn.Write([]byte("\n"))
    incMsg, err := bufio.NewReader(conn).ReadString('\n')
    if err != nil {
        return
    }
    fmt.Print("Received:", string(incMsg))


    fileDB, err := leveldb.OpenFile("./fileDB", nil)
    versDB, err := leveldb.OpenFile("./versDB", nil)
    timeDB, err := leveldb.OpenFile("./timeDB", nil)
    if err != nil {
        fmt.Println("DB Error!")
    }
    defer fileDB.Close()
    defer versDB.Close()
    defer timeDB.Close()

    //sz, _ := strconv.Atoi(cmd[2])
    //fmt.Println(sz)

    iter := versDB.NewIterator(nil, nil)
    //fmt.Println("DB")
    for iter.Next() {
            
        key := string(iter.Key())
        fmt.Println(key)
        if key == fileNm{
            flag = 1
            break
        }
    }

    if flag == 1{

        vIter := versDB.NewIterator(nil, nil)
        for vIter.Next() {
            
            key2 := string(vIter.Key())
            if key2 == fileNm{
                val2 := string(vIter.Value())
                vStr = fileNm+val2
                fmt.Println("@@",vStr)
                fmt.Println("@@@",versNm)
                if vStr == versNm{
                    err = fileDB.Put([]byte(cmd[1]), []byte(incMsg), nil)
                    if cmdLen == 4 {
                       err = timeDB.Put([]byte(cmd[1]), []byte(cmd[3]), nil)
                    }
                    return "OK "+cmd[1]+val2

                }else{
                    return "ERR_VERSION"
                }
            }
        }
    }else{
        return "ERR_FILE_NOT_FOUND"
    }
    return "OK"
}

func appendFile(cmd []string, cmdLen int, conn net.Conn) (resp string){
    
    var flag int = 0
    var vStr string
    var fStr string

    if !(cmdLen == 2 || cmdLen == 3){
        return "ERR_CMD_ERR"
    }

    fileNm := string(cmd[1])

    conn.Write([]byte("\n"))
    incMsg, err := bufio.NewReader(conn).ReadString('\n')
    if err != nil {
        return
    }
    fmt.Print("Received:", string(incMsg))


    fileDB, err := leveldb.OpenFile("./fileDB", nil)
    versDB, err := leveldb.OpenFile("./versDB", nil)
    timeDB, err := leveldb.OpenFile("./timeDB", nil)
    if err != nil {
        fmt.Println("DB Error!")
    }
    defer fileDB.Close()
    defer versDB.Close()
    defer timeDB.Close()

    //sz, _ := strconv.Atoi(cmd[2])
    //fmt.Println(sz)

    iter := fileDB.NewIterator(nil, nil)
    //fmt.Println("DB")
    for iter.Next() {
            
        key := string(iter.Key())
        fmt.Println(key)
        if key == fileNm{
            flag = 1
            fStr = string(iter.Value())
            break
        }
    }

    if flag == 1{

        vIter := versDB.NewIterator(nil, nil)
        //fmt.Println("DB")
        for vIter.Next() {
        
            key2 := string(iter.Key())
            
            if key2 == fileNm{
                vStr = string(vIter.Value())
                break
            }
        }

        //val := string(iter.Value())
        vers, _ := strconv.Atoi(vStr)
        vers = vers + 1
        vStr := strconv.Itoa(vers)
        err = versDB.Put([]byte(cmd[1]), []byte(vStr), nil)
        fmt.Println(vStr)

        fStr = fStr+incMsg
        fmt.Println(fStr)
        err = fileDB.Put([]byte(cmd[1]), []byte(fStr), nil)
        return "OK "+cmd[1]+vStr
        
    }else{
        return "ERR_FILE_NOT_FOUND"
    }
    return "OK"
}


func renameFile(cmd []string, cmdLen int) (resp string){
    
    if cmdLen > 2 || cmdLen < 2{
        return "ERR_CMD_ERR"
    }

    return " "
}


func deleteFile(cmd []string, cmdLen int) (resp string){

    var flag int = 0    
    fileNm := string(cmd[1])

    if cmdLen > 2 || cmdLen < 2{
        return "ERR_CMD_ERR"
    }

    fileDB, err := leveldb.OpenFile("./fileDB", nil)
    versDB, err := leveldb.OpenFile("./versDB", nil)
    timeDB, err := leveldb.OpenFile("./timeDB", nil)
    if err != nil {
        fmt.Println("DB Error!")
    }
    defer fileDB.Close()
    defer versDB.Close()
    defer timeDB.Close()

    iter := fileDB.NewIterator(nil, nil)
    for iter.Next() {
            
        key1 := string(iter.Key())

        if key1 == fileNm{
            flag = 1
            break
        }
    }

    if flag == 1{
        vIter := versDB.NewIterator(nil, nil)
        for vIter.Next() {
            
            key2 := string(vIter.Key())
            if key2 == fileNm{
                fmt.Println("@@")
                err = versDB.Delete([]byte(fileNm), nil)
                vIter.Release()
                break
            }
        }

        tIter := timeDB.NewIterator(nil, nil)
        for tIter.Next() {
            
            key3 := string(tIter.Key())
            if key3 == fileNm{
                fmt.Println("@@@",fileNm)
                err = timeDB.Delete([]byte(fileNm), nil)
                tIter.Release()
                break
            }
        }

        err = fileDB.Delete([]byte(fileNm), nil)
        return "OK"
        
    }else{
        return "ERR_FILE_NOT_FOUND"
    }
    return "OK"
}

func main() {

    reader := bufio.NewReader(os.Stdin)
    fmt.Println("----|Server Type|----")
    fmt.Println(" 1. Single Client")
    fmt.Println(" 2. Multi Client")
    fmt.Print("Select Server : ")
    ch, _ := reader.ReadString('\n')
    fmt.Println("---------------------")

    ch1 := "1\n"
    ch2 := "2\n"
    switch ch{
    case ch1: singleServer()
    case ch2: multiServer()
    default: fmt.Println("Invalid Choice!")
    }
    
}
