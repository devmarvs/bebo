package bebo

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type formPayload struct {
	Name   string `form:"name"`
	Age    int    `form:"age"`
	Active bool   `form:"active"`
}

type multipartPayload struct {
	Title string `form:"title"`
}

func TestBindForm(t *testing.T) {
	body := strings.NewReader("name=Kim&age=30&active=true")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := NewContext(httptest.NewRecorder(), req, nil, New())

	var payload formPayload
	if err := ctx.BindForm(&payload); err != nil {
		t.Fatalf("bind form: %v", err)
	}
	if payload.Name != "Kim" || payload.Age != 30 || !payload.Active {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestBindMultipartAndSaveFile(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	if err := writer.WriteField("title", "note"); err != nil {
		t.Fatalf("write field: %v", err)
	}
	fileWriter, err := writer.CreateFormFile("file", "note.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte("hello")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	ctx := NewContext(httptest.NewRecorder(), req, nil, New())

	var payload multipartPayload
	if err := ctx.BindMultipart(&payload, DefaultMultipartMemory); err != nil {
		t.Fatalf("bind multipart: %v", err)
	}
	if payload.Title != "note" {
		t.Fatalf("unexpected title: %s", payload.Title)
	}

	header, err := ctx.FormFile("file", DefaultMultipartMemory)
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	if header.Filename != "note.txt" {
		t.Fatalf("unexpected filename: %s", header.Filename)
	}

	dst := filepath.Join(t.TempDir(), "note.txt")
	if err := ctx.SaveUploadedFile(header, dst); err != nil {
		t.Fatalf("save uploaded file: %v", err)
	}

	contents, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(contents) != "hello" {
		t.Fatalf("unexpected file contents: %s", string(contents))
	}
}
