package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

type Agent struct {
	recorder    MerkleWriter
	backendPort string
}

const storeDir = "/store"

func main() {
	l, err := net.Listen("tcp", "localhost:"+os.Getenv("PORT"))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer l.Close()

	if _, err := os.Stat(storeDir); os.IsNotExist(err) {
		err = os.Mkdir(storeDir, os.ModeDir)
		if err != nil {
			fmt.Println("Error creating store directory:", err.Error())
			os.Exit(1)
		}
	}

	agent := Agent{
		recorder: MerkleWriter{
			Store:      storeDir,
			BlockMutex: &sync.Mutex{},
		},
		backendPort: os.Getenv("REDIS_PORT"),
	}

	agent.RestoreFromDisk()

	fmt.Println("Listening on " + "localhost:" + os.Getenv("PORT"))
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}

		go agent.HandleRequest(conn)
	}
}

func (a Agent) HandleRequest(conn net.Conn) {
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			return
		}

		err = a.recorder.WriteBlock(buf)
		if err != nil {
			fmt.Printf("Error writing block to local store: %v\n", err)
			return
		}

		reply, err := a.sendToRedis(buf)
		if err != nil {
			fmt.Printf("Error sending command to redis: %v\n", err)
			return
		}

		conn.Write(reply)
	}
}

func (a Agent) sendToRedis(command []byte) ([]byte, error) {
	redisConn, err := net.Dial("tcp", "localhost:"+a.backendPort)
	if err != nil {
		return nil, errors.Wrap(err, "Error connecting to redis host")
	}
	defer redisConn.Close()

	fmt.Fprintf(redisConn, string(command))

	reader := bufio.NewReader(redisConn)
	reply := []byte{}
	next, err := reader.Peek(1)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend")
	}

	switch string(next) {
	case `$`:
		reply, err = a.readBulkString(reader)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading reply from redis backend")
		}
	case `*`:
		reply, err = a.readArray(reader)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading reply from redis backend")
		}
	default:
		reply, err = reader.ReadBytes('\n')
		if err != nil {
			return nil, errors.Wrap(err, "Error reading reply from redis backend")
		}
	}

	return reply, nil
}

func (a Agent) readArray(r *bufio.Reader) ([]byte, error) {
	bytes, err := r.ReadBytes('\n')
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	len, err := strconv.Atoi(strings.Trim(string(bytes), "*\r\n"))
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	for i := 0; i < len; i++ {
		next, err := a.readBulkString(r)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading reply from redis backend:")
		}

		bytes = append(bytes, next...)
	}

	return bytes, nil
}

func (a Agent) readBulkString(r *bufio.Reader) ([]byte, error) {
	lenBytes, err := r.ReadBytes('\n')
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	if string(lenBytes) == "$-1\r\n" {
		return lenBytes, nil
	}

	contentBytes, err := r.ReadBytes('\n')
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	return append(lenBytes, contentBytes...), nil
}

func (a Agent) RestoreFromDisk() error {
	cmds, err := a.recorder.ReadBlocks()
	if err != nil {
		return errors.Wrap(err, "Could not read blocks from disk")
	}

	for _, cmd := range cmds {
		_, err := a.sendToRedis(cmd)
		if err != nil {
			return errors.Wrap(err, "Could not restore from disk")
		}
	}

	return nil
}
