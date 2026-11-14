package service

import (
	"context"
	"testing"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/retriever/local"
	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// TestExecutionOutcomeDrivesRetrievalRanking 闭合 AGENTS.md 要求的那条回路：
// 一个反复成功的工具，必须仅凭已记录的执行经验，排在一个语义完全相同
// 但反复失败的工具之前。
func TestExecutionOutcomeDrivesRetrievalRanking(t *testing.T) {
	root := t.TempDir()
	stores := filesystem.NewStores(root)
	ctx := context.Background()
	for _, id := range []string{"reliable_summarizer", "flaky_summarizer"} {
		tool := usageFixtureTool(id)
		if err := stores.Tools.Save(ctx, tool); err != nil {
			t.Fatalf("save %s: %v", id, err)
		}
		if err := stores.Registry.UpsertToolMetadata(ctx, tool); err != nil {
			t.Fatalf("upsert %s: %v", id, err)
		}
	}

	execution, err := NewExecutionService(ExecutionServiceConfig{
		ToolStore:    stores.Tools,
		Dependencies: stubResolver{},
		Policy:       allowAllPolicy{},
		Executor:     stubExecutor{},
		Recorder:     stores.Executions,
		Audit:        stores.Audit,
		Usage:        stores.Usage,
	})
	if err != nil {
		t.Fatalf("new execution service: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := execution.ExecuteTool(ctx, "reliable_summarizer", domain.TaskSpec{}, nil); err != nil {
			t.Fatalf("execute reliable: %v", err)
		}
	}
	for i := 0; i < 5; i++ {
		if _, err := execution.ExecuteTool(ctx, "flaky_summarizer", domain.TaskSpec{}, nil); err == nil {
			t.Fatal("expected flaky execution to fail")
		}
	}

	retrieval, err := NewRetrievalService(RetrievalServiceConfig{
		IntentExtractor: local.IntentExtractor{SemanticTopK: 10, FilterTopK: 10, RankTopK: 5, Filters: domain.RetrievalFilters{AllowedStatuses: domain.DefaultRetrievableStatuses()}},
		ToolRetriever:   local.NewToolRetriever(root),
		Ranker:          local.NewCapabilityRanker(),
	})
	if err != nil {
		t.Fatalf("new retrieval service: %v", err)
	}
	result, err := retrieval.Search(ctx, domain.TaskSpec{Intent: "summarize csv rows"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(result.Candidates))
	}
	if result.Candidates[0].Tool.Name != "reliable_summarizer" {
		t.Fatalf("top candidate = %q, want reliable_summarizer", result.Candidates[0].Tool.Name)
	}
	if result.Candidates[0].RankScore.SuccessRate != 1 {
		t.Fatalf("reliable success rate = %v, want 1", result.Candidates[0].RankScore.SuccessRate)
	}
	if result.Candidates[1].RankScore.SuccessRate != 0 {
		t.Fatalf("flaky success rate = %v, want 0", result.Candidates[1].RankScore.SuccessRate)
	}
	if result.Selected == nil || result.Selected.Tool.Name != "reliable_summarizer" {
		t.Fatalf("selected = %+v, want reliable_summarizer", result.Selected)
	}
}

// TestDefaultRetrievalHidesArchivedAndQuarantined 钉住 AGENTS.md 的规则：
// 即使 archived 和 quarantined 能力的匹配度高于任何活跃 Tool，
// 它们也必须退出默认检索集合。
func TestDefaultRetrievalHidesArchivedAndQuarantined(t *testing.T) {
	root := t.TempDir()
	registry := filesystem.NewRegistryStore(root)
	ctx := context.Background()
	for name, status := range map[string]domain.LifecycleStatus{
		"csv_summarizer_live":        domain.LifecycleApproved,
		"csv_summarizer_archived":    domain.LifecycleArchived,
		"csv_summarizer_quarantined": domain.LifecycleQuarantined,
	} {
		tool := usageFixtureTool(name)
		tool.LifecycleStatus = status
		if err := registry.UpsertToolMetadata(ctx, tool); err != nil {
			t.Fatalf("upsert %s: %v", name, err)
		}
	}

	search := func(filters domain.RetrievalFilters) []string {
		t.Helper()
		retrieval, err := NewRetrievalService(RetrievalServiceConfig{
			IntentExtractor: local.IntentExtractor{SemanticTopK: 10, FilterTopK: 10, RankTopK: 10, Filters: filters},
			ToolRetriever:   local.NewToolRetriever(root),
			Ranker:          local.NewCapabilityRanker(),
		})
		if err != nil {
			t.Fatalf("new retrieval service: %v", err)
		}
		result, err := retrieval.Search(ctx, domain.TaskSpec{Intent: "summarize csv"})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		names := make([]string, 0, len(result.Candidates))
		for _, candidate := range result.Candidates {
			names = append(names, candidate.Tool.Name)
		}
		return names
	}

	defaultNames := search(domain.RetrievalFilters{AllowedStatuses: domain.DefaultRetrievableStatuses()})
	if len(defaultNames) != 1 || defaultNames[0] != "csv_summarizer_live" {
		t.Fatalf("default retrieval = %v, want only csv_summarizer_live", defaultNames)
	}

	explicit := search(domain.RetrievalFilters{AllowedStatuses: []domain.LifecycleStatus{domain.LifecycleArchived}})
	if len(explicit) != 1 || explicit[0] != "csv_summarizer_archived" {
		t.Fatalf("explicit archived retrieval = %v, want only csv_summarizer_archived", explicit)
	}
}

type stubResolver struct{}

func (stubResolver) ResolveTool(context.Context, domain.Tool) (domain.DependencyGraph, error) {
	return domain.DependencyGraph{HealthStatus: domain.DependencyHealthy}, nil
}

func (stubResolver) ResolveLibrary(context.Context, domain.Library) (domain.DependencyGraph, error) {
	return domain.DependencyGraph{HealthStatus: domain.DependencyHealthy}, nil
}

type allowAllPolicy struct{}

func (allowAllPolicy) Evaluate(context.Context, domain.Tool, domain.TaskSpec) (domain.PolicyDecision, error) {
	return domain.PolicyDecision{Allowed: true}, nil
}

type stubExecutor struct{}

func (stubExecutor) Execute(_ context.Context, tool domain.Tool, _ map[string]any) (domain.ExecutionResult, error) {
	if tool.Name == "flaky_summarizer" {
		return domain.ExecutionResult{Success: false, Error: "synthetic failure"}, nil
	}
	return domain.ExecutionResult{Success: true, Output: `{"ok":true}`, Duration: 100 * time.Millisecond}, nil
}

func usageFixtureTool(name string) domain.Tool {
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	return domain.Tool{
		ID:              name,
		Name:            name,
		Description:     "Summarize CSV numeric columns into a short report.",
		Category:        "data_analysis",
		CurrentVersion:  "v1",
		LifecycleStatus: domain.LifecycleApproved,
		TrustLevel:      domain.TrustValidated,
		CreatedAt:       now,
		UpdatedAt:       now,
		Versions:        []domain.ToolVersion{{Version: "v1", Files: []domain.SourceFile{{Path: "main.py", Content: "print('{}')"}}, CreatedAt: now}},
		Manifest: domain.ToolManifest{
			Name:         name,
			Description:  "Summarize CSV numeric columns into a short report.",
			Category:     "data_analysis",
			Runtime:      "python",
			EntryPoint:   "main.py",
			Capabilities: []string{"csv", "summarize", "report"},
			Tags:         []string{"csv", "summary"},
		},
	}
}
