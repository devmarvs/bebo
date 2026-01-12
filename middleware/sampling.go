package middleware

import (
	"math/rand"
	"sync"
	"time"

	"github.com/devmarvs/bebo"
)

// Sampler decides whether to sample a request.
type Sampler func(*bebo.Context) bool

// SampleRate returns a sampler that samples a percentage of requests.
func SampleRate(rate float64) Sampler {
	if rate >= 1 {
		return func(*bebo.Context) bool { return true }
	}
	if rate <= 0 {
		return func(*bebo.Context) bool { return false }
	}
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	var mu sync.Mutex
	return func(*bebo.Context) bool {
		mu.Lock()
		value := rnd.Float64()
		mu.Unlock()
		return value < rate
	}
}
