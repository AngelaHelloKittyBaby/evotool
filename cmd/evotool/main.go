package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	localretriever "github.com/AngelaHelloKittyBaby/evotool/internal/adapter/retriever/local"
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
	case "search":
		return runSearch(args[1:])
	case "deps":
		return runDeps(args[1:])
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
	case "save-library":
		flags := flag.NewFlagSet("demo save-library", flag.ContinueOnError)
		root := flags.String("root", filesystem.DefaultRoot, "EvoTool memory root directory")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		return saveDemoLibrary(*root)
	case "link-tool-library":
		flags := flag.NewFlagSet("demo link-tool-library", flag.ContinueOnError)
		root := flags.String("root", filesystem.DefaultRoot, "EvoTool memory root directory")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		return linkDemoToolLibrary(*root)
	case "help", "-h", "--help":
		printDemoUsage()
		return nil
	default:
		return fmt.Errorf("unknown demo command %q", args[0])
	}
}

func runSearch(args []string) error {
	if len(args) == 0 {
		printSearchUsage()
		return nil
	}

	switch args[0] {
	case "tools":
		return runSearchTools(args[1:])
	case "libraries":
		return runSearchLibraries(args[1:])
	case "help", "-h", "--help":
		printSearchUsage()
		return nil
	default:
		return fmt.Errorf("unknown search command %q", args[0])
	}
}

func runDeps(args []string) error {
	if len(args) == 0 {
		printDepsUsage()
		return nil
	}

	switch args[0] {
	case "graph":
		flags := flag.NewFlagSet("deps graph", flag.ContinueOnError)
		root := flags.String("root", filesystem.DefaultRoot, "EvoTool memory root directory")
		id := flags.String("id", "", "root capability id")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if strings.TrimSpace(*id) == "" {
			return fmt.Errorf("dependency graph id is required")
		}
		return showDependencyGraph(*root, *id)
	case "check":
		flags := flag.NewFlagSet("deps check", flag.ContinueOnError)
		root := flags.String("root", filesystem.DefaultRoot, "EvoTool memory root directory")
		id := flags.String("id", "", "root capability id")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if strings.TrimSpace(*id) == "" {
			return fmt.Errorf("dependency check id is required")
		}
		return checkDependencies(*root, *id)
	case "help", "-h", "--help":
		printDepsUsage()
		return nil
	default:
		return fmt.Errorf("unknown deps command %q", args[0])
	}
}

type searchToolOptions struct {
	root      string
	query     string
	category  string
	runtime   string
	input     string
	output    string
	limit     int
	noNetwork bool
}

func runSearchTools(args []string) error {
	flags := flag.NewFlagSet("search tools", flag.ContinueOnError)
	root := flags.String("root", filesystem.DefaultRoot, "EvoTool memory root directory")
	query := flags.String("query", "", "search query")
	category := flags.String("category", "", "required tool category")
	runtime := flags.String("runtime", "", "required runtime")
	input := flags.String("input", "", "required input type or name")
	output := flags.String("output", "", "required output type or name")
	limit := flags.Int("limit", 3, "maximum candidates to print")
	noNetwork := flags.Bool("no-network", false, "exclude tools that require network access")
	if err := flags.Parse(args); err != nil {
		return err
	}

	searchQuery := normalizedQuery(*query, flags.Args())
	if searchQuery == "" {
		return fmt.Errorf("search query is required")
	}
	if *limit <= 0 {
		return fmt.Errorf("limit must be greater than zero")
	}

	return searchTools(searchToolOptions{
		root:      *root,
		query:     searchQuery,
		category:  *category,
		runtime:   *runtime,
		input:     *input,
		output:    *output,
		limit:     *limit,
		noNetwork: *noNetwork,
	})
}

