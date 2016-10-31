package main

import (
	"fmt"
	"testing"
)

func TestMysql(t *testing.T) {
	result := findKey("a")
	fmt.Println(result)

	res := map[string]int{"b": 123}

	err := store("a", res)
	if err != nil {
		panic(err)
	}
}
