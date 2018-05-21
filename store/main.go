package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("Starting redis server...")
	cmd := exec.Command("redis-server", "--port", os.Getenv("REDIS_PORT"))
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Could not start redis-server: %v\n", err)
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
}
