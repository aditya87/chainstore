package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	. "github.com/aditya87/chainstore/utils"
	"github.com/go-redis/redis"
)

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

	fmt.Println("Starting agent...")
	cmd = exec.Command("/app/agent")
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Could not start agent: %v\n", err)
		return
	}

	time.Sleep(3 * time.Second)

	fmt.Println("Creating redis client...")
	rClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf(
			"localhost:%s",
			os.Getenv("PORT")),
		DB: 0,
	})

	fmt.Println("Testing...")
	_, err = rClient.Set("k1", "value1", 0).Result()
	TAssert(err, IsNil)

	v1, err := rClient.Get("k1").Result()
	TAssert(err, IsNil)
	TAssert(v1, Equals, "value1")

	_, err = rClient.SAdd("k2", "value2", 0).Result()
	TAssert(err, IsNil)

	v2, err := rClient.SMembers("k2").Result()
	TAssert(err, IsNil)
	TAssert(v2, Equals, []string{"value2"})
}
