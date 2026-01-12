package bench

import (
	"net/http"
	"testing"

	"github.com/devmarvs/bebo/render"
	"github.com/devmarvs/bebo/router"
)

type discardResponse struct {
	header http.Header
}

func (d *discardResponse) Header() http.Header {
	if d.header == nil {
		d.header = make(http.Header)
	}
	return d.header
}

func (d *discardResponse) Write(p []byte) (int, error) {
	return len(p), nil
}

func (d *discardResponse) WriteHeader(int) {}

func BenchmarkRouterStatic(b *testing.B) {
	r := router.New()
	_, _ = r.Add(http.MethodGet, "/reports/:id")
	path := "/reports/123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = r.Match(http.MethodGet, path)
	}
}

func BenchmarkRouterWildcard(b *testing.B) {
	r := router.New()
	_, _ = r.Add(http.MethodGet, "/assets/*path")
	path := "/assets/css/app.css"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = r.Match(http.MethodGet, path)
	}
}

func BenchmarkJSONRender(b *testing.B) {
	writer := &discardResponse{}
	payload := struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}{Status: "ok", Count: 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = render.JSON(writer, http.StatusOK, payload)
	}
}
