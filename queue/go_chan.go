package queue

import (
	"fmt"
	"time"
)

const minBuf = 1000
const timeout = 5

func NewGoChannelPusherWaiter(buf int) *GoChannel {
	if buf < minBuf {
		buf = minBuf
	}
	jobPool := make(chan []byte, buf)
	return &GoChannel{jobQueue: jobPool}
}

type GoChannel struct {
	jobQueue chan []byte
}

func (channel *GoChannel) Push(payload []byte) error {
	var err error = nil

	clock := time.After(timeout * time.Second)

	select {
	case channel.jobQueue <- payload:
	case <-clock:
		err = fmt.Errorf("timeout pushing payload to jobPool")
	}

	return err
}

func (channel *GoChannel) Wait() []byte {
	b := <-channel.jobQueue
	return b
}
