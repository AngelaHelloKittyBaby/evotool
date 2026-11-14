package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// BenchmarkServiceConfig defines the capabilities a benchmark run needs.
// Every dependency is required: a benchmark that silently skips retrieval or
// policy evaluation would report savings it never measured.
type BenchmarkServiceConfig struct {
	Retrieval *RetrievalService
	Memory    *CapabilityMemoryService
	Policy    *PolicyService
	Now       func() time.Time
}

// BenchmarkService compares tool generation cost with and without capability memory.
type BenchmarkService struct {
	retrieval *RetrievalService
	memory    *CapabilityMemoryService
	policy    *PolicyService
	now       func() time.Time
}

// NewBenchmarkService validates the benchmark dependencies.
func NewBenchmarkService(config BenchmarkServiceConfig) (*BenchmarkService, error) {
	if config.Retrieval == nil {
		return nil, fmt.Errorf("%w: retrieval service", ErrMissingDependency)
	}
	if config.Memory == nil {
		return nil, fmt.Errorf("%w: capability memory", ErrMissingDependency)
	}
	if config.Policy == nil {
		return nil, fmt.Errorf("%w: policy service", ErrMissingDependency)
	}
	now := config.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &BenchmarkService{retrieval: config.Retrieval, memory: config.Memory, policy: config.Policy, now: now}, nil
}

// Run executes every task under both modes and returns the comparison.
// The modes share one task list and one stored capability memory, so the only
// difference is whether an existing capability is searched for before generating.
// Capabilities learned during the run are not reused by later tasks in the same
// run: that would measure in-run learning, which this benchmark does not claim.
func (s *BenchmarkService) Run(ctx context.Context, suite domain.BenchmarkSuite) (domain.BenchmarkReport, error) {
	if err := checkContext(ctx); err != nil {
		return domain.BenchmarkReport{}, err
	}
	if strings.TrimSpace(suite.Name) == "" {
		return domain.BenchmarkReport{}, fmt.Errorf("benchmark suite name is required")
	}
	if len(suite.Tasks) == 0 {
		return domain.BenchmarkReport{}, fmt.Errorf("benchmark suite requires at least one task")
	}
	for _, task := range suite.Tasks {
		if strings.TrimSpace(task.Name) == "" {
			return domain.BenchmarkReport{}, fmt.Errorf("benchmark task name is required")
		}
		if strings.TrimSpace(task.Package.Manifest.Name) == "" {
			return domain.BenchmarkReport{}, fmt.Errorf("benchmark task %q requires a generated package name", task.Name)
		}
	}

	baseline, err := s.runMode(ctx, suite, domain.BenchmarkModeBaseline)
	if err != nil {
		return domain.BenchmarkReport{}, fmt.Errorf("baseline run: %w", err)
	}
	evotool, err := s.runMode(ctx, suite, domain.BenchmarkModeEvoTool)
	if err != nil {
		return domain.BenchmarkReport{}, fmt.Errorf("evotool run: %w", err)
	}
	return domain.BenchmarkReport{SuiteName: suite.Name, Baseline: baseline, EvoTool: evotool}, nil
}

