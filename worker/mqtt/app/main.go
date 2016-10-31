package main

import (
	"bmw/worker/mqtt"
	"fmt"
)

func main() {
	config, err := mqtt.NewConfig("./config.toml")
	if err != nil {
		panic(err)
	}

	worker := mqtt.NewMqttWorker(config)

	fmt.Println(worker)
}
