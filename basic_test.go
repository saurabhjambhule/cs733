package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)


func TestTCPSimple(t *testing.T) {

	go serverMain()
	time.Sleep(1 * time.Second) 
	
	name := "TestFile"
	contents := "This is for testing purpose"
	exptime := 500000
	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		t.Error(err.Error()) 
	}

	scanner := bufio.NewScanner(conn)

	//Write a File
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", name, len(contents), exptime, contents)

	//Check for response as :OK"
	scanner.Scan()                 
	resp := scanner.Text()          
	arr := strings.Split(resp, " ") 
	expect(t, arr[0], "OK")

	//Check is version is numeric or not
	version1, err1 := strconv.ParseInt(arr[1], 10, 64) 
	fmt.Println(version1)

	if err1 != nil {
		t.Error("Non-numeric version found")
	}

	//Read a File
	fmt.Fprintf(conn, "read %v\r\n", name) 
	
	//Check is content and length matches to provided data
	scanner.Scan()
	arr = strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	expect(t, arr[1], fmt.Sprintf("%v", version1)) 
	expect(t, arr[2], fmt.Sprintf("%v", len(contents)))
	scanner.Scan()
	expect(t, contents, scanner.Text())

fmt.Println("@@@")

	//Compare and swap file contents
	fmt.Fprintf(conn, "cas %v %v %v %v\r\n%v\r\n", name, version1, len(contents), exptime, contents)

	//Check for response as :OK"
	scanner.Scan()                 
	resp = scanner.Text()    
	fmt.Println(resp)      
	arr = strings.Split(resp, " ") 
	expect(t, arr[0], "OK")

	fmt.Println("@@@")


	

}

// Useful testing function
func expect(t *testing.T, a string, b string) {
	if a != b {
		t.Error(fmt.Sprintf("Expected %v, found %v", b, a)) // t.Error is visible when running `go test -verbose`
	}
}

func expect1(t *testing.T, a string, b string) {
	if a == b {
		t.Error(fmt.Sprintf("Expected %v, found %v", b, a)) // t.Error is visible when running `go test -verbose`
	}
}
