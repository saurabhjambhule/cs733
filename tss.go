package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"os"
	"time"
)

func main() {
for i := 0; i < 3; i++ {
	go ex()
	time.Sleep(1 * time.Second)


}

reader := bufio.NewReader(os.Stdin)
          ch, _ := reader.ReadString('\n')
          fmt.Println(ch)
}


// Simple serial check of getting and setting
func ex() {
	name := "hi.txt"
	contents := "bye"
	exptime := 300000
	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		//t.Error(err.Error()) // report error through testing framework
	}

	scanner := bufio.NewScanner(conn)

	// Write a file
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", name, len(contents), exptime, contents)
	scanner.Scan() // read first line
	resp := scanner.Text() // extract the text from the buffer
	fmt.Println(resp)
	arr := strings.Split(resp, " ") // split into OK and <version>
	expect(arr[0], "OK")
	version, err := strconv.ParseInt(arr[1], 10, 64) // parse version as number
	if err != nil {
	fmt.Println("Non-numeric version found")
	}
	version = version
	fmt.Fprintf(conn, "read %v\r\n", name) // try a read now
	scanner.Scan()
    resp = scanner.Text() // extract the text from the buffer
	fmt.Println(resp)
	arr = strings.Split(scanner.Text(), " ")
	expect(arr[0], "CONTENTS")
	expect(arr[1], fmt.Sprintf("%v", version)) // expect only accepts strings, convert int version to string
	expect(arr[2], fmt.Sprintf("%v", len(contents)))	
	scanner.Scan()
	expect(contents, scanner.Text())
}

func expect(a string, b string) {
	if a != b {
		fmt.Println(fmt.Sprintf("Expected %v, found %v", b, a)) // t.Error is visible when running `go test -verbose`
	}else{
		//fmt.Println("PASS")
	}
}


