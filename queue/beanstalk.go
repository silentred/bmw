package queue

import (
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/silentred/gobeanstalk"
)

type BeanstalkConfig struct {
	Host string
	Tube string
}

type BeanstalkProducerPusher struct {
	config      *BeanstalkConfig
	conn        *gobeanstalk.Conn
	pusherConn  *gobeanstalk.Conn
	pusherMutex *sync.Mutex
	jobs        chan interface{}
}

// NewBeanstalkProducerPusher new a Producer/Push of beanstalk
func NewBeanstalkProducerPusher(config *BeanstalkConfig) *BeanstalkProducerPusher {
	conn := newBeanstalkConn(config)
	pusherConn := newBeanstalkConn(config)
	producer := &BeanstalkProducerPusher{
		config:      config,
		conn:        conn,
		pusherConn:  pusherConn,
		pusherMutex: new(sync.Mutex),
	}

	if _, err := conn.Watch(config.Tube); err != nil {
		glog.Error(err, "beanstalkd watch failed")
	}

	if err := pusherConn.Use(config.Tube); err != nil {
		glog.Error(err)
	}

	return producer
}

func newBeanstalkConn(config *BeanstalkConfig) *gobeanstalk.Conn {
	conn, err := gobeanstalk.Dial(config.Host)
	if err != nil {
		glog.Error(err, "Connecting beanstalk failed. Sleep for 2 sec then make a new connection")
		time.Sleep(2 * time.Second)
		return newBeanstalkConn(config)
	}

	return conn
}

// Wait data from queue
func (bp *BeanstalkProducerPusher) Wait() []byte {
	// try to renew the connection, if the connection is broken
	var b []byte

	defer func() {
		if err := recover(); err != nil {
			glog.Error(err)
			b = nil
		}
	}()

	j, err := bp.conn.Reserve()
	//glog.Infof("get a job from beanstalk %s", string(j.Body))

	if err != nil {
		// send "Need new conn" msg to Manager channel
		glog.Error("gobeanstalk.Conn Reserve() with Error: ", err, ". Trying to NewConnection")
		if err.Error() == "EOF" {
			bp.conn = newBeanstalkConn(bp.config)
			panic("EOF")
		}
	}

	// TODO delete job after handle correctly
	if err = bp.conn.Delete(j.ID); err != nil {
		glog.Error(err, "delete job failed")
	}

	b = j.Body

	return b
}

// Push is Pusher interface. push data from api request into queue.
func (bp *BeanstalkProducerPusher) Push(data []byte) error {
	bp.pusherMutex.Lock()
	// TODO if conn is broken? err == "EOF

	_, err := bp.pusherConn.Put(data, 0, 0, 30*time.Second)
	if err != nil {
		glog.Error(err)
	}

	bp.pusherMutex.Unlock()
	return err
}