func runSearchLibraries(args []string) error {
	flags := flag.NewFlagSet("search libraries", flag.ContinueOnError)
	root := flags.String("root", filesystem.DefaultRoot, "EvoTool memory root directory")
	query := flags.String("query", "", "library search query")
	category := flags.String("category", "", "required library category")
	runtime := flags.String("runtime", "", "required runtime")
	limit := flags.Int("limit", 5, "maximum libraries to print")
	noNetwork := flags.Bool("no-network", false, "exclude libraries that require network access")
	if err := flags.Parse(args); err != nil {
		return err
	}

	searchQuery := normalizedQuery(*query, flags.Args())
	if searchQuery == "" {
		return fmt.Errorf("search query is required")
	}
	if *limit <= 0 {
		return fmt.Errorf("limit must be greater than zero")
	}

	return searchLibraries(searchToolOptions{
		root:      *root,
		query:     searchQuery,
		category:  *category,
		runtime:   *runtime,
		limit:     *limit,
		noNetwork: *noNetwork,
	})
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

func saveDemoLibrary(root string) error {
	memory, err := newCapabilityMemory(root)
	if err != nil {
		return err
	}

	library := demoPDFParserLibrary()
	if err := memory.SaveLibrary(context.Background(), library); err != nil {
		return fmt.Errorf("save demo library: %w", err)
	}

	fmt.Printf("saved demo library %q into %s\n", library.Name, root)
	return nil
}

func linkDemoToolLibrary(root string) error {
	memory, err := newCapabilityMemory(root)
	if err != nil {
		return err
	}

	ctx := context.Background()
	tool := demoPDFToExcelTool()
	library := demoPDFParserLibrary()
	if err := memory.SaveTool(ctx, tool); err != nil {
		return fmt.Errorf("save demo tool: %w", err)
	}
	if err := memory.SaveLibrary(ctx, library); err != nil {
		return fmt.Errorf("save demo library: %w", err)
	}
	graph, err := memory.LinkToolLibraries(ctx, tool, []domain.Library{library})
	if err != nil {
		return fmt.Errorf("link demo dependency: %w", err)
	}
	printDependencyGraph(graph)
	return nil
}

func searchTools(options searchToolOptions) error {
	filters := domain.RetrievalFilters{Runtime: options.runtime}
	if options.noNetwork {
		allowed := false
		filters.NetworkAllowed = &allowed
	}

	retrieval, err := service.NewRetrievalService(service.RetrievalServiceConfig{
		IntentExtractor: localretriever.IntentExtractor{
			SemanticTopK: 50,
			FilterTopK:   maxInt(options.limit*4, options.limit),
			RankTopK:     options.limit,
			Filters:      filters,
		},
		ToolRetriever: localretriever.NewToolRetriever(options.root),
		Ranker:        localretriever.NewCapabilityRanker(),
	})
	if err != nil {
		return fmt.Errorf("create retrieval service: %w", err)
	}

	metadata := map[string]string{}
	setMetadata(metadata, "category", options.category)
	setMetadata(metadata, "runtime", options.runtime)
	setMetadata(metadata, "input", options.input)
	setMetadata(metadata, "output", options.output)

	result, err := retrieval.Search(context.Background(), domain.TaskSpec{Intent: options.query, Metadata: metadata})
	if err != nil {
		return fmt.Errorf("search tools: %w", err)
	}
	printSearchResults(result)
	return nil
}

func searchLibraries(options searchToolOptions) error {
	memory, err := newCapabilityMemory(options.root)
	if err != nil {
		return err
	}

	query := domain.RetrievalQuery{
		Intent: options.query,
		Limit:  options.limit,
	}
	if options.category != "" {
		query.Categories = []string{options.category}
	}
	if options.runtime != "" {
		query.Filters.Runtime = options.runtime
	}
	if options.noNetwork {
		allowed := false
		query.Filters.NetworkAllowed = &allowed
	}

	candidates, err := memory.SearchLibraries(context.Background(), query)
	if err != nil {
		return err
	}
	printLibraryResults(candidates)
	return nil
}

func showDependencyGraph(root string, id string) error {
	memory, err := newCapabilityMemory(root)
	if err != nil {
		return err
	}
	graph, err := memory.GetDependencyGraph(context.Background(), id)
	if err != nil {
		return err
	}
	printDependencyGraph(graph)
	return nil
}

func checkDependencies(root string, id string) error {
	memory, err := newCapabilityMemory(root)
	if err != nil {
		return err
	}
	result, err := memory.CheckToolDependencies(context.Background(), id)
	if err != nil {
		return err
	}
	printDependencyCheckResult(result)
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
		LibraryRetriever:  localretriever.NewLibraryRetriever(root),
	})
	if err != nil {
		return nil, fmt.Errorf("create capability memory: %w", err)
	}
	return memory, nil
}

func printSearchResults(result domain.RetrievalResult) {
	if len(result.Candidates) == 0 {
		fmt.Println("no matching tools found")
		return
	}
	if result.Selected != nil {
		fmt.Printf("selected: %s\n", displayToolName(result.Selected.Tool))
	}
	fmt.Println("candidates:")
	for index, candidate := range result.Candidates {
		fmt.Printf(
			"%d. %s score=%.3f semantic=%.3f category=%s runtime=%s reason=%s\n",
			index+1,
			displayToolName(candidate.Tool),
			candidate.Score,
			candidate.RankScore.SemanticSimilarity,
			displayToolCategory(candidate.Tool),
			displayToolRuntime(candidate.Tool),
			candidate.MatchReason,
		)
	}
}

func printLibraryResults(candidates []domain.LibraryCandidate) {
	if len(candidates) == 0 {
		fmt.Println("no matching libraries found")
		return
	}
	fmt.Println("libraries:")
	for index, candidate := range candidates {
		fmt.Printf(
			"%d. %s score=%.3f compatible=%t category=%s runtime=%s reason=%s\n",
			index+1,
			displayLibraryName(candidate.Library),
			candidate.Score,
			candidate.Compatible,
			displayLibraryCategory(candidate.Library),
			displayLibraryRuntime(candidate.Library),
			candidate.Reason,
		)
	}
}

