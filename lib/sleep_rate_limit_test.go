package lib

import (
	"log"
	"testing"
	"time"
)

func TestSleepLimiter(t *testing.T) {
	var limiter LimitRate
	limiter.SetRate(3)

	start := time.Now()

	for i := 0; i < 30; i++ {
		if limiter.Limit() {
			//log.Printf("i is %d \n", i)
		}
	}

	end := time.Now()

	d := end.Sub(start)
	log.Println("sleep limiter spends seconds: ", d)
}
