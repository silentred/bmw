package main

import (
	"bmw/lib"
	"flag"

	"gopkg.in/fsnotify.v1"

	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

const (
	keepDuration     = 30 * 60 * time.Second
	maxFirstDelay    = 240 * time.Second
	firstDelayOffset = 60 * time.Second
	postURI          = "/v1/snap/message/sayhi_gift"
)

var (
	debug     bool
	appDomain = "fame.being.com"
	appID     = "7478343092"
	appSecret = "xougqb8m3kfkw84q954e2olfjqdiltlp"

	postURL    string
	listenPort string
)

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
	flag.BoolVar(&debug, "debug", false, "default false, using domain fame.being.com")
	loadConfig("./", "config")
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
	appDomain = viper.GetString("appDomain")
	appID = viper.GetString("appID")
	appSecret = viper.GetString("appSecret")
	listenPort = viper.GetString("listenPort")
	if listenPort == "" {
		listenPort = ":8081"
	}

	postURL = "https://" + appDomain + postURI
	fmt.Println("postURL is ", postURL)
}

//sendAutoMessage receive request in format as `msg_id:gift_id;msg_id:gift_id.....`
func sendAutoMessage(request, receiverID string) {
	items := strings.Split(request, ";")
	count := len(items)
	if count > 0 {
		interval := keepDuration / time.Duration(count)
		start(receiverID, items, interval)
	}
}

func start(receiverID string, items []string, duration time.Duration) {
	deliverMsg := makeDeliverFunc(receiverID)

	sleepTime := time.Duration(rand.Float64() * float64(maxFirstDelay))
	glog.Info("sleep for", firstDelayOffset+sleepTime)
	time.Sleep(sleepTime)

	firstItem := items[0]
	items = shift(items)
	deliverMsg(firstItem)

	tick := time.Tick(duration)
	for range tick {
		firstItem := items[0]
		items = shift(items)
		deliverMsg(firstItem)

		if len(items) == 0 {
			break
		}
	}
}

func remove(i int, items []string) []string {
	return append(items[:i], items[i+1:]...)
}

func shift(items []string) []string {
	return remove(0, items)
}

// item = msg_id:gift_id
func makeDeliverFunc(receiverID string) func(string) {
	return func(item string) {
		var msgID, giftID string

		ids := strings.Split(item, ":")
		if len(ids) < 2 {
			glog.Error("msg_id:gift_id is wrong", ids)
			return
		}
		msgID = ids[0]
		giftID = ids[1]

		headers := getSignMap()
		params := map[string]string{
			"msg_id":      msgID,
			"gift_id":     giftID,
			"receiver_id": receiverID,
		}

		config := lib.NewReqeustConfig(params, headers, 0, nil, nil)

		result, err := HTTPPostToApp(postURL, config)
		if err != nil {
			glog.Error(err)
		}
		glog.Infof("http params %+v, result %s", config, result)

		// Println(item, receiverID)
	}
}

func getSignMap() map[string]string {
	h := md5.New()
	time := int(time.Now().Unix())
	timeStr := strconv.Itoa(time)
	io.WriteString(h, timeStr)
	io.WriteString(h, appSecret)
	sign := fmt.Sprintf("%x", h.Sum(nil))

	result := map[string]string{
		"App-Id":    appID,
		"Timestamp": timeStr,
		"Sign":      sign,
	}
	return result
}
