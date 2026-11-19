package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeStdio(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n")
	if err := (Server{}).ServeStdio(context.Background(), in, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("responses=%q", out.String())
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil || first["id"] != float64(1) {
		t.Fatalf("response=%v err=%v", first, err)
	}
}

func TestServeHTTPMethodAndRequestLimit(t *testing.T) {
	s := Server{}
	get := httptest.NewRecorder()
	s.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/", nil))
	if get.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d", get.Code)
	}
	tooLarge := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 2*1024*1024+1)))
	s.ServeHTTP(tooLarge, req)
	if tooLarge.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large status=%d", tooLarge.Code)
	}
}
