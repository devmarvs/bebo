package middleware

import (
	"sync"
	"testing"
)

func TestLimiterConcurrentAllow(t *testing.T) {
	limiter := NewLimiter(1000, 50)

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_ = limiter.Allow("client")
			}
		}()
	}

	wg.Wait()
}
