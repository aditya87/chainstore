package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/go-redis/redis"
	. "github.com/onsi/gomega"
)

func main() {
	fmt.Println("Starting redis server...")
	cmd := exec.Command("redis-server", "--port", "7777")
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Could not start redis-server: %v\n", err)
	}
	time.Sleep(3 * time.Second)

	fmt.Println("Setting up environment...")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "7777")
	os.Setenv("PORT", "3000")

	cmd = exec.Command("/app/agent")
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Could not start agent: %v\n", err)
	}

	rClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf(
			"%s:%s",
			os.Getenv("REDIS_HOST"),
			os.Getenv("PORT")),
		DB: 0,
	})

	rClient.Set("k1", "value1", 0)
	v1, err := rClient.Get("k1").Result()
	Expect(err).NotTo(HaveOccurred())
	Expect(v1).To(Equal("value1"))
	fmt.Println("Tested SET/GET")

	rClient.SAdd("k2", "value2", 0)
	v2, err := rClient.SMembers("k2").Result()
	Expect(err).NotTo(HaveOccurred())
	Expect(v2).To(Equal([]string{"value2"}))
	fmt.Println("Tested SADD/SMEMBERS")
}
