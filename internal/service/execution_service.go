package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

// ErrExecutionDenied 标识一次在执行器运行前就被拦下的执行。
var ErrExecutionDenied = errors.New("execution denied")

// ExecutionServiceConfig 提供执行所需的各项边界。
// UsageRecorder 是可选的：省略它只保留执行历史，
// 排序所用的统计则停留在默认值上。
type ExecutionServiceConfig struct {
	ToolStore    ports.ToolStore
	Dependencies ports.DependencyResolver
	Policy       ports.PolicyEngine
	Executor     ports.ToolExecutor
	Recorder     ports.ExecutionRecorder
	Audit        ports.AuditLogger
	Usage        ports.UsageRecorder
}

// ExecutionService 编排带检查的执行过程和可持久化的结果上报。
type ExecutionService struct{ config ExecutionServiceConfig }

// NewExecutionService 要求显式提供每一项安全与上报依赖。
func NewExecutionService(c ExecutionServiceConfig) (*ExecutionService, error) {
	for _, d := range []struct {
		name    string
		missing bool
	}{
		{"tool store", c.ToolStore == nil}, {"dependency resolver", c.Dependencies == nil},
		{"policy engine", c.Policy == nil}, {"executor", c.Executor == nil},
		{"execution recorder", c.Recorder == nil}, {"audit logger", c.Audit == nil},
	} {
		if d.missing {
			return nil, fmt.Errorf("%w: %s", ErrMissingDependency, d.name)
		}
	}
	return &ExecutionService{config: c}, nil
}

// ExecuteTool 读取已保存的版本，检查依赖和策略，并记录被拦截的尝试
// 与执行器的失败。只有真正运行过执行器得到的结果才会更新使用统计。
// 即使执行或后续持久化失败，调用方仍会拿到结果。
func (s *ExecutionService) ExecuteTool(ctx context.Context, id string, task domain.TaskSpec, input map[string]any) (domain.ExecutionResult, error) {
	if err := checkContext(ctx); err != nil {
		return domain.ExecutionResult{}, err
	}
	if strings.TrimSpace(id) == "" {
		return domain.ExecutionResult{}, fmt.Errorf("tool id is required")
	}
	tool, err := s.config.ToolStore.Get(ctx, id)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("load execution tool: %w", err)
	}
	result := domain.ExecutionResult{ToolID: tool.ID, Version: tool.CurrentVersion, StartedAt: time.Now().UTC()}
	if result.ToolID == "" {
		result.ToolID = id
	}
	fail := func(err error) (domain.ExecutionResult, error) {
		result.Success = false
		result.Error = err.Error()
		return s.finish(ctx, result, false, err)
	}
	found := false
	for _, version := range tool.Versions {
		if version.Version == tool.CurrentVersion && version.Version != "" {
			found = true
		}
	}
	if !found {
		return fail(fmt.Errorf("%w: current version is missing from tool package", ErrExecutionDenied))
	}
	graph, err := s.config.Dependencies.ResolveTool(ctx, tool)
	if err != nil {
		return fail(fmt.Errorf("resolve execution dependencies: %w", err))
	}
	if graph.HealthStatus != domain.DependencyHealthy {
		return fail(fmt.Errorf("%w: dependency health is %s", ErrExecutionDenied, graph.HealthStatus))
	}
	decision, err := s.config.Policy.Evaluate(ctx, tool, task)
	if err != nil {
		return fail(fmt.Errorf("evaluate execution policy: %w", err))
	}
	if !decision.Allowed || decision.RequiresReview {
		return fail(fmt.Errorf("%w: policy requires_review=%t: %s", ErrExecutionDenied, decision.RequiresReview, strings.Join(decision.Reasons, "; ")))
	}
	if err := checkContext(ctx); err != nil {
		return fail(err)
	}
	if err := s.config.Audit.Log(ctx, domain.AuditEvent{Action: "execution.started", SubjectKind: domain.AuditSubjectExecution, SubjectID: result.ToolID, Message: result.Version, CreatedAt: result.StartedAt}); err != nil {
		return fail(fmt.Errorf("audit execution start: %w", err))
	}
	executed, execErr := s.config.Executor.Execute(ctx, tool, input)
	// 标识和时间戳属于本次调用，而不是适配器的输出。
	executed.ToolID, executed.Version, executed.StartedAt = result.ToolID, result.Version, result.StartedAt
	// 从这里开始执行器已经运行过，因此这个结果属于使用经验。
	if execErr != nil {
		executed.Success = false
		executed.Error = strings.TrimSpace(strings.Join([]string{executed.Error, execErr.Error()}, " "))
		execErr = fmt.Errorf("execute tool: %w", execErr)
	} else if !executed.Success {
		if executed.Error == "" {
			executed.Error = "executor reported failure"
		}
		execErr = fmt.Errorf("execute tool: %s", executed.Error)
	}
	return s.finish(ctx, executed, true, execErr)
}

// finish 持久化本次调用。executed 表示执行器是否真的运行过：
// 策略拦截和依赖失败只写历史与审计，而执行器给出的结果
// 还会更新排序所依据的使用统计。
func (s *ExecutionService) finish(ctx context.Context, result domain.ExecutionResult, executed bool, cause error) (domain.ExecutionResult, error) {
	result.FinishedAt = time.Now().UTC()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)
	// 取消上下文不能丢掉失败记录；这里的持久化有超时上限。
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.config.Recorder.RecordExecution(recordCtx, result); err != nil {
		cause = errors.Join(cause, fmt.Errorf("record execution: %w", err))
	}
	if executed && s.config.Usage != nil {
		if err := s.config.Usage.RecordUsage(recordCtx, result); err != nil {
			cause = errors.Join(cause, fmt.Errorf("record usage: %w", err))
		}
	}
	action := "execution.completed"
	if !result.Success {
		action = "execution.failed"
	}
	if err := s.config.Audit.Log(recordCtx, domain.AuditEvent{Action: action, SubjectKind: domain.AuditSubjectExecution, SubjectID: result.ToolID, Message: result.Error, Metadata: map[string]string{"version": result.Version}, CreatedAt: result.FinishedAt}); err != nil {
		cause = errors.Join(cause, fmt.Errorf("audit execution result: %w", err))
	}
	return result, cause
}
