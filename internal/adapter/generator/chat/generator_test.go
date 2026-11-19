package chat

import (
	"context"
	"encoding/json"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestProviderContractAndUsage(t *testing.T) {
	data, err := os.ReadFile("../../../../examples/csv-analysis/generated.json")
	if err != nil {
		t.Fatal(err)
	}
	var pkg map[string]any
	_ = json.Unmarshal(data, &pkg)
	pkg["Generation"] = map[string]any{"UsageKnown": true, "Usage": map[string]any{"TotalTokens": 999999}}
	data, _ = json.Marshal(pkg)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" || r.Method != "POST" {
			t.Error("invalid authorization or method")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["model"] != "test-model" {
			t.Error("missing model")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "response-1", "model": "actual-model", "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]string{"content": string(data)}}}, "usage": map[string]int{"prompt_tokens": 80, "completion_tokens": 20, "total_tokens": 100}})
	}))
	defer endpoint.Close()
	generator, err := New(Config{Endpoint: endpoint.URL, Model: "test-model", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := generator.Generate(context.Background(), domain.TaskSpec{Intent: "summarize CSV"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.Name != "csv_numeric_summary" || !result.Generation.UsageKnown || result.Generation.Usage.TotalTokens != 100 || result.Generation.Model != "actual-model" {
		t.Fatalf("result=%+v", result)
	}
}

func TestProviderFailuresDoNotHideUsage(t *testing.T) {
	for _, tc := range []struct {
		name, content, finish string
		status                int
		usage                 bool
	}{
		{"invalid JSON", "oops", "stop", 200, true},
		{"truncated", "{}", "length", 200, true},
		{"missing usage", "{}", "stop", 200, false},
		{"API error", "secret", "", 429, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				body := map[string]any{"choices": []any{map[string]any{"finish_reason": tc.finish, "message": map[string]string{"content": tc.content}}}}
				if tc.usage {
					body["usage"] = map[string]int{"prompt_tokens": 1, "completion_tokens": 2, "total_tokens": 3}
				}
				_ = json.NewEncoder(w).Encode(body)
			}))
			defer endpoint.Close()
			generator, _ := New(Config{Endpoint: endpoint.URL, Model: "test"})
			g, err := generator.Generate(context.Background(), domain.TaskSpec{Intent: "task"})
			if err == nil || g.Generation.UsageKnown != tc.usage || strings.Contains(err.Error(), "secret") {
				t.Fatalf("g=%+v err=%v", g, err)
			}
		})
	}
}

func TestCanceledRequestAndInvalidConfig(t *testing.T) {
	if _, err := New(Config{Endpoint: "http://remote.invalid/completions", Model: "test"}); err == nil {
		t.Fatal("insecure endpoint accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g, _ := New(Config{Endpoint: "http://127.0.0.1:1", Model: "test"})
	if _, err := g.Generate(ctx, domain.TaskSpec{Intent: "task"}); err == nil {
		t.Fatal("cancellation ignored")
	}
}
