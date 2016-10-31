package main

import (
	"bmw/lib"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
)

var gcmKeys map[string]string = map[string]string{
	"com.being.fame": "AIzaSyA2Qt0938WbcyZQn7tXT-5jaItD8GZgQvs",
}
var gcmPushURL = "https://gcm-http.googleapis.com/gcm/send"

// GCM contains api of pushing data
type GCM struct {
}

type body struct {
	To   string      `json:"to"`
	Data interface{} `json:"data"`
}

type bodyData struct {
	Title   string      `json:"title"`
	Message dataMessage `json:"message"`
}

type dataMessage struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// NewGcm returns a GCM instance
func NewGcm() *GCM {
	return &GCM{}
}

func (g *GCM) send(job *PushPayload) error {
	if job.JobType != TypeGCMJob {
		return fmt.Errorf("invalid job type. not GcmJob but %d", job.JobType)
	}

	var apiKey string
	var ok bool
	if apiKey, ok = gcmKeys[job.PushService]; !ok {
		glog.Errorf("key %s not exist", job.PushService)
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "key=" + apiKey,
	}

	dataMsg := dataMessage{
		ID:      job.ID,
		Content: job.Title,
	}
	data := bodyData{
		Title:   job.Title,
		Message: dataMsg,
	}

	body := body{
		To:   job.Addressee,
		Data: data,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		glog.Error(err)
		return err
	}
	glog.Infof("GCM body is %s", bodyBytes)

	config := &lib.RequestConfig{
		Headers: headers,
		Body:    bodyBytes,
		Timeout: 15,
	}

	ret, err := lib.HTTPPost(gcmPushURL, config)
	glog.Info(string(ret))

	return err
}
