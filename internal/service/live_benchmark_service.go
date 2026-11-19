package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

// LiveBenchmarkService generates, validates and executes both comparison modes.
// Baseline never publishes tools. EvoTool learns on misses and reuses on hits.
type LiveBenchmarkService struct {
	Learning         *LearningService
	Retrieval        *RetrievalService
	Execution        *ExecutionService
	BaselineExecutor ports.ToolExecutor
	Policy           ports.PolicyEngine
}

// Run ignores suite Package/TokenUsage; expected outputs are independent oracles.
func (s LiveBenchmarkService) Run(ctx context.Context, suite domain.BenchmarkSuite) (domain.LiveBenchmarkReport, error) {
	report := domain.LiveBenchmarkReport{SuiteName: suite.Name, TokenUsageSource: "provider_reported"}
	if s.Learning == nil || s.Learning.Validator == nil || s.Learning.Generator == nil || s.Learning.Memory == nil || s.Retrieval == nil || s.Execution == nil || s.BaselineExecutor == nil || s.Policy == nil {
		return report, ErrMissingDependency
	}
	if suite.Name == "" || len(suite.Tasks) == 0 {
		return report, fmt.Errorf("nonempty benchmark suite required")
	}
	for _, task := range suite.Tasks {
		if task.Name == "" || strings.TrimSpace(task.Spec.Intent) == "" || task.Input == nil || task.Expected == nil {
			return report, fmt.Errorf("each live task requires Name, Spec.Intent, Input and Expected")
		}
	}
	for _, mode := range []domain.BenchmarkMode{domain.BenchmarkModeBaseline, domain.BenchmarkModeEvoTool} {
		summary := domain.LiveBenchmarkMode{BenchmarkResult: domain.BenchmarkResult{Mode: mode, SuiteName: suite.Name, TotalTasks: len(suite.Tasks)}}
		start := time.Now()
		generated := map[string]bool{}
		successes := 0
		for _, task := range suite.Tasks {
			if err := ctx.Err(); err != nil {
				return report, err
			}
			record := s.runTask(ctx, task, mode)
			if record.Reused {
				summary.ToolReusedCount++
			} else if record.Generated {
				summary.ToolGeneratedCount++
				name := record.ToolID
				if name != "" {
					if generated[name] {
						summary.DuplicateGenerationCount++
					}
					generated[name] = true
				}
				if record.Generation.UsageKnown {
					u := record.Generation.Usage
					summary.TotalTokenUsage.PromptTokens += u.PromptTokens
					summary.TotalTokenUsage.CompletionTokens += u.CompletionTokens
					summary.TotalTokenUsage.TotalTokens += u.TotalTokens
				} else {
					summary.MissingUsageCount++
				}
			}
			if record.PolicyBlocked {
				summary.SafetyPolicyBlockCount++
			}
			summary.LibraryReusedCount += record.LibraryReusedCount
			if record.Success {
				successes++
			}
			summary.Tasks = append(summary.Tasks, record)
		}
		n := len(suite.Tasks)
		summary.AverageLatency = time.Since(start) / time.Duration(n)
		summary.SuccessRate = float64(successes) / float64(n)
		summary.FailureRate = 1 - summary.SuccessRate
		u := summary.TotalTokenUsage
		summary.AverageTokenUsage = domain.TokenUsage{PromptTokens: u.PromptTokens / n, CompletionTokens: u.CompletionTokens / n, TotalTokens: u.TotalTokens / n}
		if mode == domain.BenchmarkModeBaseline {
			report.Baseline = summary
		} else {
			report.EvoTool = summary
		}
	}
	return report, nil
}

