#! /bin/bash

redis-server --port 6973 --maxclients 10000 &
echo "Starting Redis server..."
sleep 5
echo "Starting repl process..."
go run main.go
