package router

import (
	"fmt"
	"testing"
)

func BenchmarkMatchStatic(b *testing.B) {
	r := New()
	for i := 0; i < 1000; i++ {
		path := fmt.Sprintf("/static/%d", i)
		if _, err := r.Add("GET", path); err != nil {
			b.Fatalf("add: %v", err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/static/%d", i%1000)
		_, _, _ = r.Match("GET", path)
	}
}

func BenchmarkMatchParam(b *testing.B) {
	r := New()
	if _, err := r.Add("GET", "/users/:id"); err != nil {
		b.Fatalf("add: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = r.Match("GET", "/users/42")
	}
}
