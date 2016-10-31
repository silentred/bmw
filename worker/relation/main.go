package main

import (
	"bmw"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/spf13/viper"
	"gopkg.in/fsnotify.v1"
)

var listenPort string

//var jobChan = make(chan bmw.RetryJob)

func main() {
	flag.Parse()
	loadConfig("./", "config")

	// go func() {
	// 	for {
	// 		job := <-jobChan
	// 		startWork(job.Payload)
	// 	}
	// }()

	handler := &H{}
	route := &bmw.HandlerRoute{
		Host:    listenPort,
		Path:    "service/relation/bind",
		Topic:   "default",
		Handler: handler,
	}

	goChan := bmw.GoChannelConfig{}
	bmw := bmw.NewSingleBMW(goChan, route)

	engine := bmw.GetAPI().GetEngine()
	engine.GET("service/relation/friends", getRelation)
	engine.GET("service/relation/status", getStatus)
	engine.POST("service/relation/unbind", postUnbind)

	startCheckFriends()
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
	validTypes = viper.GetStringSlice("types")
	listenPort = viper.GetString("listenPort")
	if listenPort == "" {
		listenPort = ":8083"
	}
}

type H struct {
}

func (h *H) Handle(job bmw.RetryJob) error {
	//glog.Info(string(job.Payload))

	//use timeout
	// select {
	// case jobChan <- job:
	// case <-time.After(10 * time.Second):
	// 	return fmt.Errorf("Timeout Error")
	// }

	startWork(job.Payload)

	return nil
}

func getRelation(cxt *gin.Context) {
	uid, ok := cxt.GetQuery("uid")
	if !ok {
		cxt.JSON(200, gin.H{"error": "missing uid"})
		return
	}

	tpType, ok := cxt.GetQuery("type")
	if !ok {
		cxt.JSON(200, gin.H{"error": "missing type"})
		return
	}

	user := findUser(uid)
	if user == nil {
		cxt.JSON(200, gin.H{"error": "user is nil"})
		return
	}

	result := user.getRelation(tpType)
	glog.Infof("uid=%d, firends=%+v", user.UID, result)
	cxt.JSON(200, gin.H{"result": result})

}

func getStatus(cxt *gin.Context) {
	userCount := len(usersContainer.users)

	cxt.JSON(200, gin.H{"user_count": userCount})
}

func postUnbind(cxt *gin.Context) {
	b, err := ioutil.ReadAll(cxt.Request.Body)
	if err != nil {
		glog.Error(err)
		cxt.JSON(200, gin.H{"error": err})
	}

	err = unbindAccount(b)
	if err != nil {
		glog.Error(err)
		cxt.JSON(200, gin.H{"error": err})
	}

	cxt.JSON(200, gin.H{"result": "ok"})
}
