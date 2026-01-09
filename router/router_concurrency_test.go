package router

import (
	"fmt"
	"sync"
	"testing"
)

func TestRouterConcurrentMatch(t *testing.T) {
	r := New()
	for i := 0; i < 500; i++ {
		path := fmt.Sprintf("/users/%d", i)
		if _, err := r.Add("GET", path); err != nil {
			t.Fatalf("add: %v", err)
		}
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				path := fmt.Sprintf("/users/%d", j)
				_, _, ok := r.Match("GET", path)
				if !ok {
					select {
					case errCh <- fmt.Errorf("match failed for %s", path):
					default:
					}
					return
				}
				_ = r.Allowed(path)
			}
		}(i)
	}

	wg.Wait()
	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}
}
