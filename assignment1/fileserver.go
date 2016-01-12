package assignment1

import (
	"net"
	"os"
	"fmt"
)

func getIPAddr() {
	if len(os.Args) != 2 {
		fmt.Fprintf("Please provide IP address!")
		os.Exit(1)
	}

	ipAddr := net.ParseIP(os.Args[1])
	
	if ipAddr == nil {
		fmt.Println("Invalid address!")
	} 
	else {
		fmt.Println("The address is ", ipAddr.String())
	}
	os.Exit(0)
}

