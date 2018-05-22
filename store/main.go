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
	cmd := exec.Command("redis-server", "--port", os.Getenv("REDIS_PORT"))
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

	fmt.Println("Starting agent...")
	cmd = exec.Command("/app/agent")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Could not start agent: %v\n", err)
		return
	}

	time.Sleep(2 * time.Second)
	agentPid := cmd.Process.Pid
	agentPidFile, err := os.OpenFile("/app/agent.pid", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Fatalf("Could not open/create PID file for agent: %v\n", err)
		return
	}

	_, err = agentPidFile.Write([]byte(fmt.Sprintf("%d", agentPid)))
	if err != nil {
		log.Fatalf("Could not write PID file for agent: %v\n", err)
		return
	}
}
