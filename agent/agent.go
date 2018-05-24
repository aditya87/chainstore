package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

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

	go agent.MonitorRedis()

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

func (a Agent) findRedisProcess() *os.Process {
	redisPidBytes, err := ioutil.ReadFile("/app/redis.pid")
	if err != nil {
		log.Fatalf("Could not read Redis PID file: %s", err.Error())
	}

	redisPid, err := strconv.Atoi(string(redisPidBytes))
	if err != nil {
		log.Fatalf("Could not parse PID from Redis PID file: %s", err.Error())
	}

	redisProcess, err := os.FindProcess(redisPid)
	if err != nil {
		log.Fatalf("Redis process monitor failed: %s", err.Error())
	}

	return redisProcess
}

func (a Agent) isAlive(proc *os.Process) bool {
	return proc.Signal(syscall.Signal(0)) == nil
}

func (a Agent) MonitorRedis() {
	redisProcess := a.findRedisProcess()
	for {
		if !a.isAlive(redisProcess) {
			a.startRedis()
			redisProcess = a.findRedisProcess()
		}
		time.Sleep(2 * time.Second)
	}
}

func (a Agent) startRedis() error {
	cmd := exec.Command("redis-server", "--port", os.Getenv("REDIS_PORT"))
	err := cmd.Start()
	if err != nil {
		return errors.Wrap(err, "Could not start redis-server")
	}

	time.Sleep(2 * time.Second)
	redisPid := cmd.Process.Pid
	redisPidFile, err := os.OpenFile("/app/redis.pid", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return errors.Wrap(err, "Could not open/create PID file for redis-server")
	}

	_, err = redisPidFile.Write([]byte(fmt.Sprintf("%d", redisPid)))
	if err != nil {
		return errors.Wrap(err, "Could not write PID file for redis-server")
	}

	err = a.RestoreFromDisk()
	if err != nil {
		return errors.Wrap(err, "Could not restore from disk")
	}

	return nil
}
