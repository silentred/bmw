package main

import (
	"bmw"
	"bmw/lib"
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	proto "github.com/golang/protobuf/proto"
)

type handler struct {
}

func (h *handler) Handle(job bmw.RetryJob) error {
	time.Sleep(2 * time.Second)
	fmt.Println(job.ID)
	return nil
}

type TreeNode struct {
	ID       int
	Name     string
	Children []*TreeNode
}

func newNode(id int, name string) *TreeNode {
	return &TreeNode{
		ID:       id,
		Name:     name,
		Children: make([]*TreeNode, 0),
	}
}

func (t *TreeNode) appendChildren(children ...*TreeNode) {
	t.Children = append(t.Children, children...)
}

func main() {
	fmt.Println(os.TempDir())

	logger := log.New(os.Stdout, "[test]", log.LstdFlags|log.Lshortfile)
	logger.Println("test logger")

	var x = []int{90, 15, 81, 87, 47, 59, 81, 18, 25, 40, 56, 8}
	fmt.Println(x)

	nodeA := newNode(1, "node 1")
	nodeB := newNode(2, "node 2")
	nodeA.appendChildren(nodeB)
	fmt.Printf("p: %p \n", &nodeB)

	var network bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&network) // Will write to network.
	dec := gob.NewDecoder(&network) // Will read from network.

	err := enc.Encode(nodeA)
	if err != nil {
		log.Fatal(err)
	}

	nodeA = nil
	nodeB = nil
	time.Sleep(3 * time.Second)

	node := TreeNode{}
	dec.Decode(&node)
	fmt.Println(node)
	fmt.Println(len(node.Children))
	fmt.Printf("%p, %s", node.Children[0], node.Children[0].Name)

}

func testworker() {
	// cfg := &profile.Config{
	// 	CPUProfile:  true,
	// 	MemProfile:  true,
	// 	ProfilePath: ".",
	// }

	// defer profile.Start(cfg).Stop()

	// flag.Parse()

	// handler := &handler{}

	// routes := []bmw.SingleRoute{
	// 	bmw.SingleRoute{
	// 		Path:    "test",
	// 		Topic:   "test",
	// 		Handler: handler,
	// 	},
	// }
	// handlerRoute := new(bmw.MultiHandlerRoutes)
	// handlerRoute.Host = ":8086"
	// handlerRoute.Routes = routes

	// goChan := bmw.GoChannelConfig{}
	// bmw := bmw.NewMultiBMW(goChan, handlerRoute)

	// bmw.Start()
}

func protoTest() {
	test := &lib.Test{
		Label: "hello",
		Type:  17,
		Reps:  []int64{1, 2, 3},
	}
	fmt.Println(test)

	data, err := proto.Marshal(test)
	if err != nil {
		log.Fatal("marshaling error: ", err)
	}

	newTest := &lib.Test{}
	err = proto.Unmarshal(data, newTest)
	if err != nil {
		log.Fatal("unmarshaling error: ", err)
	}
	fmt.Println(newTest)

	// Now test and newTest contain the same data.
	fmt.Println(test)
}

func spinlock() {
	lock := lib.SpinLock{}

	for i := 0; i < 10; i++ {
		go func(i int) {
			lock.Lock()
			fmt.Println("inside", i)
			lock.Unlock()
		}(i)
	}

	lock.Lock()
	runtime.Gosched()
	lock.Unlock()
}

func gomain() {
	cmds := []string{}
	for i := 0; i < 10000; i++ {
		cmds = append(cmds, fmt.Sprintf("cmd-%d", i))
	}

	results := handleCmds(cmds)

	fmt.Println(len(results))
}

func doCmd(cmd string) string {
	return fmt.Sprintf("cmd=%s", cmd)
}

func handleCmds(cmds []string) (results []string) {
	fmt.Println(len(cmds))
	var count uint64

	group := sync.WaitGroup{}
	lock := sync.Mutex{}
	//group.Add(len(cmds))
	for _, item := range cmds {
		group.Add(1)
		go func(cmd string) {
			result := doCmd(cmd)
			atomic.AddUint64(&count, 1)

			lock.Lock()
			results = append(results, result)
			lock.Unlock()

			group.Done()
		}(item)
	}

	group.Wait()

	fmt.Printf("count=%d \n", count)
	return
}
