package service

import (
	"context"
	"testing"
	"time"

	localretriever "github.com/AngelaHelloKittyBaby/evotool/internal/adapter/retriever/local"
	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

func TestBenchmarkWithEmptyMemoryShowsNoSavings(t *testing.T) {
	report, err := newBenchmarkFixture(t, false).service.Run(context.Background(), benchmarkTestSuite())
	if err != nil {
		t.Fatalf("run benchmark: %v", err)
	}
	if report.EvoTool.ToolReusedCount != 0 {
		t.Fatalf("reused %d tools with empty memory", report.EvoTool.ToolReusedCount)
	}
	if report.GenerationSaved() != 0 {
		t.Fatalf("claimed %d generation saved without any stored capability", report.GenerationSaved())
	}
	if report.TokenSaved().TotalTokens != 0 {
		t.Fatalf("claimed %d tokens saved without any stored capability", report.TokenSaved().TotalTokens)
	}
	if report.Baseline.ToolGeneratedCount != 3 || report.EvoTool.ToolGeneratedCount != 3 {
		t.Fatalf("generated baseline=%d evotool=%d, want 3 and 3", report.Baseline.ToolGeneratedCount, report.EvoTool.ToolGeneratedCount)
	}
}

func TestBenchmarkWithStoredToolSubstitutesGeneration(t *testing.T) {
	report, err := newBenchmarkFixture(t, true).service.Run(context.Background(), benchmarkTestSuite())
	if err != nil {
		t.Fatalf("run benchmark: %v", err)
	}
	if report.Baseline.ToolGeneratedCount != 3 {
		t.Fatalf("baseline generated = %d, want 3", report.Baseline.ToolGeneratedCount)
	}
	if report.EvoTool.ToolReusedCount != 3 {
		t.Fatalf("evotool reused = %d, want 3", report.EvoTool.ToolReusedCount)
	}
	if report.EvoTool.ToolGeneratedCount != 0 {
		t.Fatalf("evotool generated = %d, want 0", report.EvoTool.ToolGeneratedCount)
	}
	if report.GenerationSaved() != 3 {
		t.Fatalf("generation saved = %d, want 3", report.GenerationSaved())
	}
	// baseline 下三个任务各花 1200 token；全部复用的那一轮一个都不花。
	if report.Baseline.TotalTokenUsage.TotalTokens != 3600 {
		t.Fatalf("baseline tokens = %d, want 3600", report.Baseline.TotalTokenUsage.TotalTokens)
	}
	if report.TokenSaved().TotalTokens != 3600 {
		t.Fatalf("tokens saved = %d, want 3600", report.TokenSaved().TotalTokens)
	}
	// baseline 把同一个包生成了三次，因此其中两次算重复。
	if report.Baseline.DuplicateGenerationCount != 2 {
		t.Fatalf("baseline duplicate generations = %d, want 2", report.Baseline.DuplicateGenerationCount)
	}
	if report.DuplicateGenerationSaved() != 2 {
		t.Fatalf("duplicate generations saved = %d, want 2", report.DuplicateGenerationSaved())
	}
}

func TestBenchmarkDoesNotReuseUnhealthyDependencies(t *testing.T) {
	fixture := newBenchmarkFixture(t, true)
	// 已保存的 Tool 引用了一个不存在的 Library。
	graph := domain.DependencyGraph{
		Nodes: []domain.DependencyNode{
			{ID: "csv_numeric_summary", Name: "csv_numeric_summary", Kind: domain.DependencyTool, Version: "v1"},
			{ID: "csv_stats_core", Name: "csv_stats_core", Kind: domain.DependencyLibrary, Version: "v1"},
		},
		Edges:        []domain.DependencyEdge{{FromID: "csv_numeric_summary", ToID: "csv_stats_core", Kind: domain.DependencyLibrary}},
		HealthStatus: domain.DependencyHealthy,
	}
	if err := fixture.registry.SaveDependencyGraph(context.Background(), graph); err != nil {
		t.Fatalf("save graph: %v", err)
	}

	report, err := fixture.service.Run(context.Background(), benchmarkTestSuite())
	if err != nil {
		t.Fatalf("run benchmark: %v", err)
	}
	if report.EvoTool.ToolReusedCount != 0 {
		t.Fatalf("reused a Tool with a broken dependency: %+v", report.EvoTool)
	}
	if report.EvoTool.DependencyFailureCount != 3 {
		t.Fatalf("dependency failures = %d, want 3", report.EvoTool.DependencyFailureCount)
	}
	if report.GenerationSaved() != 0 {
		t.Fatalf("claimed savings despite broken dependencies: %d", report.GenerationSaved())
	}
}

func TestBenchmarkCountsReusableLibraries(t *testing.T) {
	fixture := newBenchmarkFixture(t, true)
	if err := fixture.libraries.Save(context.Background(), domain.Library{
		ID: "csv_stats_core", Name: "csv_stats_core", CurrentVersion: "v1",
		LifecycleStatus: domain.LifecycleApproved, TrustLevel: domain.TrustValidated,
	}); err != nil {
		t.Fatalf("save library: %v", err)
	}

	suite := benchmarkTestSuite()
	for i := range suite.Tasks {
		suite.Tasks[i].Package.Manifest.LibraryRefs = []domain.DependencyRef{
			{ID: "csv_stats_core", Name: "csv_stats_core", Kind: domain.DependencyLibrary, Version: "v1"},
			{ID: "missing_library", Name: "missing_library", Kind: domain.DependencyLibrary, Version: "v1"},
		}
	}

	report, err := fixture.service.Run(context.Background(), suite)
	if err != nil {
		t.Fatalf("run benchmark: %v", err)
	}
	// 声明的两个 Library 中有一个能解析成功，三个任务各算一次。
	if report.Baseline.LibraryReusedCount != 3 {
		t.Fatalf("library reused = %d, want 3", report.Baseline.LibraryReusedCount)
	}
}

func TestBenchmarkCountsPolicyBlocks(t *testing.T) {
	fixture := newBenchmarkFixture(t, false)
	suite := benchmarkTestSuite()
	suite.Tasks = suite.Tasks[:1]
	suite.Tasks[0].Package.Manifest.Permissions = domain.PermissionSet{Network: domain.NetworkPolicyFull}

	report, err := fixture.service.Run(context.Background(), suite)
	if err != nil {
		t.Fatalf("run benchmark: %v", err)
	}
	if report.Baseline.SafetyPolicyBlockCount != 1 || report.EvoTool.SafetyPolicyBlockCount != 1 {
		t.Fatalf("policy blocks baseline=%d evotool=%d, want 1 and 1",
			report.Baseline.SafetyPolicyBlockCount, report.EvoTool.SafetyPolicyBlockCount)
	}
}

func TestBenchmarkRejectsInvalidSuites(t *testing.T) {
	fixture := newBenchmarkFixture(t, false)
	for _, tt := range []struct {
		name  string
		suite domain.BenchmarkSuite
	}{
		{"missing name", domain.BenchmarkSuite{Tasks: benchmarkTestSuite().Tasks}},
		{"no tasks", domain.BenchmarkSuite{Name: "empty"}},
		{"task without name", domain.BenchmarkSuite{Name: "s", Tasks: []domain.BenchmarkTask{{Package: benchmarkTestSuite().Tasks[0].Package}}}},
		{"task without package", domain.BenchmarkSuite{Name: "s", Tasks: []domain.BenchmarkTask{{Name: "t"}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := fixture.service.Run(context.Background(), tt.suite); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestBenchmarkRequiredDependencies(t *testing.T) {
	for i := 0; i < 3; i++ {
		config := BenchmarkServiceConfig{}
		switch i {
		case 0:
			config.Memory = &CapabilityMemoryService{}
			config.Policy = &PolicyService{}
		case 1:
			config.Retrieval = &RetrievalService{}
			config.Policy = &PolicyService{}
		case 2:
			config.Retrieval = &RetrievalService{}
			config.Memory = &CapabilityMemoryService{}
		}
		if _, err := NewBenchmarkService(config); err == nil {
			t.Fatalf("dependency %d: expected missing dependency error", i)
		}
	}
}

type benchmarkFixture struct {
	service   *BenchmarkService
	registry  *filesystem.RegistryStore
	libraries *filesystem.LibraryStore
}

// newBenchmarkFixture 基于真实的文件系统能力记忆构建基准，
// 使检索、依赖检查和策略都走生产代码。
func newBenchmarkFixture(t *testing.T, seedStoredTool bool) benchmarkFixture {
	t.Helper()
	root := t.TempDir()
	stores := filesystem.NewStores(root)
	ctx := context.Background()

	if seedStoredTool {
		tool := benchmarkFixtureTool()
		if err := stores.Tools.Save(ctx, tool); err != nil {
			t.Fatalf("save tool: %v", err)
		}
		if err := stores.Registry.UpsertToolMetadata(ctx, tool); err != nil {
			t.Fatalf("upsert tool metadata: %v", err)
		}
		if err := stores.Registry.SaveDependencyGraph(ctx, domain.DependencyGraph{
			Nodes:        []domain.DependencyNode{{ID: tool.ID, Name: tool.Name, Kind: domain.DependencyTool, Version: "v1"}},
			HealthStatus: domain.DependencyHealthy,
		}); err != nil {
			t.Fatalf("save dependency graph: %v", err)
		}
	}

	memory, err := NewCapabilityMemoryService(CapabilityMemoryServiceConfig{
		ToolStore:     stores.Tools,
		LibraryStore:  stores.Libraries,
		RegistryStore: stores.Registry,
	})
	if err != nil {
		t.Fatalf("new capability memory: %v", err)
	}
	retrieval, err := NewRetrievalService(RetrievalServiceConfig{
		IntentExtractor: localretriever.IntentExtractor{
			SemanticTopK: 50,
			FilterTopK:   10,
			RankTopK:     3,
			Filters:      domain.RetrievalFilters{AllowedStatuses: domain.DefaultRetrievableStatuses()},
		},
		ToolRetriever: localretriever.NewToolRetriever(root),
		Ranker:        localretriever.NewCapabilityRanker(),
	})
	if err != nil {
		t.Fatalf("new retrieval service: %v", err)
	}
	policy, err := NewPolicyService(PolicyServiceConfig{ResourceLimits: domain.ResourceLimits{
		MaxRuntimeMs: 30000, MaxMemoryBytes: 268435456, MaxOutputBytes: 1048576,
	}})
	if err != nil {
		t.Fatalf("new policy service: %v", err)
	}
	service, err := NewBenchmarkService(BenchmarkServiceConfig{
		Retrieval: retrieval,
		Memory:    memory,
		Policy:    policy,
		Now:       func() time.Time { return time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new benchmark service: %v", err)
	}
	return benchmarkFixture{service: service, registry: stores.Registry, libraries: stores.Libraries}
}

func benchmarkFixtureTool() domain.Tool {
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	return domain.Tool{
		ID:              "csv_numeric_summary",
		Name:            "csv_numeric_summary",
		Description:     "Analyze CSV numeric values and return row count, total and average",
		Category:        "data_processing",
		CurrentVersion:  "v1",
		LifecycleStatus: domain.LifecycleApproved,
		TrustLevel:      domain.TrustValidated,
		CreatedAt:       now,
		UpdatedAt:       now,
		Manifest: domain.ToolManifest{
			Name:         "csv_numeric_summary",
			Description:  "Analyze CSV numeric values and return row count, total and average",
			Category:     "data_processing",
			Runtime:      "python",
			EntryPoint:   "main.py",
			Capabilities: []string{"csv_analysis", "numeric_summary"},
			Tags:         []string{"csv", "numeric", "summary", "report"},
			Permissions:  domain.PermissionSet{Network: domain.NetworkPolicyNone},
			ResourceLimits: domain.ResourceLimits{
				MaxRuntimeMs: 30000, MaxMemoryBytes: 67108864, MaxOutputBytes: 65536,
			},
		},
	}
}

// benchmarkTestSuite 对应 examples/benchmark/csv-reporting-suite.json：
// 三个都需要同一份数值 CSV 统计的任务。
func benchmarkTestSuite() domain.BenchmarkSuite {
	base := domain.GeneratedTool{
		Manifest: domain.ToolManifest{
			Name:         "csv_numeric_summary",
			Description:  "Analyze CSV numeric values and return row count, total and average",
			Category:     "data_processing",
			Runtime:      "python",
			EntryPoint:   "main.py",
			Capabilities: []string{"csv_analysis", "numeric_summary"},
			Tags:         []string{"csv", "numeric", "summary", "report"},
			Permissions:  domain.PermissionSet{Network: domain.NetworkPolicyNone},
			ResourceLimits: domain.ResourceLimits{
				MaxRuntimeMs: 30000, MaxMemoryBytes: 67108864, MaxOutputBytes: 65536,
			},
		},
		GenerationReason: "agent needs a reusable numeric CSV summary",
	}
	tasks := make([]domain.BenchmarkTask, 0, 3)
	for _, intent := range []string{
		"Summarize the amount column of a monthly revenue CSV",
		"Summarize the amount column of a quarterly revenue CSV",
		"Summarize the amount column of an hourly sales CSV",
	} {
		tasks = append(tasks, domain.BenchmarkTask{
			Name: intent,
			Spec: domain.TaskSpec{
				Intent:         intent,
				InputSummary:   "csv text with an amount column",
				ExpectedOutput: "count, total and average of the amount column",
				Domain:         "data_processing",
			},
			Package:    base,
			TokenUsage: domain.TokenUsage{PromptTokens: 900, CompletionTokens: 300, TotalTokens: 1200},
			Input:      map[string]any{"csv": "amount\n10\n20\n30\n", "column": "amount"},
		})
	}
	return domain.BenchmarkSuite{Name: "csv_reporting_reuse", Tasks: tasks}
}
