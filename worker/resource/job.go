package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/golang/glog"
)

// message resource bit position
const (
	MsgCoverSuccess = 1 << iota
	MsgBlurSuccess
	MsgWebpSuccess
	MsgURLSuccess
	MsgCSuccess
)

// user resource bit position
const (
	UserAvatarSuccess = 1 << iota
	UserAvatarThumbSuccess
	UserCoverSuccess
)

var uploadErr = fmt.Errorf("upload error")

type createFileFunc func(string) (string, error)
type newKeyFunc func(string) string
type syncResult map[string]int

type jobNode struct {
	bitPosition  int // mask bit
	name         string
	originKey    string
	inFile       string
	outFile      string
	isDone       bool
	isProcessing bool
	resultMap    map[string]bool // fame-public => false, fame-s3 => false
	//errMap       map[string]error
	mapLock    sync.Mutex
	gWait      sync.WaitGroup
	parent     *jobNode
	children   []*jobNode
	createFile createFileFunc // create outFile from inFile
	getNewKey  newKeyFunc     // get new key
}

func (n *jobNode) appendChildren(nodes ...*jobNode) {
	for _, node := range nodes {
		node.parent = n
	}

	n.children = nodes
}

// get upload result; need test
func (n *jobNode) getResult(result *map[string]int) error {
	if n.isProcessing {
		err := fmt.Errorf("%+v is still processing", *n)
		return err
	}
	//fmt.Println("name=", n.name, "result", n.resultMap, "bit=", n.bitPosition)
	for key, value := range n.resultMap {
		if value {
			if _, ok := (*result)[key]; ok {
				(*result)[key] = (*result)[key] | n.bitPosition
			} else {
				(*result)[key] = n.bitPosition
			}
		}
	}

	for _, node := range n.children {
		err := node.getResult(result)
		if err != nil {
			return err
		}
	}

	return nil
}

