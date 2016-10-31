package mqtt

import (
	"os"
	"io/ioutil"
	"github.com/BurntSushi/toml"
	"github.com/golang/glog"
	"bmw/producer"
)

type Config struct {
	Api *MqttApiConfig
	Mqtt *mqttConfig
	Beanstalk *producer.BeanstalkConfig
}

// NewConfig new a config.
func NewConfig(conf string) (c *Config, err error) {
	var (
		file *os.File
		blob []byte
	)

	c = new(Config)
	if file, err = os.Open(conf); err != nil {
		return
	}

	if blob, err = ioutil.ReadAll(file); err != nil {
		return
	}

	if err = toml.Unmarshal(blob, c); err != nil {
		glog.Error(err)
	}

	return c , err
}

