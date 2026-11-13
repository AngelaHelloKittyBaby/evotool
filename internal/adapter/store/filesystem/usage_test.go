package filesystem

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

func TestUsageRecorderUpdatesToolAndRegistryMetadata(t *testing.T) {
	root := t.TempDir()
	stores := NewStores(root)
	ctx := context.Background()
	saved := usageTestTool("csv_numeric_summary", time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if err := stores.Tools.Save(ctx, saved); err != nil {
		t.Fatalf("save tool: %v", err)
	}
	if err := stores.Registry.UpsertToolMetadata(ctx, saved); err != nil {
		t.Fatalf("upsert metadata: %v", err)
	}

	finished := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	for _, result := range []domain.ExecutionResult{
		{Success: true, Duration: 2 * time.Second, FinishedAt: finished},
		{Success: false, Duration: time.Second, FinishedAt: finished.Add(time.Minute)},
	} {
		if err := stores.Usage.RecordUsage(ctx, domain.ExecutionResult{
			ToolID: resultToolID, Version: "v1", Success: result.Success,
			Duration: result.Duration, FinishedAt: result.FinishedAt,
		}); err != nil {
			t.Fatalf("record usage: %v", err)
		}
	}

	want := domain.ToolUsageStats{
		UsageCount: 2, SuccessCount: 1, FailureCount: 1, SuccessRate: 0.5,
		AverageLatency: 1500 * time.Millisecond, LastUsedAt: finished.Add(time.Minute),
	}
	for name, load := range map[string]func() (domain.Tool, error){
		"tool store": func() (domain.Tool, error) { return stores.Tools.Get(ctx, resultToolID) },
		"registry":   func() (domain.Tool, error) { return readToolMetadata(root, resultToolID) },
	} {
		tool, err := load()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if tool.UsageStats != want {
			t.Fatalf("%s stats = %+v, want %+v", name, tool.UsageStats, want)
		}
		if tool.Versions == nil || len(tool.Versions) != 1 {
			t.Fatalf("%s lost version data: %+v", name, tool.Versions)
		}
	}
}

func TestUsageRecorderRequiresStoredTool(t *testing.T) {
	stores := NewStores(t.TempDir())
	err := stores.Usage.RecordUsage(context.Background(), domain.ExecutionResult{ToolID: "missing_tool"})
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected missing tool error, got %v", err)
	}
}

func TestUsageRecorderRejectsUnsafeToolID(t *testing.T) {
	stores := NewStores(t.TempDir())
	if err := stores.Usage.RecordUsage(context.Background(), domain.ExecutionResult{ToolID: "../escape"}); err == nil {
		t.Fatal("expected unsafe tool id to be rejected")
	}
}

const resultToolID = "csv_numeric_summary"

func usageTestTool(name string, created time.Time) domain.Tool {
	return domain.Tool{
		ID:              name,
		Name:            name,
		Description:     "Test tool",
		Category:        "data_analysis",
		CurrentVersion:  "v1",
		LifecycleStatus: domain.LifecycleApproved,
		TrustLevel:      domain.TrustValidated,
		CreatedAt:       created,
		UpdatedAt:       created,
		Versions:        []domain.ToolVersion{{Version: "v1", Files: []domain.SourceFile{{Path: "main.py", Content: "print('{}')"}}, CreatedAt: created}},
		Manifest:        domain.ToolManifest{Name: name, Runtime: "python", EntryPoint: "main.py"},
	}
}

func readToolMetadata(root string, id string) (domain.Tool, error) {
	var tool domain.Tool
	err := readJSON(root+"/registry/metadata/tools/"+id+".json", &tool)
	return tool, err
}
