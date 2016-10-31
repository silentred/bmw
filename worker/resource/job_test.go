package main

import (
	"fmt"
	"testing"
)

func TestJobNode(t *testing.T) {
	originKey := "user_cover_key"
	req := &Request{
		FromBucket: "fame-public",
		ToBucket:   "fame-private-us-west-1",
		Key:        originKey,
		Type:       "user_cover",
	}

	job := makeDownloadJob(req)
	fmt.Println(job.resultMap)

	job.doJob()
	fmt.Println(job.resultMap)

	result := make(map[string]int)
	job.getResult(&result)

	fmt.Println(result)
}

func TestJobTree(t *testing.T) {
	// user avatar
	originKey := "user_avatar_key"
	req := &Request{
		FromBucket: "fame-public",
		ToBucket:   "fame-private-us-west-1",
		Key:        originKey,
		Type:       "user_avatar",
	}
	download := makeDownloadJob(req)
	thumbnail := makeUserAvatarThumbJob(req)
	download.appendChildren(thumbnail)

	err := download.doJob()
	fmt.Println("err is", err)

	result := make(map[string]int)
	download.getResult(&result)

	fmt.Println(result)
}

func BenchmarkJobTree(b *testing.B) {
	for i := 0; i < b.N; i++ {
		originKey := "user_avatar_key"
		req := &Request{
			FromBucket: "fame-public",
			ToBucket:   "fame-private-us-west-1",
			Key:        originKey,
			Type:       "user_avatar",
		}
		download := makeDownloadJob(req)
		thumbnail := makeUserAvatarThumbJob(req)
		download.appendChildren(thumbnail)
		download.doJob()
	}
}

func TestVideoJobTree(t *testing.T) {
	originKey := "msg_video_key"
	req := &Request{
		FromBucket: "fame-private",
		ToBucket:   "fame-private-us-west-1",
		Key:        originKey,
		Type:       "msg_video",
	}

	job := makeMsgVideoJobTree(req)

	//err := job.doJob()
	//fmt.Println(err)

	result := make(map[string]int)
	err := job.getResult(&result)
	fmt.Println(err)
	fmt.Println(result)

	job.cleanFiles()
}
