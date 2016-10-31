package main

import (
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
)

func uploadToS3(fName, key, bucket, region, contentType string) error {
	tmpFp, err := os.Open(fName)
	if err != nil {
		return err
	}
	reader, writer := io.Pipe()
	go func() {
		io.Copy(writer, tmpFp)
		tmpFp.Close()
		writer.Close()
	}()
	creds := credentials.NewSharedCredentials("./credentials", "default")
	uploader := s3manager.NewUploader(session.New(&aws.Config{
		Region:      aws.String(region),
		Credentials: creds,
	}))
	_, err = uploader.Upload(&s3manager.UploadInput{
		Body:        reader,
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ACL:         aws.String("public-read"),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return err
	}
	_ = checkObjS3(key, bucket, region)
	return nil
}

func checkObjS3(key, bucket, region string) int64 {
	creds := credentials.NewSharedCredentials("./credentials", "default")
	svc := s3.New(session.New(&aws.Config{
		Region:      aws.String(region),
		Credentials: creds,
	}))
	params := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	resp, err := svc.HeadObject(params)

	if err != nil {
		glog.Infof("file %s is not in bucket %s", key, bucket)
		return -1
	}
	glog.Infof("file %s is in bucket with size is %d", key, *resp.ContentLength)
	return *resp.ContentLength
}
