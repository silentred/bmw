package mqtt

import (
	"bmw"
	bp "bmw/producer"
	"encoding/json"

	"github.com/golang/glog"
)

// MqttWorker represents a worker which receives http request to send messages to mqtt broker
type MqttWorker struct {
	bmw.DefaultWorker
	Config *Config
}

// NewMqttWorker makes a new MqttWorker
func NewMqttWorker(config *Config) *MqttWorker {
	var (
		producer bmw.Producer
		pusher   bmw.Pusher
		consumer bmw.Consumer
		api      bmw.API
		jobChan  = make(chan interface{}, 1000)
	)

	beanstalkProducerPusher := bp.NewBeanstalkProducerPusher(config.Beanstalk, jobChan, transformToMqttJob)
	producer = beanstalkProducerPusher
	pusher = beanstalkProducerPusher
	consumer = NewMqttJobConsumer(jobChan, config.Mqtt)
	api = NewMqttApi(config.Api, pusher)

	worker := &MqttWorker{
		DefaultWorker: bmw.DefaultWorker{
			Producer: producer,
			Consumer: consumer,
			Api:      api,
			JobChan:  jobChan,
		},
		Config: config,
	}

	return worker
}

func transformToMqttJob(data []byte) (interface{}, error) {
	var err error
	job := new(MqttJob)

	if err = json.Unmarshal(data, job); err != nil {
		glog.Error("failed to convert data to MqttJob", err)
	}

	return job, err
}
