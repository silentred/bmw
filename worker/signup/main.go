package main

import (
	"bmw"
	"flag"
	"net/url"

	"github.com/golang/glog"
)

func main() {
	flag.Parse()

	route := &bmw.HandlerRoute{
		Host:    listenPort,
		Path:    "test",
		Topic:   "default",
		Handler: new(H),
	}
	//beanstalk := bmw.BeanstalkConfig{Host: "localhost:11300"}
	goChan := bmw.GoChannelConfig{}
	bmw := bmw.NewSingleBMW(goChan, route)
	bmw.Start()
}

type H struct {
}

func (h *H) Handle(job bmw.RetryJob) error {
	values, err := url.ParseQuery(string(job.Payload))
	if err != nil {
		glog.Error(err)
		return err
	}

	receiverID := values.Get("receiver_id")
	items := values.Get("items")
	glog.Infof("jobID: %s receiverID: %s items: %s", job.ID, receiverID, items)

	go sendAutoMessage(items, receiverID)

	return nil
}

// func msg() {
// 	request := "1:2;2:2;3:2;4:2;5:2;6:2"
// 	go signup.SendAutoMessage(request, "100195")
// 	signup.SendAutoMessage(request, "200")
// 	select {}
// }

// func https() {
// 	params := map[string]string{
// 		"msg_id":      "1870",
// 		"gift_id":     "1",
// 		"receiver_id": "10010095",
// 	}
// 	headers := signup.GetSignMap()
// 	//params["app_id"] = headers["app_id"]
// 	config := lib.NewReqeustConfig(params, headers, 0, nil, nil)
// 	result, err := signup.HTTPPostToApp("https://dev-fame.being.com/v1/snap/message/sayhi_gift", config)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	fmt.Printf("%s\n", string(result))
// }

// func sleep(i int) {
// 	sec := rand.Float32() * 10
// 	time.Sleep(time.Duration(sec) * time.Second)
// 	fmt.Printf("%d(%f) - ", i, sec)

// 	client1 := http.DefaultClient
// 	client2 := *client1

// 	fmt.Printf("client1 ptr is %p, type is %T \n", client1, client1)
// 	fmt.Printf("client2 ptr is %p, type is %T, %#v \n", &client2, nil, nil)

// 	time := 10 * time.Second
// 	subTime := time / 5
// 	fmt.Printf("type is %T, %#v \n", subTime, subTime)
// }
