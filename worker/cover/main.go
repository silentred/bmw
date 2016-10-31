package main

import (
	"bmw"
	"bmw/lib"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
		Path:    "cover",
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

type body struct {
	ID     string            `json:"id"`
	Type   string            `json:"type"`
	Result map[string]string `json:"result"`
}

func (h *H) Handle(job bmw.RetryJob) error {
	result := map[string]string{}
	values, err := url.ParseQuery(string(job.Payload))
	if err != nil {
		glog.Error(err)
		return err
	}
	cover := "moment/cover/"
	fileKey := values.Get("key")
	bucket := values.Get("bucket")
	region := values.Get("region")
	fileType := values.Get("type")
	callbackURL, err := url.QueryUnescape(values.Get("callback"))
	if err != nil {
		glog.Error(err)
		return err
	}
	downloadFname, err := downloadFile(fileKey, bucket, region)
	if err != nil {
		glog.Error(err)
		return err
	}
	defer os.Remove(downloadFname)
	if fileType == "image" {
		blurFname, err := imageBlur(downloadFname)
		if err != nil {
			glog.Error(err)
			return err
		}
		blurKey := cover + fileKey + "_blur.jpg"
		err = uploadFile(blurFname, blurKey, bucket, region)
		if err != nil {
			glog.Error(err)
			return err
		}
		result["blur"] = blurKey
	} else if fileType == "video" {
		// create webp
		webpFname, err := videoToWebp(downloadFname)
		if err != nil {
			glog.Error(err)
			return err
		}
		webpKey := cover + fileKey + "_webp.webp"
		err = uploadFile(webpFname, webpKey, bucket, region)
		if err != nil {
			glog.Error(err)
			return err
		}
		result["webp"] = webpKey

		// create cover picture
		v2iFname, err := videoToImage(downloadFname)
		if err != nil {
			glog.Error(err)
			return err
		}

		// create blur picture
		blurFname, err := imageBlur(v2iFname)
		if err != nil {
			glog.Error(err)
			return err
		}

		coverKey := cover + fileKey + "_cover.jpg"
		err = uploadFile(v2iFname, coverKey, bucket, region)
		if err != nil {
			glog.Error(err)
			return err
		}
		result["cover"] = coverKey

		blurKey := cover + fileKey + "_blur.jpg"
		err = uploadFile(blurFname, blurKey, bucket, region)
		if err != nil {
			glog.Error(err)
			return err
		}
		result["blur"] = blurKey
	}
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	body := body{
		ID:     job.ID,
		Type:   fileType,
		Result: result,
	}
	bodyBytes, jErr := json.Marshal(body)
	if err != nil {
		glog.Error(jErr)
		return jErr
	}
	req := lib.NewReqeustConfig(nil, headers, 15, bodyBytes, nil)
	_, pErr := lib.HTTPPost(callbackURL, req)
	if pErr != nil {
		glog.Error(pErr)
		return pErr
	}
	return nil
}

func videoToImage(inFname string) (outFname string, outErr error) {
	TmpFp, err := ioutil.TempFile("", "png")
	if err != nil {
		outErr = err
		glog.Errorf("create videoToImage file failed %s", err)
		return
	}
	outFname = TmpFp.Name()
	TmpFp.Close()
	//defer os.Remove(pTmpFname)
	pngCmdParams := []string{
		"-y",
		"-ss", "0.1",
		"-t", "0.001",
		"-i", inFname,
		"-f", "image2",
		"-metadata:s:v", "rotate=0",
		"-vframes", "1",
		"-vf", "scale=-1:400",
		outFname,
	}
	pngCmd := exec.Command("ffmpeg", pngCmdParams...)
	if err := pngCmd.Start(); err != nil {
		outErr = err
		glog.Errorf("start ffmpeg command error, %s", err)
		return
	}
	if err := pngCmd.Wait(); err != nil {
		outErr = err
		glog.Errorf("wait ffmpeg to exit error, %s", err)
		return
	}
	if FileInfo, err := os.Stat(outFname); err != nil || FileInfo.Size() == 0 {
		outErr = err
		glog.Errorf("palettegen png create failed %s", err)
		return
	}
	return
}

func imageBlur(inFname string) (outFname string, outErr error) {
	TmpFp, err := ioutil.TempFile("", "blur")
	if err != nil {
		outErr = err
		glog.Errorf("create blur file failed %s", err)
		return
	}
	outFname = TmpFp.Name()
	TmpFp.Close()
	//defer os.Remove(pTmpFname)
	blurCmdParams := []string{
		"-blur", "45x7",
		inFname,
		outFname,
	}
	blurCmd := exec.Command("convert", blurCmdParams...)
	if err := blurCmd.Start(); err != nil {
		outErr = err
		glog.Errorf("start convert command error, %s", err)
		return
	}
	if err := blurCmd.Wait(); err != nil {
		outErr = err
		glog.Errorf("wait convert to exit error, %s", err)
		return
	}
	if FileInfo, err := os.Stat(outFname); err != nil || FileInfo.Size() == 0 {
		outErr = err
		glog.Errorf("blur image create failed %s", err)
		return
	}
	return
}

func videoToWebp(inFname string) (outFname string, outErr error) {
	//use ffmpeg create palettegen image
	pTmpFp, err := ioutil.TempFile("", "png")
	if err != nil {
		outErr = err
		glog.Errorf("open png file temp file failed %s", err)
		return
	}
	pTmpFname := pTmpFp.Name()
	pTmpFp.Close()
	//be sure to delete palettegen image
	defer os.Remove(pTmpFname)
	//prepare command
	pngCmdParams := []string{
		"-y",
		"-ss", "0",
		"-t", "2",
		"-i", inFname,
		"-f", "image2",
		"-metadata:s:v", "rotate=0",
		"-vf", "fps=7,scale=-1:400:flags=lanczos,palettegen",
		pTmpFname,
	}
	//exec command
	pngCmd := exec.Command("ffmpeg", pngCmdParams...)
	if err := pngCmd.Start(); err != nil {
		outErr = err
		glog.Errorf("start ffmpeg command error, %s", err)
		return
	}
	if err := pngCmd.Wait(); err != nil {
		outErr = err
		glog.Errorf("wait ffmpeg to exit error, %s", err)
		return
	}
	if pFileInfo, pStatErr := os.Stat(pTmpFname); pStatErr != nil || pFileInfo.Size() == 0 {
		outErr = err
		err = errors.New("palettegen png create failed")
		glog.Error(err)
		return
	}
	//use ffmpeg translate video to gif
	gTmpFp, err := ioutil.TempFile("", "gif")
	if err != nil {
		outErr = err
		glog.Errorf("open gif file temp file failed, %s", err)
		return
	}
	gTmpFname := gTmpFp.Name()
	gTmpFp.Close()
	//be sure to delete temp gif files
	defer os.Remove(gTmpFname)
	//prepare command
	gifCmdParams := []string{
		"-y",
		"-ss", "0",
		"-t", "2",
		"-v", "error",
		"-i", inFname,
		"-i", pTmpFname,
		"-metadata:s:v", "rotate=0",
		"-filter_complex", "fps=8,scale=-1:400:flags=lanczos[x];[x][1:v]paletteuse",
		"-f", "gif",
		gTmpFname,
	}
	gifCmd := exec.Command("ffmpeg", gifCmdParams...)
	if err := gifCmd.Start(); err != nil {
		outErr = err
		glog.Errorf("start ffmpeg command error, %s", err)
		return
	}
	if err := gifCmd.Wait(); err != nil {
		outErr = err
		glog.Errorf("wait ffmpeg to exit error, %s", err)
		return
	}
	if gFileInfo, err := os.Stat(gTmpFname); err != nil || gFileInfo.Size() == 0 {
		outErr = err
		glog.Errorf("video translate to gif failed %s", err)
		return
	}
	//use gif2webp translate gif to animate webp
	wTmpFp, err := ioutil.TempFile("", "webp")
	if err != nil {
		outErr = err
		glog.Errorf("open webp file temp file failed, %s", err)
		return
	}
	wTmpFp.Close()
	outFname = wTmpFp.Name()
	//defer os.Remove(outFname)
	webpCmdParams := []string{
		"-lossy", gTmpFname,
		"-o", outFname,
	}
	webpCmd := exec.Command("gif2webp", webpCmdParams...)
	if err := webpCmd.Start(); err != nil {
		outErr = err
		glog.Errorf("start gif2webp command error, %s", err)
		return
	}
	if err := webpCmd.Wait(); err != nil {
		outErr = err
		glog.Errorf("wait gif2webp to exit error, %s", err)
		return
	}
	if wFileInfo, err := os.Stat(outFname); err != nil || wFileInfo.Size() == 0 {
		outErr = err
		defer os.Remove(outFname)
		return
	}
	return
}

func downloadFile(key, bucket, region string) (outFname string, outErr error) {
	tmpFp, err := ioutil.TempFile("", "tmp")
	outFname = tmpFp.Name()
	if err != nil {
		outErr = err
		glog.Errorf("Failed to create file %s, error: %s", outFname, err)
		return
	}
	defer tmpFp.Close()
	creds := credentials.NewSharedCredentials("./credentials", "default")
	downloader := s3manager.NewDownloader(session.New(&aws.Config{
		Region:      aws.String(region),
		Credentials: creds,
	}))
	numBytes, err := downloader.Download(tmpFp,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
	)
	if err != nil {
		outErr = err
		glog.Errorf("Failed to download file %s name %s, error: %s", key, outFname, err)
		return
	}
	//defer os.Remove(tmpFname)
	glog.Infof("Downloaded file %s name %s bytes %d", key, outFname, numBytes)
	return
}

func uploadFile(fName, key, bucket, region string) error {
	tmpFp, err := os.Open(fName)
	if err != nil {
		glog.Errorf("Failed to open upload-file %s, error: %s", key, err)
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
	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:   reader,
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		ACL:    aws.String("public-read"),
	})
	if err != nil {
		glog.Errorf("Failed to upload file %s error: %s", key, err)
		return err
	}
	defer os.Remove(fName)
	glog.Infof("Successfully upload file %s", result.Location)
	return nil
}
