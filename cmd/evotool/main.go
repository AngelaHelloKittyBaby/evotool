package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/service"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "demo":
		return runDemo(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runDemo(args []string) error {
	if len(args) == 0 {
		printDemoUsage()
		return nil
	}

	switch args[0] {
	case "save-tool":
		flags := flag.NewFlagSet("demo save-tool", flag.ContinueOnError)
		root := flags.String("root", filesystem.DefaultRoot, "EvoTool memory root directory")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		return saveDemoTool(*root)
	case "help", "-h", "--help":
		printDemoUsage()
		return nil
	default:
		return fmt.Errorf("unknown demo command %q", args[0])
	}
}

func saveDemoTool(root string) error {
	memory, err := newCapabilityMemory(root)
	if err != nil {
		return err
	}

	tool := demoPDFToExcelTool()
	if err := memory.SaveTool(context.Background(), tool); err != nil {
		return fmt.Errorf("save demo tool: %w", err)
	}

	fmt.Printf("saved demo tool %q into %s\n", tool.Name, root)
	return nil
}

func newCapabilityMemory(root string) (*service.CapabilityMemoryService, error) {
	stores := filesystem.NewStores(root)
	memory, err := service.NewCapabilityMemoryService(service.CapabilityMemoryServiceConfig{
		ToolStore:         stores.Tools,
		LibraryStore:      stores.Libraries,
		RegistryStore:     stores.Registry,
		ExecutionRecorder: stores.Executions,
		AuditLogger:       stores.Audit,
	})
	if err != nil {
		return nil, fmt.Errorf("create capability memory: %w", err)
	}
	return memory, nil
}

func demoPDFToExcelTool() domain.Tool {
	now := time.Now().UTC()
	return domain.Tool{
		ID:              "pdf_to_excel",
		Name:            "pdf_to_excel",
		Description:     "Demo tool that represents converting PDF tables to Excel.",
		Category:        "document_processing",
		CurrentVersion:  "v1",
		LifecycleStatus: domain.LifecycleApproved,
		TrustLevel:      domain.TrustValidated,
		CreatedAt:       now,
		UpdatedAt:       now,
		Manifest: domain.ToolManifest{
			Name:         "pdf_to_excel",
			Description:  "Convert PDF tables to Excel format. This demo implementation only returns a derived xlsx path.",
			Category:     "document_processing",
			Runtime:      "python",
			EntryPoint:   "main.py",
			Capabilities: []string{"pdf_table_extraction", "xlsx_generation"},
			Tags:         []string{"pdf", "table", "excel", "document"},
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"file"},
				"properties": map[string]any{
					"file": map[string]any{"type": "string", "description": "PDF file path"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"xlsx_path": map[string]any{"type": "string", "description": "Generated xlsx path"},
				},
			},
			Examples: []domain.ManifestExample{
				{Input: `{"file":"report.pdf"}`, Output: `{"xlsx_path":"report.xlsx"}`, Description: "Convert report.pdf to report.xlsx"},
			},
			Environment: domain.EnvironmentSpec{Runtime: "python", Network: false},
			Permissions: domain.PermissionSet{Network: domain.NetworkPolicyNone},
			ResourceLimits: domain.ResourceLimits{
				MaxRuntimeMs:   5000,
				MaxMemoryBytes: 128 * 1024 * 1024,
				MaxOutputBytes: 1024 * 1024,
			},
		},
		Versions: []domain.ToolVersion{
			{
				Version:   "v1",
				CreatedAt: now,
				Files: []domain.SourceFile{
					{Path: "main.py", Content: demoPythonToolSource()},
					{Path: "README.md", Content: demoToolReadme()},
					{Path: "tests/test_main.py", Content: demoPythonToolTest()},
				},
			},
		},
	}
}

func demoPythonToolSource() string {
	return `#!/usr/bin/env python3
import json
import os
import sys


def main():
    payload = json.load(sys.stdin)
    pdf_path = payload.get("file", "input.pdf")
    base, _ = os.path.splitext(pdf_path)
    print(json.dumps({"xlsx_path": base + ".xlsx"}))


if __name__ == "__main__":
    main()
`
}

func demoToolReadme() string {
	return `# pdf_to_excel

Demo generated tool for EvoTool.

This demo does not parse real PDFs yet. It exists to prove that EvoTool can save a generated tool package into capability memory.
`
}

func demoPythonToolTest() string {
	return `import json
import subprocess
import sys


def test_demo_tool_outputs_xlsx_path():
    proc = subprocess.run(
        [sys.executable, "main.py"],
        input=json.dumps({"file": "report.pdf"}),
        text=True,
        capture_output=True,
        check=True,
    )
    assert json.loads(proc.stdout)["xlsx_path"] == "report.xlsx"
`
}

func printUsage() {
	fmt.Println(`EvoTool

Usage:
  evotool demo save-tool [--root .evotool]
  evotool help`)
}

func printDemoUsage() {
	fmt.Println(`EvoTool demo commands

Usage:
  evotool demo save-tool [--root .evotool]`)
}