func printDependencyGraph(graph domain.DependencyGraph) {
	if len(graph.Nodes) == 0 {
		fmt.Println("dependency graph is empty")
		return
	}
	fmt.Printf("dependency graph: %s health=%s\n", graph.Nodes[0].ID, graph.HealthStatus)
	fmt.Println("nodes:")
	for _, node := range graph.Nodes {
		fmt.Printf("- %s %s version=%s\n", node.Kind, node.ID, node.Version)
	}
	if len(graph.Edges) == 0 {
		fmt.Println("edges: none")
		return
	}
	fmt.Println("edges:")
	for _, edge := range graph.Edges {
		fmt.Printf("- %s -> %s (%s)\n", edge.FromID, edge.ToID, edge.Kind)
	}
}

func printDependencyCheckResult(result domain.DependencyCheckResult) {
	fmt.Printf("dependency health: %s healthy=%t\n", result.Graph.HealthStatus, result.Healthy)
	if len(result.Issues) == 0 {
		fmt.Println("issues: none")
		return
	}
	fmt.Println("issues:")
	for _, issue := range result.Issues {
		fmt.Printf("- %s %s status=%s %s\n", issue.Kind, issue.DependencyID, issue.Status, issue.Message)
	}
}

func displayToolName(tool domain.Tool) string {
	if strings.TrimSpace(tool.Name) != "" {
		return tool.Name
	}
	return tool.ID
}

func displayToolCategory(tool domain.Tool) string {
	if strings.TrimSpace(tool.Category) != "" {
		return tool.Category
	}
	return tool.Manifest.Category
}

func displayToolRuntime(tool domain.Tool) string {
	if strings.TrimSpace(tool.Manifest.Runtime) != "" {
		return tool.Manifest.Runtime
	}
	return tool.Manifest.Environment.Runtime
}

func displayLibraryName(library domain.Library) string {
	if strings.TrimSpace(library.Name) != "" {
		return library.Name
	}
	return library.ID
}

func displayLibraryCategory(library domain.Library) string {
	if strings.TrimSpace(library.Category) != "" {
		return library.Category
	}
	return library.Manifest.Category
}

func displayLibraryRuntime(library domain.Library) string {
	return library.Manifest.Runtime
}

func normalizedQuery(query string, args []string) string {
	query = strings.TrimSpace(query)
	if query != "" {
		return query
	}
	return strings.TrimSpace(strings.Join(args, " "))
}

func setMetadata(metadata map[string]string, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		metadata[key] = value
	}
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func demoPDFParserLibrary() domain.Library {
	now := time.Now().UTC()
	return domain.Library{
		ID:              "pdf_parser",
		Name:            "pdf_parser",
		Description:     "Parse PDF documents and extract structured tables.",
		Category:        "document_processing",
		CurrentVersion:  "v1",
		LifecycleStatus: domain.LifecycleApproved,
		TrustLevel:      domain.TrustValidated,
		CreatedAt:       now,
		UpdatedAt:       now,
		Manifest: domain.LibraryManifest{
			Name:        "pdf_parser",
			Description: "Reusable PDF parsing and table extraction primitives.",
			Category:    "document_processing",
			Runtime:     "python",
			Exports: []domain.LibraryExport{
				{Name: "parse_pdf", Kind: "function", Description: "Parse a PDF document."},
				{Name: "extract_tables", Kind: "function", Description: "Extract tables from a parsed PDF."},
			},
			Tags: []string{"pdf", "parser", "table", "document"},
		},
		Versions: []domain.LibraryVersion{
			{
				Version:   "v1",
				CreatedAt: now,
				Files: []domain.SourceFile{
					{Path: "source/parser.py", Content: demoPDFParserSource()},
					{Path: "README.md", Content: demoLibraryReadme()},
				},
			},
		},
	}
}

func demoPDFParserSource() string {
	return `def parse_pdf(path):
    return {"path": path, "pages": []}


def extract_tables(parsed_pdf):
    return parsed_pdf.get("tables", [])
`
}

func demoLibraryReadme() string {
	return `# pdf_parser

Reusable PDF parsing library for generated EvoTool tools.
`
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
  evotool demo save-library [--root .evotool]
  evotool demo link-tool-library [--root .evotool]
  evotool search tools --query "pdf excel" [--root .evotool]
  evotool search libraries --query "pdf parser" [--root .evotool]
  evotool deps graph --id pdf_to_excel [--root .evotool]
  evotool deps check --id pdf_to_excel [--root .evotool]
  evotool help`)
}

func printDemoUsage() {
	fmt.Println(`EvoTool demo commands

Usage:
  evotool demo save-tool [--root .evotool]
  evotool demo save-library [--root .evotool]
  evotool demo link-tool-library [--root .evotool]`)
}

func printSearchUsage() {
	fmt.Println(`EvoTool search commands

Usage:
  evotool search tools --query "pdf excel" [--root .evotool]
  evotool search libraries --query "pdf parser" [--root .evotool]
  evotool search tools "pdf excel" --category document_processing --runtime python`)
}

func printDepsUsage() {
	fmt.Println(`EvoTool dependency commands

Usage:
  evotool deps graph --id pdf_to_excel [--root .evotool]
  evotool deps check --id pdf_to_excel [--root .evotool]`)
}
