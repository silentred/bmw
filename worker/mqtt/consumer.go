package mqtt

import (
	"github.com/golang/glog"
	"errors"
	Pool "github.com/jolestar/go-commons-pool"
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
)

type MqttJob struct {
	Topic    string `json:"topic"`
	Body     string `json:"body"`
	Qos      byte `json:"qos"`
	Retained bool `json:"retained"`
}

var mqttJobError = errors.New("job type is not MqttJob")

// get job from go-chan, handle it
type MqttJobConsumer struct {
	jobChan chan interface{}
	mqttPool *Pool.ObjectPool
}

type mqttConfig struct {
	Host, Username, Password string
}

func NewMqttJobConsumer(jobChan chan interface{}, config *mqttConfig) *MqttJobConsumer {
	pool := newPool(config.Host, config.Username, config.Password)
	return &MqttJobConsumer{
		jobChan: jobChan,
		mqttPool: pool,
	}
}

func (mjc *MqttJobConsumer) GetJob() interface{} {
	job := <- mjc.jobChan
	return job
}

func (mjc *MqttJobConsumer) Handle(job interface{}) error {
	if mqttJob, ok := job.(MqttJob); ok {
		obj, err := mjc.mqttPool.BorrowObject()
		if err != nil {
			glog.Error(err)
		}
		if client, ok := obj.(*MQTT.Client); ok {
			token := client.Publish(mqttJob.Topic, mqttJob.Qos, mqttJob.Retained, mqttJob.Body)
			token.WaitTimeout(waitTimeout)
		} else {
			glog.Error("Borrowed object in mqttPool is not *MQTT.Client")
		}
		mjc.mqttPool.ReturnObject(obj)

		return nil
	}

	return mqttJobError
}

func (mjc *MqttJobConsumer) Start() {
	for {
		job := mjc.GetJob()
		mjc.Handle(job)
	}
}

