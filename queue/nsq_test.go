package queue

import "testing"

func TestPub(t *testing.T) {
	publishNSQ("write_test", []byte("test !!!!"))

	handleNSQ()
}
