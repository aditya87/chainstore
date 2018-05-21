package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	l, err := net.Listen("tcp", "localhost:"+os.Getenv("PORT"))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer l.Close()

	err = os.Mkdir("/store", os.ModeDir)
	if err != nil {
		fmt.Println("Error creating store directory:", err.Error())
		os.Exit(1)
	}

	fmt.Println("Listening on " + "localhost:" + os.Getenv("PORT"))
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			return
		}

		err = writeBlock(buf)
		if err != nil {
			fmt.Printf("Error writing block to local store: %v\n", err)
			return
		}

		reply, err := sendToRedis(buf)
		if err != nil {
			fmt.Printf("Error sending command to redis: %v\n", err)
			return
		}

		conn.Write(reply)
	}
}
