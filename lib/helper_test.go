package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestGetPort(t *testing.T) {
	port := GetPort()
	fmt.Println("The free port is", port)
}

func TestNewHttpRequest(t *testing.T) {
	query := map[string]string{
		"aa": "bb",
	}
	var body []byte
	req, err := NewHTTPReqeust("GET", "http://baidu.com/search", query, query, body)
	if err != nil {
		fmt.Println(err)
	}

	err = req.Write(os.Stdout)
	if err != nil {
		fmt.Println(err)
	}
}

// func TestConcurrentHTTPGet(t *testing.T) {
// 	limit := make(chan int, 1)
// 	for i := 0; i < 3; i++ {
// 		go func(i int) {
// 			fmt.Println("start ", i)
// 			limit <- 1
// 			b, err := HTTPGet("http://fame.baidu.com/", nil, nil, 3)
// 			if err != nil {
// 				fmt.Println(err)
// 			}
// 			fmt.Println("length of body is ", len(b), i)
// 			<-limit
// 		}(i)
// 	}

// 	time.Sleep(5 * time.Second)
// }

func TestPanic(t *testing.T) {
	//start()
}

func start() {
	for {
		fmt.Println("start for loop")
		b := doPanic()
		fmt.Println(string(b))
		fmt.Println("end for loop")
		time.Sleep(2 * time.Second)
	}
}

func doPanic() []byte {
	var b []byte
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("catch panic: ", err)
			b = []byte("return []byte")
		}
	}()

	fmt.Println("start panic")
	panic("A panic")
	b = []byte("sdf")
	return b
}

func TestCommand(t *testing.T) {
	c := exec.Command("ls", "/")

	//b, _ := c.Output()

	reader, err := c.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}

	if err = c.Start(); err != nil {
		fmt.Println(err)
	}

	b, _ := ioutil.ReadAll(reader)

	if err = c.Wait(); err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(b))

}

func TestTime(t *testing.T) {
	now := time.Now()
	fmt.Println(now)

	tick := time.Tick(5 * time.Second)
	for t := range tick {
		fmt.Println(t)
		break
	}

	fmt.Println("end")
}
