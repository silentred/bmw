package queue

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestBeanstalk(t *testing.T) {
	config := &BeanstalkConfig{
		Host: "localhost:11300",
		Tube: "default",
	}

	pusher := NewBeanstalkProducerPusher(config)

	go func() {
		data := pusher.Wait()
		log.Println(string(data))
	}()

	err := pusher.Push([]byte("test!!!"))
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second * 1)

}

func TestPanic(t *testing.T) {
	defer func() {
		//time.Sleep(1 * time.Second)
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	go func() {
		fmt.Println("after")
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(err)
			}
		}()

		panic("panicking")

	}()

	// panic("outside")

	time.Sleep(1 * time.Second)
}
