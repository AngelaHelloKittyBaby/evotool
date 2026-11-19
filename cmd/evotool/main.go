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
	case "learn", "execute", "generate":
		return runToolWorkflow(args[0], args[1:])
	case "search":
		return runSearch(args[1:])
	case "benchmark":
		return runBenchmark(args[1:])
	case "deps":
		return runDeps(args[1:])
	case "lifecycle":
		return runLifecycle(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
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

func runBenchmark(args []string) error {
	if len(args) == 0 {
		printBenchmarkUsage()
		return nil
	}

	switch args[0] {
	case "live":
		return runLiveBenchmark(args[1:])
	case "run":
		flags := flag.NewFlagSet("benchmark run", flag.ContinueOnError)
		root := flags.String("root", filesystem.DefaultRoot, "EvoTool memory root directory")
		suitePath := flags.String("suite", "", "benchmark suite JSON file")
		outPath := flags.String("out", "", "optional result JSON file")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("unexpected positional arguments")
		}
		if strings.TrimSpace(*suitePath) == "" {
			return fmt.Errorf("benchmark suite file is required")
		}
		return runBenchmarkSuite(*root, *suitePath, *outPath)
	case "help", "-h", "--help":
		printBenchmarkUsage()
		return nil
	default:
		return fmt.Errorf("unknown benchmark command %q", args[0])
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

func runLifecycle(args []string) error {
	if len(args) == 0 {
		printLifecycleUsage()
		return nil
	}

	switch args[0] {
	case "scan":
		flags := flag.NewFlagSet("lifecycle scan", flag.ContinueOnError)
		root := flags.String("root", filesystem.DefaultRoot, "EvoTool memory root directory")
		staleDays := flags.Int("stale-days", 90, "days without use before a Tool becomes stale")
		archiveDays := flags.Int("archive-days", 180, "days without use before a Tool becomes an archive candidate")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if *staleDays <= 0 {
			return fmt.Errorf("stale-days must be greater than zero")
		}
		if *archiveDays <= *staleDays {
			return fmt.Errorf("archive-days must be greater than stale-days")
		}
		return scanLifecycle(*root, *staleDays, *archiveDays)
	case "help", "-h", "--help":
		printLifecycleUsage()
		return nil
	default:
		return fmt.Errorf("unknown lifecycle command %q", args[0])
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

func searchTools(options searchToolOptions) error {
	retrieval, err := newRetrievalService(options.root, options)
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
		Intent:  options.query,
		Limit:   options.limit,
		Filters: domain.RetrievalFilters{AllowedStatuses: domain.DefaultRetrievableStatuses()},
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

func scanLifecycle(root string, staleDays int, archiveDays int) error {
	stores := filesystem.NewStores(root)
	lifecycle, err := service.NewLifecycleService(service.LifecycleServiceConfig{
		ToolStore: stores.Tools,
		Policy: domain.LifecyclePolicy{
			StaleAfter:   time.Duration(staleDays) * 24 * time.Hour,
			ArchiveAfter: time.Duration(archiveDays) * 24 * time.Hour,
		},
	})
	if err != nil {
		return fmt.Errorf("create lifecycle service: %w", err)
	}

	report, err := lifecycle.Scan(context.Background())
	if err != nil {
		return fmt.Errorf("scan lifecycle: %w", err)
	}
	printLifecycleScanReport(report, staleDays, archiveDays)
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

// newBenchmarkService 用 CLI 正在使用的同一套检索和策略实现装配基准路径，
// 这样报告出来的收益反映的是真实检索流水线，而不是基准专用的捷径。
func newBenchmarkService(root string) (*service.BenchmarkService, error) {
	memory, err := newCapabilityMemory(root)
	if err != nil {
		return nil, err
	}
	retrieval, err := newRetrievalService(root, searchToolOptions{runtime: "", limit: 3})
	if err != nil {
		return nil, err
	}
	policy, err := service.NewPolicyService(service.PolicyServiceConfig{
		ResourceLimits: domain.ResourceLimits{
			MaxRuntimeMs:   30000,
			MaxMemoryBytes: 268435456,
			MaxOutputBytes: 1048576,
		},
	})
	if err != nil {
		return nil, err
	}
	return service.NewBenchmarkService(service.BenchmarkServiceConfig{
		Retrieval: retrieval,
		Memory:    memory,
		Policy:    policy,
	})
}

func newRetrievalService(root string, options searchToolOptions) (*service.RetrievalService, error) {
	filters := domain.RetrievalFilters{
		Runtime:         options.runtime,
		AllowedStatuses: domain.DefaultRetrievableStatuses(),
	}
	if options.noNetwork {
		allowed := false
		filters.NetworkAllowed = &allowed
	}
	return service.NewRetrievalService(service.RetrievalServiceConfig{
		IntentExtractor: localretriever.IntentExtractor{
			SemanticTopK: 50,
			FilterTopK:   maxInt(options.limit*4, options.limit),
			RankTopK:     options.limit,
			Filters:      filters,
		},
		ToolRetriever: localretriever.NewToolRetriever(root),
		Ranker:        localretriever.NewCapabilityRanker(),
	})
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

func printLifecycleScanReport(report domain.LifecycleScanReport, staleDays int, archiveDays int) {
	fmt.Printf(
		"lifecycle scan: tools=%d candidates=%d stale_after=%dd archive_after=%dd\n",
		report.TotalTools,
		len(report.Candidates),
		staleDays,
		archiveDays,
	)
	if len(report.Candidates) == 0 {
		fmt.Println("candidates: none")
		fmt.Println("read-only scan: no files changed")
		return
	}

	fmt.Println("candidates:")
	for _, candidate := range report.Candidates {
		fmt.Printf(
			"- %s current=%s proposed=%s action=%s idle=%s last_activity=%s reason=%s\n",
			candidate.ToolID,
			candidate.CurrentStatus,
			candidate.ProposedStatus,
			candidate.Action,
			candidate.IdleFor.Round(time.Second),
			candidate.LastActivityAt.Format(time.RFC3339),
			candidate.Reason,
		)
	}
	fmt.Println("read-only scan: no files changed")
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

func printUsage() {
	fmt.Println(`EvoTool

Usage:
  evotool generate --task task.json --root .evotool [--image python:3.13-alpine]
  evotool benchmark live --suite live-suite.json --root dedicated-memory [--out report.json]
  evotool learn --package generated.json [--root .evotool] [--image python:3.13-alpine]
  evotool execute --id csv_analyzer --input input.json [--root .evotool]
  evotool benchmark run --suite benchmark.json [--root .evotool] [--out result.json]
  evotool search tools --query "csv numeric summary" [--root .evotool]
  evotool search libraries --query "csv numeric summary" [--root .evotool]
  evotool deps graph --id csv_numeric_summary [--root .evotool]
  evotool deps check --id csv_numeric_summary [--root .evotool]
  evotool lifecycle scan [--root .evotool] [--stale-days 90] [--archive-days 180]
  evotool help`)
}

func printSearchUsage() {
	fmt.Println(`EvoTool search commands

Usage:
  evotool search tools --query "csv numeric summary" [--root .evotool]
  evotool search libraries --query "csv statistics" [--root .evotool]
  evotool search tools "csv numeric summary" --category data_processing --runtime python`)
}

func printDepsUsage() {
	fmt.Println(`EvoTool dependency commands

Usage:
  evotool deps graph --id csv_numeric_summary [--root .evotool]
  evotool deps check --id csv_numeric_summary [--root .evotool]`)
}

func printLifecycleUsage() {
	fmt.Println(`EvoTool lifecycle commands

Usage:
  evotool lifecycle scan [--root .evotool] [--stale-days 90] [--archive-days 180]`)
}
