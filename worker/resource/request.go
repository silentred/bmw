package main

import (
	"bmw"
	"bmw/lib"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

// Request is the input of the job
type Request struct {
	Key         string `json:"key"`
	FromBucket  string `json:"from_bucket"`
	Type        string `json:"type"`
	ToBucket    string `json:"to_bucket"`
	CallbackURL string `json:"callback"`
	jobTree     *jobNode
}

// CallbackBody is post body to the callbackURL
type CallbackBody struct {
	Req    *Request       `json:"request"`
	Result map[string]int `json:"result"`
}

// BatchRequest contains many requests at once
type BatchRequest struct {
	Requests    []Request `json:"requests"`
	CallbackURL string    `json:"callback"`
}

func (req *Request) getBothBuckets() []string {
	return []string{req.FromBucket, req.ToBucket}
}

func (req *Request) makeJobs() error {
	switch req.Type {
	case "msg_video":
		req.jobTree = makeMsgVideoJobTree(req)
	case "msg_image":
		req.jobTree = makeMsgImageJobTree(req)
	case "msg_url_only":
		req.jobTree = makeMsgURLOnlyJobTree(req)
	case "msg_c":
		req.jobTree = makeMsgCJobTree(req)
	case "user_avatar":
		req.jobTree = makeUserAvatarJobTree(req)
	case "user_cover":
		req.jobTree = makeUserCoverJobTree(req)
	default:
		return fmt.Errorf("no such type %s", req.Type)
	}

	return nil
}

func (req *Request) doJobs() (syncResult, error) {
	if req.jobTree != nil {
		err := req.jobTree.doJob()
		if err != nil {
			glog.Error(err)
			return nil, err
		}

		result := make(map[string]int)
		err = req.jobTree.getResult(&result)
		if err != nil {
			glog.Error(err)
			return nil, err
		}

		return result, nil
	}

	return nil, fmt.Errorf("doJobs before making jobTree")
}

// handle the upload query
func handleRequest(job *bmw.RetryJob) error {
	req := new(Request)
	err := json.Unmarshal(job.Payload, req)
	if err != nil {
		glog.Error(err)
		return err
	}

	err = req.makeJobs()
	if err != nil {
		glog.Error(err)
		return err
	}

	result, err := req.doJobs()
	if err != nil {
		glog.Error(err)
		if err == uploadErr {
			return bmw.RetryError
		}
	}

	// save result
	store(req.Key, result)

	// callback
	callback := CallbackBody{req, result}
	body, err := json.Marshal(callback)
	if err != nil {
		glog.Error(err)
		return err
	}
	config := lib.NewReqeustConfig(nil, nil, 30, body, nil)
	lib.HTTPPost(req.CallbackURL, config)

	return nil
}

func handleBatchRequest(job *bmw.RetryJob) error {
	var batchCallback []CallbackBody

	req := new(BatchRequest)
	err := json.Unmarshal(job.Payload, req)
	if err != nil {
		glog.Error(err)
		return err
	}

	for _, item := range req.Requests {
		err = item.makeJobs()
		if err != nil {
			glog.Error(err)
			return err
		}

		result, err := item.doJobs()
		if err != nil {
			glog.Error(err)
			if err == uploadErr {
				return bmw.RetryError
			}
		}

		// save result
		store(item.Key, result)

		// callback
		callback := CallbackBody{&item, result}
		batchCallback = append(batchCallback, callback)
	}

	body, err := json.Marshal(batchCallback)
	if err != nil {
		glog.Error(err)
		return err
	}
	config := lib.NewReqeustConfig(nil, nil, 30, body, nil)
	lib.HTTPPost(req.CallbackURL, config)

	return nil
}

// handle the key query
func findKeyQuery(cxt *gin.Context) {
	key, ok := cxt.GetQuery("key")
	if !ok {
		cxt.JSON(200, gin.H{"error": "missing key"})
		return
	}

	result := findKey(key)
	if result != nil {
		cxt.JSON(200, gin.H{"result": result})
		return
	}

	cxt.JSON(200, gin.H{"error": "key not found"})
}
