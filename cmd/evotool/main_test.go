package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDemoSaveTool(t *testing.T) {
	root := t.TempDir()
	if err := run([]string{"demo", "save-tool", "--root", root}); err != nil {
		t.Fatalf("run demo save-tool: %v", err)
	}

	assertFile(t, filepath.Join(root, "tools", "pdf_to_excel", "manifest.json"))
	assertFile(t, filepath.Join(root, "tools", "pdf_to_excel", "metadata.json"))
	assertFile(t, filepath.Join(root, "tools", "pdf_to_excel", "versions", "v1", "main.py"))
	assertFile(t, filepath.Join(root, "registry", "metadata", "tools", "pdf_to_excel.json"))
	assertFile(t, filepath.Join(root, "registry", "audit", "events.jsonl"))
}

func TestRunSearchTools(t *testing.T) {
	root := t.TempDir()
	if err := run([]string{"demo", "save-tool", "--root", root}); err != nil {
		t.Fatalf("run demo save-tool: %v", err)
	}
	if err := run([]string{"search", "tools", "--root", root, "--query", "pdf table excel", "--limit", "1"}); err != nil {
		t.Fatalf("run search tools: %v", err)
	}
}

func TestRunSearchToolsRequiresQuery(t *testing.T) {
	if err := run([]string{"search", "tools"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunDemoSaveLibrary(t *testing.T) {
	root := t.TempDir()
	if err := run([]string{"demo", "save-library", "--root", root}); err != nil {
		t.Fatalf("run demo save-library: %v", err)
	}

	assertFile(t, filepath.Join(root, "libraries", "pdf_parser", "manifest.json"))
	assertFile(t, filepath.Join(root, "libraries", "pdf_parser", "versions", "v1", "source", "parser.py"))
	assertFile(t, filepath.Join(root, "registry", "metadata", "libraries", "pdf_parser.json"))
}

func TestRunSearchLibraries(t *testing.T) {
	root := t.TempDir()
	if err := run([]string{"demo", "save-library", "--root", root}); err != nil {
		t.Fatalf("run demo save-library: %v", err)
	}
	if err := run([]string{"search", "libraries", "--root", root, "--query", "pdf parser", "--limit", "1"}); err != nil {
		t.Fatalf("run search libraries: %v", err)
	}
}

func TestRunSearchLibrariesRequiresQuery(t *testing.T) {
	if err := run([]string{"search", "libraries"}); err == nil {
		t.Fatal("expected error")
	}
}
func TestRunUnknownCommand(t *testing.T) {
	if err := run([]string{"unknown"}); err == nil {
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
