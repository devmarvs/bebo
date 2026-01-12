package bebo_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/testutil"
)

func TestGoldenJSONResponse(t *testing.T) {
	app := bebo.New()
	app.GET("/hello", func(ctx *bebo.Context) error {
		payload := struct {
			Status string `json:"status"`
		}{Status: "ok"}
		return ctx.JSON(http.StatusOK, payload)
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	path := filepath.Join("testdata", "golden", "hello.json")
	testutil.AssertGolden(t, path, rec.Body.Bytes())
}
