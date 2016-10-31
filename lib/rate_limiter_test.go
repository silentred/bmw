package lib

import (
	"log"
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	limiter := NewRateLimiter(3)
	start := time.Now()

	for i := 0; i < 30; i++ {
		if limiter.Limit() {
			log.Printf("i is %d \n", i)
		}
	}

	end := time.Now()

	d := end.Sub(start)
	log.Println("my limiter spends seconds: ", d)
}
