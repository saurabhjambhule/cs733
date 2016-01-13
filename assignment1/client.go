package main

import (
    "fmt"
    "net"
    "os"
    "bufio"
    _ "strconv"
    _ "strings"
)


func clientMain() {

    conn, err := net.Dial("tcp", "127.0.0.1:8080")
    checkError(err)
    for { 
        reader := bufio.NewReader(os.Stdin)
        flag := cmdMenu(conn)
        
        //fmt.Println("@@@", flag)
        if flag == 1{
            fmt.Print("Enter Command: ")
        }else{
            fmt.Print("")
            flag = 1
        }
        text, _ := reader.ReadString('\n')
        fmt.Fprintf(conn, text + "\n")
        message, _ := bufio.NewReader(conn).ReadString('\n')
        //fmt.Println("@@", tmp)
        fmt.Println("Server Respond: "+message)
    }
    os.Exit(0)
}


func cmdMenu(conn net.Conn)(flag int){
    reader := bufio.NewReader(os.Stdin)
    fmt.Println("-------|File Menu|-------")
    fmt.Println(" 1. Write      4. Append")
    fmt.Println(" 2. Read       5. Rename")
    fmt.Println(" 3. Comp&Swap  6. Delete")
    fmt.Print("Enter your choice : ")
    ch, _ := reader.ReadString('\n')
    fmt.Println("-------------------------")

    ch1 := "1\n"
    ch2 := "2\n"
    ch3 := "3\n"
    ch4 := "4\n"
    ch5 := "5\n"
    ch6 := "6\n"
    
    switch ch{
        case ch1:
            fmt.Println("write <filename> <numbytes> [<exptime>]")
            fmt.Println("<content bytes>")
            fmt.Print("Enter Command: ")
            text, _ := reader.ReadString('\n')
            fmt.Fprintf(conn, text + "\n")
            message, _ := bufio.NewReader(conn).ReadString('\n')
            fmt.Println(" "+message)
            return -1

        case ch2:
            fmt.Println("read <filename>")
            return 1
    
        case ch3:
            fmt.Println("cas <filename> <version> <numbytes> [<exptime>]")
            fmt.Println("<content bytes>")
            fmt.Print("Enter Command: ")
            text, _ := reader.ReadString('\n')
            fmt.Fprintf(conn, text + "\n")
            message, _ := bufio.NewReader(conn).ReadString('\n')
            fmt.Println(" "+message)
            return -1

        case ch4:
            fmt.Println("append <filename> <numbytes> [<exptime>]")
            fmt.Println("<content bytes>")
            fmt.Print("Enter Command: ")
            text, _ := reader.ReadString('\n')
            fmt.Fprintf(conn, text + "\n")
            message, _ := bufio.NewReader(conn).ReadString('\n')
            fmt.Println(" "+message)
            return -1

        case ch5:
            fmt.Println("rename <filename> <newfilename>")
            return 1

        case ch6:
            fmt.Println("delete <filename>")
            return 1

        default: fmt.Println("Invalid Choice!")
    }
    return
    
}


func checkError(err error) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
        os.Exit(1)
    }
}


func main() {
    clientMain()
}