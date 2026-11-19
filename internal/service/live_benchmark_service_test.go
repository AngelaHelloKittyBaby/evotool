package service_test

import (
	"context"
	"errors"
	"testing"

	pythonexecutor "github.com/AngelaHelloKittyBaby/evotool/internal/adapter/executor/python"
	local "github.com/AngelaHelloKittyBaby/evotool/internal/adapter/retriever/local"
	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/service"
)

type measuredGenerator struct {
	tool  domain.GeneratedTool
	calls int
	fail  bool
}

func (g *measuredGenerator) Generate(context.Context, domain.TaskSpec) (domain.GeneratedTool, error) {
	g.calls++
	if g.fail {
		return g.tool, errors.New("model failure")
	}
	return g.tool, nil
}

func TestLiveBenchmarkMeasuresFailuresAndUsage(t *testing.T) {
	for _, tc := range []struct {
		name                                           string
		modelFail, testFail, wrongOutput, missingUsage bool
		rate                                           float64
		calls                                          int
	}{
		{name: "success", rate: 1, calls: 3},
		{name: "wrong output", wrongOutput: true, rate: 0, calls: 3},
		{name: "model failure", modelFail: true, rate: 0, calls: 4},
		{name: "validation failure", testFail: true, rate: 0, calls: 4},
		{name: "unknown usage", missingUsage: true, rate: 1, calls: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			stores := filesystem.NewStores(root)
			memory, err := service.NewCapabilityMemoryService(service.CapabilityMemoryServiceConfig{ToolStore: stores.Tools, LibraryStore: stores.Libraries, RegistryStore: stores.Registry, AuditLogger: stores.Audit})
			if err != nil {
				t.Fatal(err)
			}
			pkg := generatedFixture(t)
			pkg.Generation = domain.GenerationInfo{Model: "test", UsageKnown: !tc.missingUsage, Usage: domain.TokenUsage{PromptTokens: 8, CompletionTokens: 2, TotalTokens: 10}}
			gen := &measuredGenerator{tool: pkg, fail: tc.modelFail}
			sandbox := &checkingSandbox{fail: tc.testFail}
			learning := &service.LearningService{Memory: memory, Validator: service.ValidationService{Sandbox: sandbox}, Generator: gen}
			retrieval, err := service.NewRetrievalService(service.RetrievalServiceConfig{IntentExtractor: local.IntentExtractor{}, ToolRetriever: local.NewToolRetriever(root), Ranker: local.NewCapabilityRanker()})
			if err != nil {
				t.Fatal(err)
			}
			deps, err := service.NewDependencyService(service.DependencyServiceConfig{RegistryStore: stores.Registry, LibraryStore: stores.Libraries})
			if err != nil {
				t.Fatal(err)
			}
			policy, err := service.NewPolicyService(service.PolicyServiceConfig{ResourceLimits: domain.ResourceLimits{MaxRuntimeMs: 30000, MaxMemoryBytes: 268435456, MaxOutputBytes: 1048576}})
			if err != nil {
				t.Fatal(err)
			}
			executor := pythonexecutor.Executor{Sandbox: sandbox}
			execution, err := service.NewExecutionService(service.ExecutionServiceConfig{ToolStore: stores.Tools, Dependencies: deps, Policy: policy, Executor: executor, Recorder: stores.Executions, Audit: stores.Audit})
			if err != nil {
				t.Fatal(err)
			}
			bench := service.LiveBenchmarkService{Learning: learning, Retrieval: retrieval, Execution: execution, BaselineExecutor: executor, Policy: policy}
			expected := map[string]any{"count": 3, "total": 60, "average": 20}
			if tc.wrongOutput {
				expected["total"] = 999
			}
			task := domain.BenchmarkTask{Name: "task", Spec: domain.TaskSpec{Intent: "csv numeric summary"}, Input: map[string]any{"csv": "amount\n10\n20\n30\n", "column": "amount"}, Expected: expected, TokenUsage: domain.TokenUsage{TotalTokens: 999999}}
			report, err := bench.Run(context.Background(), domain.BenchmarkSuite{Name: "measured", Tasks: []domain.BenchmarkTask{task, task}})
			if err != nil {
				t.Fatal(err)
			}
			if gen.calls != tc.calls || report.Baseline.SuccessRate != tc.rate || report.EvoTool.SuccessRate != tc.rate {
				t.Fatalf("calls=%d report=%+v", gen.calls, report)
			}
			if tc.missingUsage {
				if report.EvoTool.MissingUsageCount != 1 || report.EvoTool.TotalTokenUsage.TotalTokens != 0 {
					t.Fatal("unknown usage treated as known")
				}
			} else if report.Baseline.TotalTokenUsage.TotalTokens != 20 {
				t.Fatal("provider usage lost or replaced by suite tokens")
			}
			if tc.modelFail || tc.testFail {
				items, _ := stores.Tools.List(context.Background())
				if len(items) != 0 {
					t.Fatal("failed generated tool saved")
				}
			}
		})
	}
}
