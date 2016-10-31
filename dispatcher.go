package bmw

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/golang/glog"
)

var (
	defaultDispatcherConfig *DispatcherConfig
)

func init() {
	cpuNum := runtime.NumCPU()
	defaultDispatcherConfig = &DispatcherConfig{
		MaxWorkers: cpuNum,
		WorkerRate: 10,
	}

}

// Dispatcher manages the workers
type Dispatcher struct {
	// A pool of workers channels that are registered with the dispatcher
	WorkerPool chan chan RetryJob
	Config     *DispatcherConfig
	handler    Handler
	waiter     Waiter
}

type DispatcherConfig struct {
	MaxWorkers int
	WorkerRate int
}

// NewDispatcher makes a dispatcher
func NewDispatcher(handler Handler, waiter Waiter, config *DispatcherConfig) *Dispatcher {
	if config == nil {
		config = defaultDispatcherConfig
	}

	pool := make(chan chan RetryJob, config.MaxWorkers)
	return &Dispatcher{
		WorkerPool: pool,
		Config:     config,
		handler:    handler,
		waiter:     waiter,
	}
}

// Run generates workers, make them start working
func (d *Dispatcher) Run() {
	// starting n number of workers
	for i := 0; i < d.Config.MaxWorkers; i++ {
		worker := NewWorker(fmt.Sprintf("NO_%d", i), d.WorkerPool, d.handler, uint64(d.Config.WorkerRate), true)
		worker.Start()
	}

	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {
		jobByte := d.waiter.Wait()
		var job RetryJob
		if err := json.Unmarshal(jobByte, &job); err != nil {
			glog.Error(err)
		}

		// a job request has been received
		//go func(job RetryJob) {
		// try to obtain a worker job channel that is available.
		// this will block until a worker is idle
		jobChannel := <-d.WorkerPool

		// dispatch the job to the worker job channel
		jobChannel <- job
		//}(job)
	}
}
