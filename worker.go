package bmw

import (
	"bmw/lib"
	"errors"

	"github.com/golang/glog"
)

const (
	minConcurrency = 10
)

var (
	MaxRetry   = uint8(3)
	RetryError = errors.New("need retry")
)

// A buffered channel that we can send work requests on.
//var JobQueue chan RetryJob

// Worker represents the worker that executes the job
type Worker struct {
	ID         string
	WorkerPool chan chan RetryJob
	JobChannel chan RetryJob
	quit       chan bool
	handler    Handler
	rateLimit  uint64
	async      bool
	limiter    *lib.RateLimiter
}

func NewWorker(ID string, workerPool chan chan RetryJob, handler Handler, concurrency uint64, async bool) Worker {
	if concurrency < minConcurrency {
		concurrency = minConcurrency
	}

	limiter := lib.NewRateLimiter(concurrency)

	return Worker{
		ID:         ID,
		WorkerPool: workerPool,
		JobChannel: make(chan RetryJob),
		quit:       make(chan bool),
		handler:    handler,
		rateLimit:  concurrency,
		async:      async,
		limiter:    limiter,
	}
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w *Worker) Start() {
	go func() {
		for {
			// register the current worker into the worker queue.
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				// we have received a work request.
				if w.async {
					if w.limiter.Limit() {
						go w.handle(job)
					}
				} else {
					w.handle(job)
				}

			case <-w.quit:
				// we have received a signal to stop
				glog.Error("worker is quitting")
				return
			}
		}
	}()
}

func (w *Worker) handle(job RetryJob) {
	glog.Infof("worker %s is handling, jobID=%s&retry=%d", w.ID, job.ID, job.Retry)

	if job.Retry < MaxRetry {
		if err := w.handler.Handle(job); err != nil {
			glog.Error(err)
			if err == RetryError {
				job.Retry++
				jobChannel := <-w.WorkerPool
				jobChannel <- job
			}
		}
	} else {
		glog.Warningf("reach MaxRetry, Job.ID=%s", job.ID)
	}
}

// Stop signals the worker to stop listening for work requests.
func (w *Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}
