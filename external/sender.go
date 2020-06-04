package main

import (
	"fmt"
	"net"
	"os"
)

// Constants
const (
	ConnHost = "localhost"
	ConnPort = "3333"
	ConnType = "tcp"
)

func main() {
	l, _ := net.Listen(ConnType, ConnHost+":"+ConnPort)
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("error: " + err.Error())
			os.Exit(1)
		}
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("error: " + err.Error())
	}
	fmt.Println(string(buffer))
	conn.Write([]byte("MEssage recevied"))
	conn.Close()
}