// runMode replays the suite in one mode. Baseline always generates; EvoTool
// consults capability memory first and generates only when nothing reusable exists.
func (s *BenchmarkService) runMode(ctx context.Context, suite domain.BenchmarkSuite, mode domain.BenchmarkMode) (domain.BenchmarkResult, error) {
	result := domain.BenchmarkResult{Mode: mode, SuiteName: suite.Name, TotalTasks: len(suite.Tasks)}
	generated := map[string]struct{}{}
	var totalLatency time.Duration

	for _, task := range suite.Tasks {
		if err := checkContext(ctx); err != nil {
			return domain.BenchmarkResult{}, err
		}
		started := s.now()

		reused := false
		if mode == domain.BenchmarkModeEvoTool {
			outcome, err := s.assessReuse(ctx, task)
			if err != nil {
				return domain.BenchmarkResult{}, err
			}
			if outcome.matched && !outcome.healthy {
				result.DependencyFailureCount++
			}
			reused = outcome.healthy
		}

		if reused {
			result.ToolReusedCount++
		} else {
			result.ToolGeneratedCount++
			name := task.Package.Manifest.Name
			if _, seen := generated[name]; seen {
				result.DuplicateGenerationCount++
			}
			generated[name] = struct{}{}
			result.TotalTokenUsage.PromptTokens += task.TokenUsage.PromptTokens
			result.TotalTokenUsage.CompletionTokens += task.TokenUsage.CompletionTokens
			result.TotalTokenUsage.TotalTokens += task.TokenUsage.TotalTokens
		}

		// Library reuse and policy risk are properties of the package the Agent
		// would submit, so they are measurable in both modes.
		if reusedLibraries, err := s.countReusableLibraries(ctx, task.Package); err != nil {
			return domain.BenchmarkResult{}, err
		} else {
			result.LibraryReusedCount += reusedLibraries
		}
		if blocked, err := s.policyBlocked(ctx, task.Package); err != nil {
			return domain.BenchmarkResult{}, err
		} else if blocked {
			result.SafetyPolicyBlockCount++
		}

		totalLatency += s.now().Sub(started)
	}

	if result.TotalTasks > 0 {
		result.AverageLatency = totalLatency / time.Duration(result.TotalTasks)
		result.AverageTokenUsage = domain.TokenUsage{
			PromptTokens:     result.TotalTokenUsage.PromptTokens / result.TotalTasks,
			CompletionTokens: result.TotalTokenUsage.CompletionTokens / result.TotalTasks,
			TotalTokens:      result.TotalTokenUsage.TotalTokens / result.TotalTasks,
		}
	}
	// SuccessRate and FailureRate stay zero: this benchmark replays reuse
	// decisions, it does not execute the generated tools. Reporting 1.0 would
	// be an assertion nobody measured.
	return result, nil
}

type reuseOutcome struct {
	matched bool
	healthy bool
}

// assessReuse searches capability memory and verifies the match is actually
// runnable. A dependency check is part of the test: a matching Tool whose
// Libraries are missing is not reusable, so the caller must generate instead.
func (s *BenchmarkService) assessReuse(ctx context.Context, task domain.BenchmarkTask) (reuseOutcome, error) {
	found, err := s.retrieval.Search(ctx, task.Spec)
	if err != nil {
		return reuseOutcome{}, fmt.Errorf("retrieve for task %q: %w", task.Name, err)
	}
	if found.Selected == nil {
		return reuseOutcome{}, nil
	}
	toolID := found.Selected.Tool.ID
	if strings.TrimSpace(toolID) == "" {
		toolID = found.Selected.Tool.Name
	}
	check, err := s.memory.CheckToolDependencies(ctx, toolID)
	if err != nil {
		// No stored graph means the Tool was never registered as reusable;
		// treating that as healthy would claim a reuse execution would refuse.
		if errors.Is(err, fs.ErrNotExist) {
			return reuseOutcome{matched: true}, nil
		}
		return reuseOutcome{}, fmt.Errorf("check dependencies for %q: %w", toolID, err)
	}
	return reuseOutcome{matched: true, healthy: check.Healthy}, nil
}

// countReusableLibraries reports how many declared Library references already
// resolve against stored capability memory.
func (s *BenchmarkService) countReusableLibraries(ctx context.Context, pkg domain.GeneratedTool) (int, error) {
	refs := append(append([]domain.DependencyRef{}, pkg.Manifest.LibraryRefs...), pkg.Dependencies...)
	count := 0
	for _, ref := range refs {
		if ref.Kind != "" && ref.Kind != domain.DependencyLibrary {
			continue
		}
		id := ref.ID
		if strings.TrimSpace(id) == "" {
			id = ref.Name
		}
		if strings.TrimSpace(id) == "" {
			continue
		}
		if _, err := s.memory.GetLibrary(ctx, id); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return 0, fmt.Errorf("load library %q: %w", id, err)
		}
		count++
	}
	return count, nil
}

// policyBlocked reports whether the submitted package would be stopped before
// execution. It evaluates declared metadata only; no code runs here.
func (s *BenchmarkService) policyBlocked(ctx context.Context, pkg domain.GeneratedTool) (bool, error) {
	decision, err := s.policy.Evaluate(ctx, domain.Tool{
		Manifest:        pkg.Manifest,
		CurrentVersion:  "v1",
		LifecycleStatus: domain.LifecycleApproved,
		TrustLevel:      domain.TrustValidated,
	}, domain.TaskSpec{})
	if err != nil {
		return false, fmt.Errorf("evaluate policy for %q: %w", pkg.Manifest.Name, err)
	}
	return !decision.Allowed || decision.RequiresReview, nil
}
