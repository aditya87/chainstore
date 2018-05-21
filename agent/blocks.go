package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type MerkleWriter struct {
	Store      string
	BlockMutex *sync.Mutex
}

func (m MerkleWriter) WriteBlock(cmd []byte) error {
	m.BlockMutex.Lock()
	defer m.BlockMutex.Unlock()

	if !m.isWrite(cmd) {
		return nil
	}

	files, err := ioutil.ReadDir(m.Store)
	if err != nil {
		return errors.Wrapf(err, "Error reading %s directory", m.Store)
	}

	blockName := fmt.Sprintf("%s/t%d", m.Store, len(files))
	block, err := os.Create(blockName)
	if err != nil {
		return errors.Wrap(err, "Error creating block file")
	}

	var prevHash string
	var prevTime string
	if len(files) == 0 {
		prevHash = "init"
	} else {
		prevBlock, err := ioutil.ReadFile(fmt.Sprintf("%s/t%d", m.Store, len(files)-1))
		if err != nil {
			return errors.Wrap(err, "Error reading previous block file")
		}
		prevHash = fmt.Sprintf("%x", sha256.Sum256(prevBlock))
		prevTime = strings.Trim(strings.Split(string(prevBlock), ":")[2], "prev_hash")
	}

	blockContent := fmt.Sprintf("command:%s\r\ntime:%d\r\nprev_hash:%s\r\nprev_time:%s",
		string(cmd),
		time.Now().UnixNano(),
		prevHash,
		prevTime)

	_, err = block.Write([]byte(blockContent))
	if err != nil {
		return errors.Wrap(err, "Error writing to block file")
	}

	return nil
}

func (m MerkleWriter) isWrite(cmd []byte) bool {
	inst := strings.Split(string(cmd), "\r\n")[2]
	return inst == "append" ||
		strings.Contains(inst, "pop") ||
		strings.Contains(inst, "push") ||
		strings.Contains(inst, "set") ||
		strings.Contains(inst, "incr") ||
		strings.Contains(inst, "decr") ||
		strings.Contains(inst, "expire") ||
		strings.Contains(inst, "flush") ||
		strings.Contains(inst, "rem") ||
		strings.Contains(inst, "del") ||
		strings.Contains(inst, "trim") ||
		strings.Contains(inst, "persist") ||
		strings.Contains(inst, "rename") ||
		strings.Contains(inst, "add")
}