// need test;
func (n *jobNode) doJob() error {
	var genFileErr error

	if n.isProcessing {
		err := fmt.Errorf("%+v is already processing", *n)
		glog.Error(err)
		genFileErr = err
	}

	// set isProcessing to true
	n.isProcessing = true

	// if inFile is empty, go back to parent, take its outFile
	if len(n.inFile) == 0 && n.parent != nil && len(n.parent.outFile) > 0 {
		n.inFile = n.parent.outFile
	}

	// if inFile is not empty
	if len(n.inFile) > 0 {
		errChan := make(chan error, len(n.resultMap))

		// createFile(inFile), and set outFile
		newFile, err := n.createFile(n.inFile)
		n.outFile = newFile
		if err != nil {
			glog.Error(err)
			genFileErr = err
		}

		// go func(){}(), use gWait to upload to outFileDest, set final
		if genFileErr == nil {
			for destBucket, result := range n.resultMap {
				if !result {
					n.gWait.Add(1)

					go func(destBucket string) {
						defer n.gWait.Done()

						if pool, ok := bucketPool[destBucket]; ok {
							bucket := pool.Get()
							defer pool.Put(bucket)

							if b, ok := bucket.(Bucket); ok {

								var newKey string
								if n.getNewKey == nil {
									newKey = n.originKey
								} else {
									newKey = n.getNewKey(n.originKey)
								}

								// TODO upload file; check if has timeout; handle error
								//fmt.Println("pretend uploading", newFile, "key=", newKey, "bucket=", destBucket)
								err := b.Upload(newKey, newFile)
								if err != nil {
									glog.Error(err)
									errChan <- err
									return
								}

								n.mapLock.Lock() // should use spinlock;
								n.resultMap[destBucket] = true
								n.mapLock.Unlock()

							} else {
								err := fmt.Errorf("%#v is not Bucket", bucket)
								glog.Error(err)
							}
						} else {
							err := fmt.Errorf("bucket %s not definded", destBucket)
							glog.Error(err)
						}
					}(destBucket)
				}
			}

			n.gWait.Wait()

			if len(errChan) > 0 {
				// if upload failed , retry
				return uploadErr
			}
			close(errChan)
		}

	}

	n.isDone = true
	n.isProcessing = false

	// children node doJob
	if n.children != nil && len(n.children) > 0 {
		for _, node := range n.children {
			if err := node.doJob(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (n *jobNode) cleanFiles() {
	if len(n.children) > 0 {
		for _, node := range n.children {
			node.cleanFiles()
		}
	}

	if n.isDone {
		glog.Infof("clean outFile=%s", n.outFile)
		os.Remove(n.outFile)
	} else {
		glog.Errorf("file %s is not ready to remove", n.outFile)
	}
}

// ====== make jobTree ====

func newBaseJobNode(name, originKey string, bitPostion int, destBuckets []string) *jobNode {
	resultMap := make(map[string]bool)
	for _, value := range destBuckets {
		resultMap[value] = false
	}

	return &jobNode{
		name:        name,
		bitPosition: bitPostion,
		originKey:   originKey,
		mapLock:     sync.Mutex{},
		gWait:       sync.WaitGroup{},
		resultMap:   resultMap,
	}
}

// download job
func makeDownloadJob(request *Request) *jobNode {
	var bitPosition int

	switch request.Type {
	case "msg_video", "msg_image", "msg_url_only":
		bitPosition = MsgURLSuccess
	case "msg_c":
		bitPosition = MsgCSuccess
	case "user_avatar":
		bitPosition = UserAvatarSuccess
	case "user_cover":
		bitPosition = UserCoverSuccess
	}

	bucket := request.FromBucket
	key := request.Key
	destBuckets := []string{request.ToBucket}

	pool, err := getBucektPool(bucket)
	if err != nil {
		glog.Error(err)
		return nil
	}

	if b, ok := pool.Get().(Bucket); ok {
		url := b.GetURL(key)

		job := newBaseJobNode(request.Type, key, bitPosition, destBuckets)

		job.inFile = url          // as root node, inFile is nessesory
		job.createFile = download // work function, generates outFile
		job.getNewKey = nil       // get key for upload

		job.resultMap[bucket] = true // set source bucket to be true, means the resource is present in that bucket

		return job
	}

	return nil
}

// ======== All kinds of jobNode =========

// make user avatar thumbnail
func makeUserAvatarThumbJob(request *Request) *jobNode {
	job := newBaseJobNode(request.Type, request.Key, UserAvatarThumbSuccess, request.getBothBuckets())

	job.createFile = thumbnailImage
	job.getNewKey = getUserAvatarThumbKey

	return job
}

// make message image cover
func makeMsgImageCover(request *Request) *jobNode {
	job := newBaseJobNode(request.Type, request.Key, MsgCoverSuccess, request.getBothBuckets())

	job.createFile = thumbnailImage
	job.getNewKey = getMsgCoverKey

	return job
}

// make message cover of video
func makeMsgVideoCover(request *Request) *jobNode {
	job := newBaseJobNode(request.Type, request.Key, MsgCoverSuccess, request.getBothBuckets())

	job.createFile = videoToCover
	job.getNewKey = getMsgCoverKey

	return job
}

// make message blur cover of both image and video
func makeMsgImageBlur(request *Request) *jobNode {
	job := newBaseJobNode(request.Type, request.Key, MsgBlurSuccess, request.getBothBuckets())

	job.createFile = blurImage
	job.getNewKey = getMsgBlurKey

	return job
}

// make message webp cover of video
func makeMsgVideoWebp(request *Request) *jobNode {
	job := newBaseJobNode(request.Type, request.Key, MsgWebpSuccess, request.getBothBuckets())

	job.createFile = videoToWebp
	job.getNewKey = getMsgWebpKey

	return job
}

// ======= generate tree of jobNodes ========

// download->(videoCover->imgBlur, webp)
func makeMsgVideoJobTree(request *Request) *jobNode {
	root := makeDownloadJob(request)

	cover := makeMsgVideoCover(request)
	blur := makeMsgImageBlur(request)
	webp := makeMsgVideoWebp(request)

	cover.appendChildren(blur)

	root.appendChildren(cover, webp)

	return root
}

// download->imgCover->imgBlur
func makeMsgImageJobTree(request *Request) *jobNode {
	root := makeDownloadJob(request)

	cover := makeMsgImageCover(request)
	blur := makeMsgImageBlur(request)

	cover.appendChildren(blur)
	root.appendChildren(cover)

	return root
}

func makeMsgCJobTree(request *Request) *jobNode {
	return makeDownloadJob(request)
}

func makeUserAvatarJobTree(request *Request) *jobNode {
	root := makeDownloadJob(request)

	cover := makeUserAvatarThumbJob(request)
	root.appendChildren(cover)

	return root
}

func makeMsgURLOnlyJobTree(request *Request) *jobNode {
	return makeDownloadJob(request)
}

func makeUserCoverJobTree(request *Request) *jobNode {
	return makeDownloadJob(request)
}

// ======= originKey to newKey =========

func getMsgCoverKey(originKey string) string {
	return fmt.Sprintf("moment/cover/%s", originKey)
}

func getMsgBlurKey(originKey string) string {
	return fmt.Sprintf("%s_blur", getMsgCoverKey(originKey))
}

func getMsgWebpKey(originKey string) string {
	return fmt.Sprintf("%s_webp", getMsgCoverKey(originKey))
}

func getUserAvatarThumbKey(originKey string) string {
	return fmt.Sprintf("%s_thumb", getMsgCoverKey(originKey))
}
