package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Agent struct {
	recorder    MerkleWriter
	backendPort string
}

const storeDir = "/store"

func main() {
	logFile, err := os.OpenFile("/app/agent.log", os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalln("Could not open logfile")
	}

	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	agent := Agent{
		recorder: MerkleWriter{
			Store:      storeDir,
			BlockMutex: &sync.Mutex{},
		},
		backendPort: "6379",
	}

	agent.RestoreFromDisk()
	if _, err := os.Stat(storeDir); os.IsNotExist(err) {
		err = os.Mkdir(storeDir, os.ModeDir)
		if err != nil {
			log.Fatalln("Error creating store directory:", err.Error())
		}
	}

	go agent.MonitorRedis()

	rChan := make(chan error, 1)
	go agent.ListenRepl(rChan)
	<-rChan

	agent.ListenRedis()
}

func (a Agent) ListenRedis() {
	l, err := net.Listen("tcp", "localhost:3000")
	if err != nil {
		log.Fatalln("Error listening:", err.Error())
	}
	defer l.Close()

	log.Println("Listening on " + "localhost:3000")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		go a.HandleRedisRequest(conn)
	}
}

func (a Agent) ListenRepl(rChan chan error) {
	lr, err := net.Listen("tcp", "localhost:3001")
	if err != nil {
		log.Fatalln("Error listening:", err.Error())
	}
	defer lr.Close()

	rChan <- nil
	log.Println("Listening on " + "localhost:3001")
	for {
		replConn, err := lr.Accept()
		if err != nil {
			log.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		go a.HandleReplicationRequest(replConn)
	}
}

func (a Agent) HandleRedisRequest(conn net.Conn) {
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			return
		}

		err = a.recorder.WriteBlockCmd(buf)
		if err != nil {
			log.Printf("Error writing block to local store: %v\n", err)
			return
		}

		reply, err := a.sendToRedis(buf)
		if err != nil {
			log.Printf("Error sending command to redis: %v\n", err)
			return
		}

		conn.Write(reply)
	}
}

func (a Agent) HandleReplicationRequest(conn net.Conn) {
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("Error reading: %v\n", err)
			return
		}

		replBlock := strings.Trim(strings.Split(string(buf), "REPL")[1], "ENDREPL")
		log.Println("REPL block: ", replBlock)
		prevHash := strings.Trim(strings.Split(replBlock, ":")[3], "\r\nprev_time")
		cmd := strings.Trim(strings.Split(replBlock, ":")[1], "\r\ntime")

		myLastBlock, err := a.recorder.ReadLastBlock()
		if err != nil {
			log.Printf("Error reading last block: %v\n", err)
			return
		}

		myLastHash := fmt.Sprintf("%x", sha256.Sum256([]byte(myLastBlock)))
		if prevHash != myLastHash {
			log.Printf("wrong hash: %v, expected: %v\n", prevHash, myLastHash)
			return
		}

		err = a.recorder.WriteBlock([]byte(replBlock))
		if err != nil {
			log.Printf("Error recording: %v\n", err)
			return
		}

		log.Printf("Sending command: %v\n", cmd)
		reply, err := a.sendToRedis([]byte(fmt.Sprintf("%s\r\n", cmd)))
		if err != nil {
			log.Printf("Error sending command to redis: %v\n", err)
			return
		}
		log.Printf("Received reply: %v\n", string(reply))

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

func (a Agent) isRedisAlive() bool {
	cmd := exec.Command("pgrep", "-f", "redis")
	err := cmd.Run()
	return err == nil
}

func (a Agent) MonitorRedis() {
	for {
		if !a.isRedisAlive() {
			log.Println("Could not find Redis process, restarting...")
			a.startRedis()
		}
		time.Sleep(2 * time.Second)
	}
}

func (a Agent) startRedis() error {
	cmd := exec.Command("/app/redis-start")
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "Could not start redis-server")
	}

	err = a.RestoreFromDisk()
	if err != nil {
		return errors.Wrap(err, "Could not restore from disk")
	}

	return nil
}