func (s LiveBenchmarkService) runTask(ctx context.Context, task domain.BenchmarkTask, mode domain.BenchmarkMode) domain.LiveBenchmarkTask {
	row := domain.LiveBenchmarkTask{Name: task.Name}
	fail := func(err error) domain.LiveBenchmarkTask {
		row.Error = err.Error()
		row.PolicyBlocked = errors.Is(err, ErrExecutionDenied)
		return row
	}
	var tool domain.Tool
	if mode == domain.BenchmarkModeEvoTool {
		found, err := s.Retrieval.Search(ctx, task.Spec)
		if err != nil {
			return fail(err)
		}
		if found.Selected != nil {
			row.Reused = true
			tool = found.Selected.Tool
		}
	}
	if !row.Reused {
		row.Generated = true
		g, err := s.Learning.GenerateCandidate(ctx, task.Spec)
		row.Generation = g.Generation
		row.ToolID = g.Manifest.Name
		if err != nil {
			return fail(err)
		}
		decision, policyErr := s.Policy.Evaluate(ctx, domain.Tool{Manifest: g.Manifest, CurrentVersion: "v1", LifecycleStatus: domain.LifecycleApproved, TrustLevel: domain.TrustValidated}, task.Spec)
		if policyErr != nil {
			return fail(policyErr)
		}
		if !decision.Allowed || decision.RequiresReview {
			return fail(ErrExecutionDenied)
		}
		if mode == domain.BenchmarkModeEvoTool {
			var r domain.ValidationResult
			tool, r, err = s.Learning.LearnSubmittedTool(ctx, g)
			_ = r
			if err != nil {
				return fail(err)
			}
		} else {
			if len(g.Manifest.LibraryRefs) > 0 {
				if s.Learning.Libraries == nil {
					return fail(fmt.Errorf("library resolver required"))
				}
				_, pins, err := s.Learning.Libraries.Resolve(ctx, g.Manifest.LibraryRefs)
				if err != nil {
					return fail(err)
				}
				g.Manifest.LibraryRefs = pins
			}
			valid, err := s.Learning.Validator.ValidateGeneratedTool(ctx, g)
			if err != nil {
				return fail(err)
			}
			if !valid.Valid {
				return fail(fmt.Errorf("baseline validation failed: %v", valid.Errors))
			}
			files := append(append([]domain.SourceFile{}, g.SourceFiles...), g.Tests...)
			if g.Readme != "" {
				files = append(files, domain.SourceFile{Path: "README.md", Content: g.Readme})
			}
			tool = domain.Tool{ID: g.Manifest.Name, Name: g.Manifest.Name, Manifest: g.Manifest, CurrentVersion: "v1", Versions: []domain.ToolVersion{{Version: "v1", Files: files}}, LifecycleStatus: domain.LifecycleApproved, TrustLevel: domain.TrustValidated}
		}
	}
	row.ToolID = tool.ID
	var result domain.ExecutionResult
	var err error
	if mode == domain.BenchmarkModeEvoTool {
		result, err = s.Execution.ExecuteTool(ctx, tool.ID, task.Spec, task.Input)
	} else {
		decision, e := s.Policy.Evaluate(ctx, tool, task.Spec)
		if e != nil {
			return fail(e)
		}
		if !decision.Allowed || decision.RequiresReview {
			return fail(ErrExecutionDenied)
		}
		result, err = s.BaselineExecutor.Execute(ctx, tool, task.Input)
	}
	row.Output = result.Output
	if err != nil {
		return fail(err)
	}
	if !result.Success {
		return fail(errors.New("executor reported failure"))
	}
	row.LibraryReusedCount = len(tool.Manifest.LibraryRefs)
	var actual any
	if err = json.Unmarshal([]byte(result.Output), &actual); err != nil {
		return fail(err)
	}
	expectedJSON, err := json.Marshal(task.Expected)
	if err != nil {
		return fail(err)
	}
	var expected any
	_ = json.Unmarshal(expectedJSON, &expected)
	if !reflect.DeepEqual(actual, expected) {
		return fail(fmt.Errorf("execution output differs from Expected"))
	}
	row.Success = true
	return row
}
