package queue

// transform queue data to job
type Transform func([]byte) (interface{}, error)
