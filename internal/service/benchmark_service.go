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

// BenchmarkServiceConfig 定义一次基准运行所需的能力。
// 每项依赖都是必需的：如果基准悄悄跳过检索或策略评估，
// 它就会报告一个自己从未测量过的收益。
type BenchmarkServiceConfig struct {
	Retrieval *RetrievalService
	Memory    *CapabilityMemoryService
	Policy    *PolicyService
	Now       func() time.Time
}

// BenchmarkService 对比有能力和没有能力记忆时的工具生成成本。
type BenchmarkService struct {
	retrieval *RetrievalService
	memory    *CapabilityMemoryService
	policy    *PolicyService
	now       func() time.Time
}

// NewBenchmarkService 校验基准所需的依赖。
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

// Run 让每个任务在两种模式下各跑一遍，返回对比结果。
// 两种模式共用同一份任务列表和同一份已保存的能力记忆，唯一差别是
// 生成之前是否先检索已有能力。
// 本轮运行中新学到的能力不会被同一轮后续任务复用：那度量的是
// 运行内学习，不在本基准的主张范围内。
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

// runMode 以单一模式重放整套任务。
// baseline 每次都生成；evotool 先查能力记忆，没有可复用的才生成。
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

		// Library 复用情况和策略风险都属于 Agent 会提交的那个包本身的属性，
		// 因此在两种模式下都可以度量。
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
	// SuccessRate 和 FailureRate 保持为 0：本基准重放的是复用决策，
	// 它并不执行生成的工具。填 1.0 就是一个没人测量过的断言。
	return result, nil
}

type reuseOutcome struct {
	matched bool
	healthy bool
}

// assessReuse 检索能力记忆，并确认命中的工具确实可运行。
// 依赖检查是判断的一部分：命中了但 Library 缺失的 Tool 不可复用，
// 调用方必须改为生成。
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
		// 没有保存依赖图意味着这个 Tool 从未被登记为可复用；
		// 把它当作健康，就等于声称一次执行时会拒绝的复用是成功的。
		if errors.Is(err, fs.ErrNotExist) {
			return reuseOutcome{matched: true}, nil
		}
		return reuseOutcome{}, fmt.Errorf("check dependencies for %q: %w", toolID, err)
	}
	return reuseOutcome{matched: true, healthy: check.Healthy}, nil
}

// countReusableLibraries 统计声明的 Library 引用中有多少个
// 已经在已保存的能力记忆里解析得到。
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

// policyBlocked 判断待提交的包是否会在执行前被拦下。
// 它只评估声明的元数据，不运行任何代码。
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
