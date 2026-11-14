package main

import (
	"context"
	"testing"
	"time"

	localretriever "github.com/AngelaHelloKittyBaby/evotool/internal/adapter/retriever/local"
	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/service"
)

// Tests seed capability memory through the store API instead of a demo CLI
// command: fixtures that ship in the binary would be indistinguishable from
// real learned capabilities once saved.

func TestRunSearchTools(t *testing.T) {
	root := t.TempDir()
	seedTool(t, root, cliTestTool("csv_numeric_summary", "data_analysis", "python", []string{"csv", "numeric", "summary"}))

	if err := run([]string{"search", "tools", "--root", root, "--query", "csv numeric summary", "--limit", "1"}); err != nil {
		t.Fatalf("run search tools: %v", err)
	}
}

func TestRunSearchToolsRequiresQuery(t *testing.T) {
	if err := run([]string{"search", "tools"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunSearchToolsExcludesArchived(t *testing.T) {
	root := t.TempDir()
	archived := cliTestTool("csv_numeric_summary", "data_analysis", "python", []string{"csv", "numeric", "summary"})
	archived.LifecycleStatus = domain.LifecycleArchived
	seedTool(t, root, archived)

	retrieval, err := service.NewRetrievalService(service.RetrievalServiceConfig{
		IntentExtractor: localretriever.IntentExtractor{
			SemanticTopK: 10,
			FilterTopK:   10,
			RankTopK:     5,
			Filters:      domain.RetrievalFilters{AllowedStatuses: domain.DefaultRetrievableStatuses()},
		},
		ToolRetriever: localretriever.NewToolRetriever(root),
		Ranker:        localretriever.NewCapabilityRanker(),
	})
	if err != nil {
		t.Fatalf("new retrieval service: %v", err)
	}
	result, err := retrieval.Search(context.Background(), domain.TaskSpec{Intent: "csv numeric summary"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("archived tool surfaced in default search: %+v", result.Candidates)
	}
}

func TestRunSearchLibraries(t *testing.T) {
	root := t.TempDir()
	seedLibrary(t, root, cliTestLibrary("csv_stats_core", "data_analysis", "python", []string{"csv", "numeric", "statistics"}))

	if err := run([]string{"search", "libraries", "--root", root, "--query", "csv numeric statistics", "--limit", "1"}); err != nil {
		t.Fatalf("run search libraries: %v", err)
	}
}

func TestRunSearchLibrariesRequiresQuery(t *testing.T) {
	if err := run([]string{"search", "libraries"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunDepsGraph(t *testing.T) {
	root := t.TempDir()
	seedLinkedCapabilities(t, root)

	if err := run([]string{"deps", "graph", "--root", root, "--id", "csv_numeric_summary"}); err != nil {
		t.Fatalf("run deps graph: %v", err)
	}
	if err := run([]string{"deps", "check", "--root", root, "--id", "csv_numeric_summary"}); err != nil {
		t.Fatalf("run deps check: %v", err)
	}
}

func TestRunDepsGraphRequiresID(t *testing.T) {
	if err := run([]string{"deps", "graph"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunDepsCheckRequiresID(t *testing.T) {
	if err := run([]string{"deps", "check"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunLifecycleScan(t *testing.T) {
	root := t.TempDir()
	seedTool(t, root, cliTestTool("csv_numeric_summary", "data_analysis", "python", []string{"csv", "numeric", "summary"}))

	stores := filesystem.NewStores(root)
	tool, err := stores.Tools.Get(context.Background(), "csv_numeric_summary")
	if err != nil {
		t.Fatalf("get tool: %v", err)
	}
	tool.UpdatedAt = time.Now().UTC().Add(-48 * time.Hour)
	if err := stores.Tools.Update(context.Background(), tool); err != nil {
		t.Fatalf("age tool: %v", err)
	}

	if err := run([]string{
		"lifecycle",
		"scan",
		"--root",
		root,
		"--stale-days",
		"1",
		"--archive-days",
		"3",
	}); err != nil {
		t.Fatalf("run lifecycle scan: %v", err)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if err := run([]string{"unknown"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunRemovedDemoCommand(t *testing.T) {
	if err := run([]string{"demo", "save-tool"}); err == nil {
		t.Fatal("demo command must stay removed rather than ship fabricated capabilities")
	}
}

func seedTool(t *testing.T, root string, tool domain.Tool) {
	t.Helper()
	stores := filesystem.NewStores(root)
	if err := stores.Tools.Save(context.Background(), tool); err != nil {
		t.Fatalf("save tool: %v", err)
	}
	if err := stores.Registry.UpsertToolMetadata(context.Background(), tool); err != nil {
		t.Fatalf("upsert tool metadata: %v", err)
	}
}

func seedLibrary(t *testing.T, root string, library domain.Library) {
	t.Helper()
	stores := filesystem.NewStores(root)
	if err := stores.Libraries.Save(context.Background(), library); err != nil {
		t.Fatalf("save library: %v", err)
	}
	if err := stores.Registry.UpsertLibraryMetadata(context.Background(), library); err != nil {
		t.Fatalf("upsert library metadata: %v", err)
	}
}

func seedLinkedCapabilities(t *testing.T, root string) {
	t.Helper()
	tool := cliTestTool("csv_numeric_summary", "data_analysis", "python", []string{"csv", "numeric", "summary"})
	library := cliTestLibrary("csv_stats_core", "data_analysis", "python", []string{"csv", "numeric", "statistics"})
	seedTool(t, root, tool)
	seedLibrary(t, root, library)
	memory, err := newCapabilityMemory(root)
	if err != nil {
		t.Fatalf("new capability memory: %v", err)
	}
	if _, err := memory.LinkToolLibraries(context.Background(), tool, []domain.Library{library}); err != nil {
		t.Fatalf("link tool library: %v", err)
	}
}

func cliTestTool(name, category, runtime string, tags []string) domain.Tool {
	now := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	return domain.Tool{
		ID:              name,
		Name:            name,
		Description:     "Compute numeric summary statistics over CSV columns.",
		Category:        category,
		CurrentVersion:  "v1",
		LifecycleStatus: domain.LifecycleApproved,
		TrustLevel:      domain.TrustValidated,
		CreatedAt:       now,
		UpdatedAt:       now,
		Manifest: domain.ToolManifest{
			Name:         name,
			Description:  "Compute numeric summary statistics over CSV columns.",
			Category:     category,
			Runtime:      runtime,
			EntryPoint:   "main.py",
			Capabilities: tags,
			Tags:         tags,
			Environment:  domain.EnvironmentSpec{Runtime: runtime},
		},
	}
}

func cliTestLibrary(name, category, runtime string, tags []string) domain.Library {
	now := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	return domain.Library{
		ID:              name,
		Name:            name,
		Description:     "Shared CSV statistics primitives.",
		Category:        category,
		CurrentVersion:  "v1",
		LifecycleStatus: domain.LifecycleApproved,
		TrustLevel:      domain.TrustValidated,
		CreatedAt:       now,
		UpdatedAt:       now,
		Manifest: domain.LibraryManifest{
			Name:        name,
			Description: "Shared CSV statistics primitives.",
			Category:    category,
			Runtime:     runtime,
			Tags:        tags,
		},
	}
}
