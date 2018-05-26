package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	fmt.Println("Creating log file...")
	_, err := os.OpenFile("/app/agent.log", os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalf("Could not create logfile: %v\n", err)
	}

	fmt.Println("Starting agent...")
	cmd := exec.Command("/app/agent")
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
