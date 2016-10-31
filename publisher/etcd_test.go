package publisher

import (
	"testing"
	"time"
)

func TestPublisher(t *testing.T) {
	srv := NewService("1", "web", "localhost:8080")
	pub := NewEtcdPublisher([]string{"http://127.0.0.1:2379"}, 10)

	err := pub.Register(srv)
	if err != nil {
		panic(err)
	}

	go func() {
		select {
		case <-time.After(6 * time.Second):
			err := pub.Unregister(srv)
			if err != nil {
				panic(err)
			}
		}
	}()

	// Heartbeat
	pub.Heartbeat(srv)

	// TODO

}
