package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	CONN_HOST = "localhost"
	CONN_PORT = "3333"
	CONN_TYPE = "tcp"

	REDIS_HOST = "localhost:6973"
)

func main() {
	l, err := net.Listen(CONN_TYPE, CONN_HOST+":"+CONN_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer l.Close()

	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)
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
	defer conn.Close()
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
		return
	}

	redisConn, err := net.Dial("tcp", REDIS_HOST)
	if err != nil {
		fmt.Println("Error connecting to redis host:", err.Error())
		return
	}
	defer redisConn.Close()

	fmt.Fprintf(redisConn, string(buf))

	reader := bufio.NewReader(redisConn)
	reply := []byte{}
	next, err := reader.Peek(1)
	if err != nil {
		fmt.Println("Error reading reply from redis backend:", err.Error())
		return
	}

	switch string(next) {
	case `$`:
		reply, err = readBulkString(reader)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	case `*`:
		reply, err = readArray(reader)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	default:
		reply, err = reader.ReadBytes('\n')
		if err != nil {
			fmt.Println(errors.Wrap(err, "Error reading reply from redis backend:").Error())
			return
		}
	}

	conn.Write(reply)
}

func readArray(r *bufio.Reader) ([]byte, error) {
	bytes, err := r.ReadBytes('\n')
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	len, err := strconv.Atoi(strings.Trim(string(bytes), "*\r\n"))
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	for i := 0; i < len; i++ {
		next, err := readBulkString(r)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading reply from redis backend:")
		}

		bytes = append(bytes, next...)
	}

	return bytes, nil
}

func readBulkString(r *bufio.Reader) ([]byte, error) {
	lenBytes, err := r.ReadBytes('\n')
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	contentBytes, err := r.ReadBytes('\n')
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	return append(lenBytes, contentBytes...), nil
}
