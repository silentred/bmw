package main

import (
	"bmw"
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/spf13/viper"
	"gopkg.in/fsnotify.v1"
)

var listenPort = ":8085"

func main() {
	flag.Parse()
	loadConfig("./", "config")

	handler := &handler{}
	batchHandler := &batchHandler{}

	routes := []bmw.SingleRoute{
		bmw.SingleRoute{
			Path:    "service/resource/upload",
			Topic:   "upload",
			Handler: handler,
		},
		bmw.SingleRoute{
			Path:    "service/resource/upload/batch",
			Topic:   "upload-batch",
			Handler: batchHandler,
		},
	}
	handlerRoute := new(bmw.MultiHandlerRoutes)
	handlerRoute.Host = listenPort
	handlerRoute.Routes = routes

	goChan := bmw.GoChannelConfig{}
	bmw := bmw.NewMultiBMW(goChan, handlerRoute)

	engine := bmw.GetAPI().GetEngine()
	engine.GET("service/resource/key", findKeyQuery)

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
		initMysql()
	})

	setVar()
	initMysql()
}

func setVar() {
	listenPort = viper.GetString("listenPort")
	QiniuAccessToken = viper.GetString("qiniuAccessToken")
	QiniuSecretToken = viper.GetString("qiniuSecretToken")
	S3Region = viper.GetString("s3Region")
	S3Credentials = viper.GetString("s3Credentials")
	bucketDomainMap = viper.GetStringMapString("bucketDomainMap")
	allBuckets = viper.GetStringSlice("allBuckets")
	mysqlConn = viper.GetString("mysqlConn")
}

type handler struct {
}

func (h *handler) Handle(job bmw.RetryJob) error {
	return handleRequest(&job)
}

type batchHandler struct {
}

func (h *batchHandler) Handle(job bmw.RetryJob) error {
	return handleBatchRequest(&job)
}
