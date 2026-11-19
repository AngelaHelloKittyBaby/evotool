package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// TestLiveBenchmarkDocker uses a local fake provider plus real Docker. It tests
// the HTTP/learning/execution pipeline, not a vendor model's generation quality.
func TestLiveBenchmarkDocker(t *testing.T) {
	if os.Getenv("EVOTOOL_DOCKER_TEST") != "1" {
		t.Skip("optional Docker integration")
	}
	data, err := os.ReadFile("../../examples/csv-analysis/generated.json")
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "local-test", "model": "stub", "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]string{"content": string(data)}}}, "usage": map[string]int{"prompt_tokens": 80, "completion_tokens": 20, "total_tokens": 100}})
	}))
	defer provider.Close()
	t.Setenv("EVOTOOL_MODEL_ENDPOINT", provider.URL)
	t.Setenv("EVOTOOL_MODEL", "stub")
	t.Setenv("EVOTOOL_MODEL_API_KEY", "")
	root := filepath.Join(t.TempDir(), "memory")
	out := filepath.Join(t.TempDir(), "report.json")
	if err := runLiveBenchmark([]string{"--root", root, "--suite", "../../examples/benchmark/csv-live-suite.json", "--out", out}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var report domain.LiveBenchmarkReport
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || report.Baseline.SuccessRate != 1 || report.EvoTool.SuccessRate != 1 || report.EvoTool.ToolReusedCount != 1 || report.Baseline.TotalTokenUsage.TotalTokens != 200 || report.EvoTool.TotalTokenUsage.TotalTokens != 100 {
		t.Fatalf("calls=%d report=%+v", calls.Load(), report)
	}
	tool, err := filesystem.NewStores(root).Tools.Get(context.Background(), "csv_numeric_summary")
	if err != nil || tool.Generation.Usage.TotalTokens != 100 {
		t.Fatalf("generation metadata missing: %+v %v", tool.Generation, err)
	}
}
