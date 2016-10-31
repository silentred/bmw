package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
	"qiniupkg.com/api.v7/kodo"
)

var (
	QiniuAccessToken = "IZy_H_8V09pFN0d4D_CROnezQ6UaMwqdcJ9PN2rx"
	QiniuSecretToken = "uNq3vv-709geTDx06d4t8g0Hyec8N_HY1rswzTVJ"

	S3Region      = "us-west-1"
	S3Credentials = "./credentials"

	allBuckets = []string{"fame-private", "fame-public", "fame-private-us-west-1", "dev-resource", "fame-dev"}

	bucketDomainMap = map[string]string{
		"fame-private": "f0.nihao.com",
		"fame-public":  "f1.nihao.com",
		"dev-resource": "7xqgm2.com2.z0.glb.qiniucdn.com",

		"fame-private-us-west-1": "dtxwytoppugh2.cloudfront.net",
		"fame-dev":               "s3-us-west-1.amazonaws.com/fame-dev",
	}

	bucketPool = map[string]sync.Pool{
		"fame-private": sync.Pool{
			New: func() interface{} {
				return NewQiniuBucket("fame-private")
			},
		},
		"fame-public": sync.Pool{
			New: func() interface{} {
				return NewQiniuBucket("fame-public")
			},
		},
		"fame-private-us-west-1": sync.Pool{
			New: func() interface{} {
				return NewS3Bucket("fame-private-us-west-1")
			},
		},
		"dev-resource": sync.Pool{
			New: func() interface{} {
				return NewQiniuBucket("dev-resource")
			},
		},
		"fame-dev": sync.Pool{
			New: func() interface{} {
				return NewS3Bucket("fame-dev")
			},
		},
	}
)

func init() {
	kodo.SetMac(QiniuAccessToken, QiniuSecretToken)
}

func getBucektPool(bucket string) (sync.Pool, error) {
	if pool, ok := bucketPool[bucket]; ok {
		return pool, nil
	}

	return sync.Pool{}, fmt.Errorf("no such bucket %s", bucket)
}

// Bucket is the upload manager of storage service
type Bucket interface {
	GetURL(key string) string
	FetchResource(key string) []byte
	Upload(key, file string) error
}

// ======== QiniuBucket ========

type QiniuBucket struct {
	Name   string
	Domain string
	//auth
	bucket kodo.Bucket
}

func NewQiniuBucket(name string) Bucket {
	client := kodo.New(0, nil) // 用默认配置创建 Client
	bucket := client.Bucket(name)
	if domain, ok := bucketDomainMap[name]; ok {
		return QiniuBucket{
			Name:   name,
			Domain: domain,
			bucket: bucket,
		}
	}

	glog.Errorf("domain not found for bucket=%s", name)
	return nil
}

func (b QiniuBucket) exists(key string) bool {
	entry, err := b.bucket.Stat(nil, key)
	if err != nil {
		//glog.Infof("err=%s file %s is not in bucket %s", err, key, b.Name)
		return false
	}

	if entry.Fsize > 0 {
		glog.Infof("file %s is in bucket %s with size=%d", key, b.Name, entry.Fsize)
		return true
	}

	return false
}

func (b QiniuBucket) Upload(key string, file string) error {
	// checkout if exists
	if b.exists(key) {
		return nil
	}

	err := b.bucket.PutFile(nil, nil, key, file, nil)
	if err != nil {
		glog.Errorf("upload to qiniu with error %s", err)
	}

	return err
}

func (b QiniuBucket) GetURL(key string) string {
	return fmt.Sprintf("http://%s/%s", b.Domain, key)
}

// TODO
func (b QiniuBucket) FetchResource(key string) []byte {
	return nil
}

//======== S3Bucket ========

type S3Bucket struct {
	Name     string
	Domain   string
	uploader *s3manager.Uploader
	service  *s3.S3
}

func NewS3Bucket(name string) Bucket {
	creds := credentials.NewSharedCredentials(S3Credentials, "default")
	uploader := s3manager.NewUploader(session.New(&aws.Config{
		Region:      aws.String(S3Region),
		Credentials: creds,
	}))

	service := s3.New(session.New(&aws.Config{
		Region:      aws.String(S3Region),
		Credentials: creds,
	}))

	if domain, ok := bucketDomainMap[name]; ok {
		return S3Bucket{
			Name:     name,
			Domain:   domain,
			uploader: uploader,
			service:  service,
		}
	}

	glog.Errorf("domain not found for bucket=%s", name)
	return nil
}

func (b S3Bucket) exists(key string) bool {
	params := &s3.HeadObjectInput{
		Bucket: aws.String(b.Name),
		Key:    aws.String(key),
	}
	resp, err := b.service.HeadObject(params)
	if err != nil {
		//glog.Infof("file %s is not in bucket %s", key, b.Name)
		return false
	}

	glog.Infof("file %s is in bucket=%s with size=%d", key, b.Name, *resp.ContentLength)
	return true
}

func (b S3Bucket) Upload(key string, file string) error {
	// checkout if exists
	if b.exists(key) {
		return nil
	}

	fd, err := os.Open(file)
	defer fd.Close()

	if err != nil {
		glog.Errorf("Opening file failed: %s", err)
		return err
	}

	result, err := b.uploader.Upload(&s3manager.UploadInput{
		Body:   fd,
		Bucket: aws.String(b.Name),
		Key:    aws.String(key),
		ACL:    aws.String("public-read"),
	})

	if err != nil {
		glog.Errorf("Failed to upload file %s error: %s", key, err)
		return err
	}

	glog.Infof("Successfully upload file %s", result.Location)
	return nil
}

func (b S3Bucket) GetURL(key string) string {
	return fmt.Sprintf("http://%s/%s", b.Domain, key)
}

// TODO
func (b S3Bucket) FetchResource(key string) []byte {
	return nil
}
