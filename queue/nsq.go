package queue

import (
	"fmt"
	"time"

	"github.com/nsqio/go-nsq"
)

func publishNSQ(topic string, body []byte) error {
	addr := "127.0.0.1:4150"
	config := nsq.NewConfig()
	producer, err := nsq.NewProducer(addr, config)
	if err != nil {
		fmt.Println(err)
	}

	err = producer.Publish(topic, body)
	if err != nil {
		fmt.Println(err)
	}

	producer.Stop()

	return err
}

func handleNSQ() {
	config := nsq.NewConfig()
	consumer, err := nsq.NewConsumer("write_test", "chan_test", config)
	if err != nil {
		fmt.Println(err)
	}
	consumer.AddHandler(&Handler{})

	addr := "127.0.0.1:4150"
	err = consumer.ConnectToNSQD(addr)
	if err != nil {
		fmt.Println(err)
	}

	select {
	case <-consumer.StopChan:
		fmt.Println("consumer is stopped")
	case <-time.After(8 * time.Second):
		consumer.Stop()
		fmt.Println("after 2sec, stop consumer")
	}

}

type Handler struct {
}

func (h *Handler) HandleMessage(message *nsq.Message) error {
	msg := string(message.Body)
	fmt.Println("====== Message Log =======")
	fmt.Println(message.ID)
	fmt.Println(msg)
	fmt.Println("====== Message End =======")
	return nil
}
