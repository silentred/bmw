package main

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"github.com/disintegration/imaging"
	"gopkg.in/pipe.v2"
)

func TestImage(t *testing.T) {
	p := pipe.Line(
		pipe.ReadFile("test.png"),
		resize(300, 300),
		blur(0.5),
	)

	output, err := pipe.CombinedOutput(p)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	buf := bytes.NewBuffer(output)
	img, _ := imaging.Decode(buf)

	imaging.Save(img, "test_a.png")
}

func TestCmd(t *testing.T) {
	cmdStr := `file/test.png|thumbnail/x300|blur/20x8`
	cmd := Cmd{cmdStr, "test_b.png", nil, nil}

	cmd.parse()
	cmd.doOps()
}

func TestBenchCmd(t *testing.T) {
	var cmds []Cmd
	cmd_a := Cmd{`file/test.png|thumbnail/x300|blur/20x8`, "test_a.png", nil, nil}
	cmd_b := Cmd{`file/test.png|thumbnail/500x1000|blur/20x108`, "test_b.png", nil, nil}
	cmd_c := Cmd{`file/test.png|thumbnail/300x300`, "test_c.png", nil, nil}

	cmds = append(cmds, cmd_a)
	cmds = append(cmds, cmd_b)
	cmds = append(cmds, cmd_c)

	bench := BenchCmd{
		cmds:      cmds,
		waitGroup: sync.WaitGroup{},
		lock:      sync.Mutex{},
	}

	bench.doCmds()

	fmt.Println(bench.errs)
}

func BenchmarkOdd(b *testing.B) {
	var x = []int{90, 15, 81, 87, 47, 59, 81, 18, 25, 40, 56, 8}
	for i := 0; i < b.N; i++ {
		remainOdd(x)
	}
}
