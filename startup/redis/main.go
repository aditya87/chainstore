package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	fmt.Println("Starting redis server...")
	cmd := exec.Command("redis-server", "--port", "6379")
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Could not start redis-server: %v\n", err)
		return
	}

	time.Sleep(2 * time.Second)
	redisPid := cmd.Process.Pid
	redisPidFile, err := os.OpenFile("/app/redis.pid", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Fatalf("Could not open/create PID file for redis-server: %v\n", err)
		return
	}

	_, err = redisPidFile.Write([]byte(fmt.Sprintf("%d", redisPid)))
	if err != nil {
		log.Fatalf("Could not write PID file for redis-server: %v\n", err)
		return
	}
}
