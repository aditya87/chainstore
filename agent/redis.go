package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func sendToRedis(command []byte) ([]byte, error) {
	redisConn, err := net.Dial("tcp", "localhost:"+os.Getenv("REDIS_PORT"))
	if err != nil {
		return nil, errors.Wrap(err, "Error connecting to redis host")
	}
	defer redisConn.Close()

	fmt.Fprintf(redisConn, string(command))

	reader := bufio.NewReader(redisConn)
	reply := []byte{}
	next, err := reader.Peek(1)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend")
	}

	switch string(next) {
	case `$`:
		reply, err = readBulkString(reader)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading reply from redis backend")
		}
	case `*`:
		reply, err = readArray(reader)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading reply from redis backend")
		}
	default:
		reply, err = reader.ReadBytes('\n')
		if err != nil {
			return nil, errors.Wrap(err, "Error reading reply from redis backend")
		}
	}

	return reply, nil
}

func readArray(r *bufio.Reader) ([]byte, error) {
	bytes, err := r.ReadBytes('\n')
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	len, err := strconv.Atoi(strings.Trim(string(bytes), "*\r\n"))
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	for i := 0; i < len; i++ {
		next, err := readBulkString(r)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading reply from redis backend:")
		}

		bytes = append(bytes, next...)
	}

	return bytes, nil
}

func readBulkString(r *bufio.Reader) ([]byte, error) {
	lenBytes, err := r.ReadBytes('\n')
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	contentBytes, err := r.ReadBytes('\n')
	if err != nil {
		return nil, errors.Wrap(err, "Error reading reply from redis backend:")
	}

	return append(lenBytes, contentBytes...), nil
}
