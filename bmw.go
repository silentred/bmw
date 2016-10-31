package bmw

import "net/http"

// Pusher push raw data from API request to queue
type Pusher interface {
	Push([]byte) error
}

// Waiter waits for job from queue
type Waiter interface {
	Wait() []byte
}

// PusherWaiter is both pusher and waiter
type PusherWaiter interface {
	Pusher
	Waiter
}

// Serializer serialize the job, therefore which can be pushed into queue
type Serializer interface {
	Serialize(interface{}) []byte
	Unserialize([]byte) interface{}
}

// Handler handles payloadJob
type Handler interface {
	Handle(job RetryJob) error
}

// Parser parse request to PayloadJob
type Parser interface {
	Parse(*http.Request) interface{}
}

// one route path needs { PayloadJob serializer;   }

// order：
// 1. API receives a request. [[[ Parse the query, transform to PayloadJob (not same for every route)
// 2. serialize PayloadJob to []byte. ]]] Make a RetryJob, serialize it, push to queue

// 3. Consumer waits for job bytes from queue; Unserialize to RetryJob
// 4. [[[ Unserialize PayloadJob. ]]] Handle the job.

// new order:
// 1. API receives request body as Payload, make a RetryJob, serialize it, push to queue;
// 2. Dispatcher waits data from queue, push to each worker. Worker handles job, decide if retry is needed
