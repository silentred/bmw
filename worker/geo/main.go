package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hailocab/go-geoindex"
)

func main() {
	c := make(chan bool)
	m := make(map[string]string)
	go func() {
		m["1"] = "a" // First conflicting access.
		c <- true
	}()
	m["2"] = "b" // Second conflicting access.
	<-c
	for k, v := range m {
		fmt.Println(k, v)
	}
}

func testgeo() {
	// create points index with resolution (cell size) 0.5 km
	index := geoindex.NewPointsIndex(geoindex.Km(0.5))

	// Adds a point in the index, if a point with the same id exists it's removed and the new one is added
	index.Add(geoindex.NewGeoPoint("id1", 51.51, -0.11))

	// index.Remove("id3")                                  // ok
	// index.Add(geoindex.NewGeoPoint("id1", 51.53, -0.11)) // update
	// all := index.GetAll()
	// allData, _ := json.Marshal(all)
	// fmt.Fprintln(os.Stdout, string(allData))

	// add 1000000 points
	for i := 0; i < 10; i++ {
		index.Add(geoindex.NewGeoPoint(fmt.Sprintf("id_%d", i), 1.51, -0.11))
	}

	// get the k-nearest points to a point, within some distance
	points := index.KNearest(geoindex.NewGeoPoint("id2", 51.51, -0.17), 5, geoindex.Km(5), func(p geoindex.Point) bool {
		return true
	})

	data, _ := json.Marshal(points)
	fmt.Fprintln(os.Stdout, string(data))
}
