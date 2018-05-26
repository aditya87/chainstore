package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	. "github.com/aditya87/chainstore/utils"
	"github.com/go-redis/redis"
)

var rClient *redis.Client

func main() {
	Setup()

	// Test that PIDs are written to /app/
	// TestPIDs()

	// Test that agent proxies to redis server
	TestProxy()

	// Test that agent writes incoming transactions to Merkle chain on disk
	// TestMerkleWrites()

	// Test that agent can restore redis server from Merkle chain upon restart
	// TestRestoreFromDisk()

	// Test that agent restarts and restores redis server if it gets killed
	// TestRestoreAfterRedisKill()

	// Test that agent replicates transactions from other agents
	TestReplication()
}

func Setup() {
	fmt.Println("Starting store...")
	cmd := exec.Command("/app/startup")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Could not start store: %v\n", err)
		return
	}

	time.Sleep(6 * time.Second)

	fmt.Println("Creating redis client...")
	rClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:3000",
		DB:       0,
		PoolSize: 10,
	})
}

func TestPIDs() {
	agentPid, err := ioutil.ReadFile("/app/agent.pid")
	TAssert(IsNil, err)

	_, err = strconv.Atoi(string(agentPid))
	TAssert(IsNil, err)

	redisPid, err := ioutil.ReadFile("/app/redis.pid")
	TAssert(IsNil, err)

	_, err = strconv.Atoi(string(redisPid))
	TAssert(IsNil, err)

	TAssert(IsNotNil, redisPid)
}

func TestProxy() {
	_, err := rClient.Set("k1", "value1", 0).Result()
	TAssert(IsNil, err)

	v1, err := rClient.Get("k1").Result()
	TAssert(IsNil, err)
	TAssert(Equals, v1, "value1")

	_, err = rClient.SAdd("k2", "value2").Result()
	TAssert(IsNil, err)

	v2, err := rClient.SMembers("k2").Result()
	TAssert(IsNil, err)
	TAssert(Equals, v2, []string{"value2"})
}

func TestMerkleWrites() {
	f, err := ioutil.ReadFile("/store/t0")
	TAssert(IsNil, err)

	block := string(f)
	TAssert(ContainsSubstring, block, "command:*3\r\n$3\r\nset\r\n$2\r\nk1\r\n$6\r\nvalue1")
	TAssert(ContainsSubstring, block, "time:")
	TAssert(ContainsSubstring, block, "prev_hash:init")

	f, err = ioutil.ReadFile("/store/t1")
	TAssert(IsNil, err)

	prevHash := fmt.Sprintf("%x", sha256.Sum256([]byte(block)))
	prevTime := strings.Trim(strings.Split(block, ":")[2], "prev_hash")
	block = string(f)
	TAssert(ContainsSubstring, block, "command:*3\r\n$4\r\nsadd\r\n$2\r\nk2\r\n$6\r\nvalue2")
	TAssert(ContainsSubstring, block, "time:")
	TAssert(ContainsSubstring, block, "prev_hash:"+prevHash)
	TAssert(ContainsSubstring, block, "prev_time:"+prevTime)
}

func TestRestoreFromDisk() {
	agentPidBytes, err := ioutil.ReadFile("/app/agent.pid")
	TAssert(IsNil, err)

	agentPid, err := strconv.Atoi(string(agentPidBytes))
	TAssert(IsNil, err)

	agentProcess, err := os.FindProcess(agentPid)
	TAssert(IsNil, err)

	err = agentProcess.Kill()
	TAssert(IsNil, err)

	redisPidBytes, err := ioutil.ReadFile("/app/redis.pid")
	TAssert(IsNil, err)

	redisPid, err := strconv.Atoi(string(redisPidBytes))
	TAssert(IsNil, err)

	redisProcess, err := os.FindProcess(redisPid)
	TAssert(IsNil, err)

	err = redisProcess.Kill()
	TAssert(IsNil, err)

	Setup()
	v1, err := rClient.Get("k1").Result()
	TAssert(IsNil, err)
	TAssert(Equals, v1, "value1")

	v2, err := rClient.SMembers("k2").Result()
	TAssert(IsNil, err)
	TAssert(Equals, v2, []string{"value2"})
}

func TestRestoreAfterRedisKill() {
	redisPidBytes, err := ioutil.ReadFile("/app/redis.pid")
	TAssert(IsNil, err)

	redisPid, err := strconv.Atoi(string(redisPidBytes))
	TAssert(IsNil, err)

	redisProcess, err := os.FindProcess(redisPid)
	TAssert(IsNil, err)

	err = redisProcess.Kill()
	TAssert(IsNil, err)

	TAssertEventual(func() bool {
		redisPidBytes, _ = ioutil.ReadFile("/app/redis.pid")
		newRedisPid, _ := strconv.Atoi(string(redisPidBytes))
		return newRedisPid != redisPid
	}, 20)

	v1, err := rClient.Get("k1").Result()
	TAssert(IsNil, err)
	TAssert(Equals, v1, "value1")

	v2, err := rClient.SMembers("k2").Result()
	TAssert(IsNil, err)
	TAssert(Equals, v2, []string{"value2"})
}

func TestReplication() {
	f, err := ioutil.ReadFile("/store/t1")
	TAssert(IsNil, err)

	block := string(f)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(block)))
	replTransaction := fmt.Sprintf("REPLcommand:*3\r\n$3\r\nset\r\n$1\r\ny\r\n$2\r\n10\r\n\r\ntime:%d\r\nprev_hash:%s\r\nprev_time:%d\r\nENDREPL", time.Now().UnixNano(), hash, time.Now().UnixNano())

	replConn, err := net.Dial("tcp", "localhost:3001")
	TAssert(IsNil, err)
	defer replConn.Close()

	fmt.Fprintf(replConn, replTransaction)
	reader := bufio.NewReader(replConn)

	var bytes []byte
	go func() {
		bytes, err = reader.ReadBytes('\n')
	}()

	TAssertEventual(func() bool { return string(bytes) == "+OK\r\n" }, 3)

	f, err = ioutil.ReadFile("/store/t2")
	TAssert(IsNil, err)

	block = string(f)
	TAssert(ContainsSubstring, block, "command:*3\r\n$3\r\nset\r\n$1\r\ny\r\n$2\r\n10")
	TAssert(ContainsSubstring, block, "time:")
	TAssert(ContainsSubstring, block, "prev_hash:"+hash)
	TAssert(ContainsSubstring, block, "prev_time:")
}
