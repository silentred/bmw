package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	pipe "gopkg.in/pipe.v2"
)

type Cmd struct {
	cmd    string
	saveas string
	ops    []Op
	err    error
}

// resize, blur
func (cmd *Cmd) parse() []Op {
	var ops []Op
	strCmds := strings.Split(cmd.cmd, "|")
	for _, item := range strCmds {
		sourceRegex := regexp.MustCompile(`file/(.+)`)
		reziseRegex := regexp.MustCompile(`thumbnail/(\d*)x(\d*)`)
		blurRegex := regexp.MustCompile(`blur/(\d+)x(\d+)`)

		matches := sourceRegex.FindStringSubmatch(item)
		if len(matches) > 0 {
			filename := matches[1]
			fileOp := ReadFileOp{filename}
			ops = append(ops, fileOp)
		}

		matches = nil
		matches = reziseRegex.FindStringSubmatch(item)
		if len(matches) > 0 {
			width, _ := strconv.Atoi(matches[1])
			height, _ := strconv.Atoi(matches[2])
			resizeOp := ResizeOp{width, height}
			ops = append(ops, resizeOp)
		}

		matches = nil
		matches = blurRegex.FindStringSubmatch(item)
		if len(matches) > 0 {
			a, _ := strconv.Atoi(matches[1])
			b, _ := strconv.Atoi(matches[2])
			if a == 0 {
				a = 1
			}

			sigma := float64(b) / float64(a)
			blurOp := BlurOp{sigma}
			ops = append(ops, blurOp)
		}

	}

	cmd.ops = ops

	return ops
}

func (cmd *Cmd) doOps() error {
	var pipes []pipe.Pipe
	for _, item := range cmd.ops {
		pipes = append(pipes, item.getPipe())
	}

	p := pipe.Line(pipes...)
	output, err := pipe.CombinedOutput(p)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	buf := bytes.NewBuffer(output)
	img, _ := imaging.Decode(buf)

	imaging.Save(img, cmd.saveas)
	return nil
}

type Op interface {
	getPipe() pipe.Pipe
}

type ReadFileOp struct {
	file string
}

func (c ReadFileOp) getPipe() pipe.Pipe {
	return pipe.ReadFile(c.file)
}

type BlurOp struct {
	sigma float64
}

func (c BlurOp) getPipe() pipe.Pipe {
	return blur(c.sigma)
}

type ResizeOp struct {
	width, height int
}

func (c ResizeOp) getPipe() pipe.Pipe {
	return resize(c.width, c.height)
}

func blur(sigma float64) pipe.Pipe {
	return pipe.TaskFunc(func(state *pipe.State) error {
		img, err := imaging.Decode(state.Stdin)
		if err != nil {
			fmt.Println(err)
			return err
		}

		res := imaging.Blur(img, sigma)
		imaging.Encode(state.Stdout, res, imaging.PNG)

		return nil
	})
}

func resize(width, height int) pipe.Pipe {
	return pipe.TaskFunc(func(state *pipe.State) error {
		img, err := imaging.Decode(state.Stdin)
		if err != nil {
			fmt.Println(err)
			return err
		}

		res := imaging.Resize(img, width, height, imaging.Lanczos)

		imaging.Encode(state.Stdout, res, imaging.PNG)
		return nil
	})
}

type BenchCmd struct {
	cmds      []Cmd
	waitGroup sync.WaitGroup
	errs      []error
	lock      sync.Mutex
}

func (b *BenchCmd) doCmds() {
	for _, item := range b.cmds {
		b.waitGroup.Add(1)

		go func(cmd Cmd) {
			cmd.parse()
			err := cmd.doOps()

			b.lock.Lock()
			b.errs = append(b.errs, err)
			b.lock.Unlock()

			b.waitGroup.Done()
		}(item)
	}

	b.waitGroup.Wait()
}

func remainOdd(x []int) []int {
	y := x[:0]
	for _, n := range x {
		if n%2 != 0 {
			y = append(y, n)
		}
	}
	return y
}
