package lib

import (
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
)

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// GetPort returns the usable port in localhost
func GetPort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// RandomString returns the random string with length of n
func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

// CurrentWorkload returns the current workload of the machine
func CurrentWorkload() int {
	// TODO
	return 50
}

// GetPrivilege returns the privilege of API according to workload
func GetPrivilege(workload int) int {
	return 50
}

// GetExternalIP returns external ip
func GetExternalIP() string {
	resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(bytes)
}
