package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestCreateUser(t *testing.T) {
	user := new(User)
	user.Friends = make(map[string][]UserFriend)

	fmt.Printf("%#v \n", user.Friends["fb"])
	if value, ok := user.Friends["fb"]; ok {
		fmt.Println("has fb", value)
	} else {
		user.Friends["fb"] = make([]UserFriend, 0)
		fmt.Printf("%#v \n", user.Friends["fb"])
	}

	userFriend := UserFriend{
		TpID:   "test",
		TpName: "testName",
	}

	if userFriend.UID == "" {
		fmt.Printf("userFriend is  %+v \n", userFriend)
	}
}

func TestRequestRace(t *testing.T) {
	req := newInputRequest()
	b, _ := json.Marshal(req)

	for i := 0; i < 2; i++ {
		go startWork(b)
	}

	select {}
}

func newInputRequest() InputRequest {

	friend := InputFriend{TpID: "2", TpName: "2name"}
	var friends []InputFriend

	req := InputRequest{
		UID:     "1",
		Type:    "fb",
		TpID:    "1",
		Friends: append(friends, friend),
	}

	return req
}
