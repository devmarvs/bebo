package realtime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/router"
)

func TestSSESend(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	ctx := bebo.NewContext(rec, req, router.Params{}, bebo.New())

	stream, err := StartSSE(ctx, SSEOptions{})
	if err != nil {
		t.Fatalf("start sse: %v", err)
	}

	if err := stream.Send(SSEMessage{Event: "update", Data: "hello"}); err != nil {
		t.Fatalf("send: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: update") {
		t.Fatalf("expected event line, got %q", body)
	}
	if !strings.Contains(body, "data: hello") {
		t.Fatalf("expected data line, got %q", body)
	}
}
