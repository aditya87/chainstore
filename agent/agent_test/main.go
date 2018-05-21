package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/aditya87/chainstore/utils"
	"github.com/go-redis/redis"
)

var rClient *redis.Client

func main() {
	fmt.Println("Starting redis server...")
	cmd := exec.Command("redis-server", "--port", "7777")
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Could not start redis-server: %v\n", err)
		return
	}
	time.Sleep(3 * time.Second)

	fmt.Println("Setting up environment...")
	os.Setenv("REDIS_PORT", "7777")
	os.Setenv("PORT", "3000")

	go func() {
		fmt.Println("Starting agent...")
		cmd = exec.Command("/app/agent")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout
		err := cmd.Run()

		if err != nil {
			log.Fatalf("Could not start agent: %v\n", err)
			return
		}
	}()

	time.Sleep(2 * time.Second)

	fmt.Println("Creating redis client...")
	rClient = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf(
			"localhost:%s",
			os.Getenv("PORT")),
		DB:       0,
		PoolSize: 10,
	})

	// Test that agent proxies to redis server
	TestProxy()

	// Test that agent writes incoming transactions to Merkle chain on disk
	TestMerkleWrites()
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
