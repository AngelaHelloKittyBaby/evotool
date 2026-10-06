package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

func TestToolStoreSaveAndGet(t *testing.T) {
	root := t.TempDir()
	store := NewToolStore(root)
	ctx := context.Background()

	tool := domain.Tool{
		ID:             "pdf_to_excel",
		Name:           "pdf_to_excel",
		Description:    "Convert PDF tables to Excel",
		CurrentVersion: "v1",
		Manifest: domain.ToolManifest{
			Name:        "pdf_to_excel",
			Description: "Convert PDF tables to Excel",
			Runtime:     "python",
			EntryPoint:  "main.py",
		},
		Versions: []domain.ToolVersion{
			{
				Version: "v1",
				Files: []domain.SourceFile{
					{Path: "main.py", Content: "print('ok')"},
					{Path: "tests/test_main.py", Content: "def test_ok(): assert True"},
				},
			},
		},
	}

	if err := store.Save(ctx, tool); err != nil {
		t.Fatalf("save tool: %v", err)
	}

	loaded, err := store.Get(ctx, "pdf_to_excel")
	if err != nil {
		t.Fatalf("get tool: %v", err)
	}
	if loaded.Name != tool.Name {
		t.Fatalf("loaded tool name = %q, want %q", loaded.Name, tool.Name)
	}

	assertFile(t, filepath.Join(root, "tools", "pdf_to_excel", "manifest.json"))
	assertFile(t, filepath.Join(root, "tools", "pdf_to_excel", "versions", "v1", "main.py"))
	assertFile(t, filepath.Join(root, "tools", "pdf_to_excel", "versions", "v1", "tests", "test_main.py"))
}

func TestLibraryStoreSaveAndGet(t *testing.T) {
	root := t.TempDir()
	store := NewLibraryStore(root)
	ctx := context.Background()

	library := domain.Library{
		ID:             "pdf_parser",
		Name:           "pdf_parser",
		Description:    "Parse PDF files",
		CurrentVersion: "v1",
		Manifest: domain.LibraryManifest{
			Name:        "pdf_parser",
			Description: "Parse PDF files",
			Runtime:     "python",
		},
		Versions: []domain.LibraryVersion{
			{
				Version: "v1",
				Files: []domain.SourceFile{
					{Path: "source/parser.py", Content: "def parse(): pass"},
				},
			},
		},
	}

	if err := store.Save(ctx, library); err != nil {
		t.Fatalf("save library: %v", err)
	}

	loaded, err := store.Get(ctx, "pdf_parser")
	if err != nil {
		t.Fatalf("get library: %v", err)
	}
	if loaded.Name != library.Name {
		t.Fatalf("loaded library name = %q, want %q", loaded.Name, library.Name)
	}

	assertFile(t, filepath.Join(root, "libraries", "pdf_parser", "manifest.json"))
	assertFile(t, filepath.Join(root, "libraries", "pdf_parser", "versions", "v1", "source", "parser.py"))
}

func TestRegistryAndRecorders(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	registry := NewRegistryStore(root)
	executions := NewExecutionRecorder(root)
	audit := NewAuditLogger(root)

	tool := domain.Tool{ID: "csv_analyzer", Name: "csv_analyzer"}
	if err := registry.UpsertToolMetadata(ctx, tool); err != nil {
		t.Fatalf("upsert tool metadata: %v", err)
	}
	assertFile(t, filepath.Join(root, "registry", "metadata", "tools", "csv_analyzer.json"))

	graph := domain.DependencyGraph{
		Nodes: []domain.DependencyNode{{ID: "csv_analyzer", Name: "csv_analyzer", Kind: domain.DependencyTool}},
	}
	if err := registry.SaveDependencyGraph(ctx, graph); err != nil {
		t.Fatalf("save dependency graph: %v", err)
	}
	loaded, err := registry.GetDependencyGraph(ctx, "csv_analyzer")
	if err != nil {
		t.Fatalf("get dependency graph: %v", err)
	}
	if len(loaded.Nodes) != 1 {
		t.Fatalf("dependency node count = %d, want 1", len(loaded.Nodes))
	}

	if err := executions.RecordExecution(ctx, domain.ExecutionResult{ToolID: "csv_analyzer", Version: "v1", Success: true}); err != nil {
		t.Fatalf("record execution: %v", err)
	}
	assertFile(t, filepath.Join(root, "registry", "execution_history", "csv_analyzer.jsonl"))

	if err := audit.Log(ctx, domain.AuditEvent{Action: "tool.saved", SubjectID: "csv_analyzer", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("log audit: %v", err)
	}
	assertFile(t, filepath.Join(root, "registry", "audit", "events.jsonl"))
}

func TestRejectsUnsafeSourcePath(t *testing.T) {
	store := NewToolStore(t.TempDir())
	err := store.Save(context.Background(), domain.Tool{
		ID:   "bad_tool",
		Name: "bad_tool",
		Versions: []domain.ToolVersion{
			{Version: "v1", Files: []domain.SourceFile{{Path: "../escape.py", Content: "bad"}}},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func assertFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file %s: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("expected file %s, got directory", path)
	}
}
