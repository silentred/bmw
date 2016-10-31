package main

import (
	"bmw"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"time"

	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/spf13/viper"
	fsnotify "gopkg.in/fsnotify.v1"
)

type Recorder struct {
	File     *os.File
	Key      string
	Bucket   string
	Region   string
	UpdateAt time.Time
}

var (
	listenPort       string
	mqttServer       string
	mqttClientid     string
	mqttUsername     string
	mqttPassword     string
	mqttCleanSession bool
	mqttQos          byte
	topicName        string
	client           *MQTT.Client
	recorders        = make(map[string]*Recorder)
)

type H struct {
}

func onMessageReceived(client *MQTT.Client, message MQTT.Message) {
	if recorder, ok := recorders[message.Topic()]; ok {
		body := message.Payload()

		Fp := recorder.File
		n, err := Fp.Write(body)
		if err != nil {
			glog.Errorf("err: %s, n: %d", err, n)
		}

		Fp.WriteString("\n")

		recorder.UpdateAt = time.Now()
	}
}

func connectMqtt() *MQTT.Client {
	connOpts := MQTT.NewClientOptions()
	connOpts.SetClientID(mqttClientid)
	connOpts.SetCleanSession(mqttCleanSession)
	connOpts.SetUsername(mqttUsername)
	connOpts.SetPassword(mqttPassword)
	connOpts.AddBroker(mqttServer)

	mqttClient := MQTT.NewClient(connOpts)
	if token := mqttClient.Connect(); token.WaitTimeout(20*time.Second) && token.Error() != nil {
		glog.Errorf("Connect to MQTT Broker %s error: %s", mqttServer, token.Error())
	} else {
		glog.Infof("Connected to %s successed", mqttServer)
	}
	return mqttClient
}

func syncFile(topic string) {
	ticker := time.NewTicker(300 * time.Second)

	for {
		select {
		case <-ticker.C:
			if recorder, ok := recorders[topic]; ok {
				Fp := recorder.File
				Fname := Fp.Name()
				if recorder.UpdateAt.Add(5 * time.Minute).After(time.Now()) {
					//upload file if receive any message in 5 minutes
					Fp.Sync()
					// what happens when uploading same file simultaneously?
					go uploadFile(Fname, recorder.Key, recorder.Bucket, recorder.Region, false)
				} else if recorder.UpdateAt.Add(2 * time.Hour).Before(time.Now()) {
					//Unsubscribe topic and deliver resource if don't receive any message in 2 hours
					if token := client.Unsubscribe(topic); token.WaitTimeout(20*time.Second) && token.Error() != nil {
						glog.Errorf("Unsubscribe topic %s error: %s", topic, token.Error())
					} else {
						glog.Infof("Unsubscribe topic %s success", topic)
					}
					Fp.Close()
					os.Remove(Fname)
					delete(recorders, topic)
					return
				}
			} else {
				return
			}
		}
	}
}

func uploadFile(filename, key, bucket, region string, removeFlag bool) error {
	tmpFp, err := os.Open(filename)
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
	glog.Infof("Successfully upload file %s", result.Location)
	if removeFlag {
		os.Remove(filename)
	}
	return nil
}

func loadConfig(path, name string) {
	viper.SetConfigName(name)
	viper.AddConfigPath(path)
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		glog.Errorf("Fatal error config file: %s \n", err)
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
	mqttServer = viper.GetString("mqttServer")
	mqttClientid = viper.GetString("mqttClientid")
	mqttUsername = viper.GetString("mqttUsername")
	mqttPassword = viper.GetString("mqttPassword")
	mqttCleanSession = viper.GetBool("mqttCleanSession")
	mqttQos = byte(viper.GetInt("mqttQos"))
	topicName = viper.GetString("topicName")
	if listenPort == "" {
		listenPort = ":8087"
	}
}

func startRecorder(cxt *gin.Context) {
	topic, ok := cxt.GetQuery("sub_topic")
	if !ok {
		cxt.JSON(200, gin.H{"error": "missing sub_topic"})
		return
	}

	key, ok := cxt.GetQuery("key")
	if !ok {
		cxt.JSON(200, gin.H{"error": "missing key"})
		return
	}

	bucket, ok := cxt.GetQuery("bucket")
	if !ok {
		cxt.JSON(200, gin.H{"error": "missing bucket"})
		return
	}

	region, ok := cxt.GetQuery("region")
	if !ok {
		cxt.JSON(200, gin.H{"error": "missing region"})
		return
	}

	glog.Infof("Received start-recorder with topicname %s; key %s; bucket %s;", topic, key, bucket)

	if _, ok := recorders[topic]; !ok {
		Fp, err := ioutil.TempFile("", "txt")
		if err != nil {
			glog.Errorf("Create recorder file error %s", err.Error())
			cxt.JSON(200, gin.H{"error": "can't create file"})
			return
		}
		glog.Infof("Create recorder file %s success", Fp.Name())

		recorder := &Recorder{Fp, key, bucket, region, time.Now()}
		recorders[topic] = recorder
		cxt.JSON(200, gin.H{"result": "ok"})
		glog.Infof("response start-recorder has done")

		if token := client.Subscribe(topic, mqttQos, onMessageReceived); token.WaitTimeout(20*time.Second) && token.Error() != nil {
			glog.Errorf("Subscribe topic %s error: %s", topic, token.Error())
			return
		}
		glog.Infof("Subscribe topic %s success", topic)
		go syncFile(topic)
	}
}

func stopRecorder(cxt *gin.Context) {
	topic, ok := cxt.GetQuery("sub_topic")
	if !ok {
		cxt.JSON(200, gin.H{"error": "missing sub_topic"})
		return
	}

	glog.Infof("Received stop-recorder resquest with topicname %s", topic)

	if token := client.Unsubscribe(topic); token.WaitTimeout(20*time.Second) && token.Error() != nil {
		glog.Errorf("Unsubscribe topic %s error: %s", topic, token.Error())
	} else {
		glog.Infof("Unsubscribe topic %s success", topic)
	}

	if recorder, ok := recorders[topic]; ok {
		Fp := recorder.File
		Fname := Fp.Name()

		Fp.Sync()
		Fp.Close()

		go uploadFile(Fname, recorder.Key, recorder.Bucket, recorder.Region, true)
		delete(recorders, topic)
	}
	cxt.JSON(200, gin.H{"result": "ok"})
}

func main() {
	flag.Parse()
	loadConfig("./", "config")
	route := &bmw.HandlerRoute{
		Host:    listenPort,
		Path:    "recorder",
		Topic:   "default",
		Handler: new(H),
	}

	client = connectMqtt()

	goChan := bmw.GoChannelConfig{}
	bmw := bmw.NewSingleBMW(goChan, route)
	engine := bmw.GetAPI().GetEngine()
	engine.GET("recorder/start", startRecorder)
	engine.GET("recorder/end", stopRecorder)

	bmw.Start()
}

func (h *H) Handle(job bmw.RetryJob) error {
	//startWork(job.Payload)
	return nil
}
