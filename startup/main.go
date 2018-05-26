package main

import (
	"log"
	"os/exec"
)

func main() {
	cmd := exec.Command("/app/redis-start")
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Could not run redis-start: %v\n", err)
		return
	}

	cmd = exec.Command("/app/agent-start")
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Could not run agent-start: %v\n", err)
		return
	}
}
