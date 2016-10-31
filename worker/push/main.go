package main

import (
	"bmw"
	"flag"
	"fmt"
	"net/url"
	"strconv"

	"github.com/golang/glog"
	"github.com/spf13/viper"
	"gopkg.in/fsnotify.v1"
)

const TypeAPNsJob = 1
const TypeGCMJob = 2

var (
	listenPort string
)

func main() {
	flag.Parse()
	loadConfig("./", "config")

	handler := &H{
		apns: NewAPNs(),
		gcm:  new(GCM),
	}

	route := &bmw.HandlerRoute{
		Host:    listenPort,
		Path:    "push",
		Topic:   "default",
		Handler: handler,
	}
	//beanstalk := bmw.BeanstalkConfig{Host: "localhost:11300"}
	goChan := bmw.GoChannelConfig{}
	bmw := bmw.NewSingleBMW(goChan, route)
	bmw.Start()
}

func loadConfig(path, name string) {
	viper.SetConfigName(name)
	viper.AddConfigPath(path)
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(event fsnotify.Event) {
		glog.Info(event.String())
		setVar()
	})

	setVar()
}

func setVar() {
	certFiles = viper.GetStringMapString("apns")
	certDir = viper.GetString("apns_cert_dir")
	gcmKeys = viper.GetStringMapString("gcm_keys")
	listenPort = viper.GetString("listenPort")

	if listenPort == "" {
		listenPort = ":8082"
	}
}

type H struct {
	apns *APNs
	gcm  *GCM
}

type PushPayload struct {
	ID        string `json:"id"`
	JobType   int    `json:"job_type"`
	Addressee string `json:"addressee"`
	Title     string `json:"title"`
	Badge     int    `json:"badge"`
	Sound     string `json:"sound"`
	// for APNs: "com.being.fame" or "com.being.fame.inhouse"
	// for andorid: "GCM"
	PushService string `json:"push_service"`
}

func (h *H) Handle(job bmw.RetryJob) error {
	values, err := url.ParseQuery(string(job.Payload))
	if err != nil {
		glog.Error(err)
		return err
	}

	payload, err := bindValue(values, job.ID)
	glog.Infof("jobID: %s payload: %+v", job.ID, payload)

	switch payload.JobType {
	case TypeAPNsJob:
		err = h.apns.send(payload)
	case TypeGCMJob:
		err = h.gcm.send(payload)
	default:
		glog.Errorf("job_type %d is invalid", payload.JobType)
	}

	return err
}

func bindValue(values url.Values, id string) (*PushPayload, error) {
	jobType, err := strconv.Atoi(values.Get("job_type"))
	if err != nil {
		glog.Error(err, "failed to convert job_type to Int by Atoi()")
		return nil, err
	}

	var badge int
	badge, _ = strconv.Atoi(values.Get("badge"))

	addressee := values.Get("addressee")
	title := values.Get("title")
	sound := values.Get("sound")
	service := values.Get("push_service")

	payload := &PushPayload{
		ID:          id,
		JobType:     jobType,
		Addressee:   addressee,
		Title:       title,
		Badge:       badge,
		Sound:       sound,
		PushService: service,
	}

	return payload, nil
}
