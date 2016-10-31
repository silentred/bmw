package main

import (
	"bmw"
	"errors"
	"fmt"
	"os"
	"sync"

	apns "github.com/anachronistic/apns"
	"github.com/golang/glog"
)

var (
	certFiles = map[string]string{
		"com.being.fame":         "fame_aps_production.pem",
		"com.being.fame.inhouse": "fame_inhouse_aps_production.pem",
	}
	certDir string = "./"
)

type APNs struct {
	pushPool *sync.Pool
}

func NewAPNs() *APNs {
	apns := &APNs{
		pushPool: &sync.Pool{
			New: func() interface{} {
				return apns.NewPushNotification()
			},
		},
	}

	return apns
}

func (a *APNs) send(job *PushPayload) error {
	if job.JobType != TypeAPNsJob {
		return fmt.Errorf("invalid job type. not APNsJob but %d", job.JobType)
	}

	payload := apns.NewPayload()
	payload.Alert = job.Title
	payload.Badge = job.Badge

	pn, ok := a.pushPool.Get().(*apns.PushNotification)
	defer a.pushPool.Put(pn)

	if ok {
		pn.DeviceToken = job.Addressee
		pn.AddPayload(payload)
	} else {
		err := errors.New("object from pushPool is not PushNotification")
		glog.Error(err)
		return err
	}

	if fileName, ok := certFiles[job.PushService]; ok {
		certFile := certDir + fileName
		if _, err := os.Stat(certFile); err != nil {
			glog.Error(err)
			return err
		}

		client := apns.NewClient("gateway.push.apple.com:2195", certFile, certFile)
		resp := client.Send(pn)

		alert, _ := pn.PayloadString()

		glog.Infof("APNs result: %t, %s", resp.Success, alert)
		if resp.Error != nil {
			glog.Error(resp.Error)
			return bmw.RetryError
		}
	} else {
		err := fmt.Errorf("no config for key %s", job.PushService)
		return err
	}

	return nil
}
