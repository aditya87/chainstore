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

	files, err := ioutil.ReadDir("/store")
	if err != nil {
		return errors.Wrap(err, "Error reading /store directory")
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
