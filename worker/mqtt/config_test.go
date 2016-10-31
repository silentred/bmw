package mqtt

import (
	"fmt"
	"testing"
	"github.com/golang/glog"
)

func TestConfig(t *testing.T) {
	file := "./app/config.toml"
	config, err := NewConfig(file)
	if err != nil {
		glog.Fatal(err)
	}

	fmt.Println(config.Api.Host)
}
