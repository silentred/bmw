package main

import (
	"bmw"
	"bmw/lib"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/spf13/viper"
)

var (
	listenPort string
)

func main() {
	flag.Parse()
	loadConfig("./", "config")

	route := &bmw.HandlerRoute{
		Host:    listenPort,
		Path:    "rsync",
		Topic:   "default",
		Handler: new(H),
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
	listenPort = viper.GetString("listenPort")

	if listenPort == "" {
		listenPort = ":8084"
	}
}

type H struct {
}

type resource struct {
	URL    string `json:"url"`
	Key    string `json:"key"`
	Bucket string `json:"bucket"`
	Region string `json:"region"`
	Field  string `json:"field"`
	Status int8   `json:"status"`
}

type Resources []resource

type responseBody struct {
	Callback  string    `json:"callback"`
	Resources Resources `json:"resources"`
}

type body struct {
	ID        string    `json:"id"`
	Resources Resources `json:"resources"`
}

func (h *H) Handle(job bmw.RetryJob) error {
	var res responseBody
	err := json.Unmarshal(job.Payload, &res)
	if err != nil {
		glog.Error(err)
		return err
	}
	resource := res.Resources
	callbackURL := res.Callback

	var result Resources
	for _, value := range resource {
		value.Status = 1
		fKey := value.Key
		downloadURL := value.URL
		bucket := value.Bucket
		region := value.Region

		sLength := checkObjS3(fKey, bucket, region)
		if sLength != -1 {
			result = append(result, value)
			continue
		}

		downloadConfig := lib.NewReqeustConfig(nil, nil, 200, nil, nil)
		fName, fType, fLength, err := lib.HTTPGetFile(downloadURL, downloadConfig)
		if err != nil {
			glog.Errorf("Download file failed: %s ", err)
			value.Status = 0
			result = append(result, value)
			continue
		}
		glog.Infof("Download file %s type %s length %d", fKey, fType, fLength)
		defer os.Remove(fName)
		fInfo, _ := os.Stat(fName)
		if fInfo.Size() != fLength {
			glog.Error("Download-file's size is not match")
			value.Status = 0
			result = append(result, value)
			continue
		}
		if err = uploadToS3(fName, fKey, bucket, region, fType); err != nil {
			glog.Errorf("Upload file error %s", err)
			value.Status = 0
			result = append(result, value)
			continue
		}
		result = append(result, value)
		glog.Infof("Upload file %s success with length %d", fKey, fLength)
	}

	body := body{
		ID:        job.ID,
		Resources: result,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		glog.Error(err)
		return err
	}
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	callbackConfig := lib.NewReqeustConfig(nil, headers, 10, bodyBytes, nil)
	glog.Infof("request %s body %s", callbackURL, string(bodyBytes))
	_, err = lib.HTTPPost(callbackURL, callbackConfig)
	if err != nil {
		glog.Error(err)
		return err
	}
	return nil
}
