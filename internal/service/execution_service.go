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

// ErrExecutionDenied identifies an execution blocked before the executor runs.
var ErrExecutionDenied = errors.New("execution denied")

// ExecutionServiceConfig supplies the required execution boundaries.
type ExecutionServiceConfig struct {
	ToolStore    ports.ToolStore
	Dependencies ports.DependencyResolver
	Policy       ports.PolicyEngine
	Executor     ports.ToolExecutor
	Recorder     ports.ExecutionRecorder
	Audit        ports.AuditLogger
}

// ExecutionService orchestrates checked execution and durable result reporting.
type ExecutionService struct{ config ExecutionServiceConfig }

// NewExecutionService requires every safety and reporting dependency explicitly.
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

// ExecuteTool loads the stored version, checks dependencies and policy, and
// records both blocked attempts and executor failures. Callers receive the result
// even when execution or subsequent persistence fails.
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
		return s.finish(ctx, result, err)
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
	// Identity and timestamps belong to this invocation, not adapter output.
	executed.ToolID, executed.Version, executed.StartedAt = result.ToolID, result.Version, result.StartedAt
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
	return s.finish(ctx, executed, execErr)
}

func (s *ExecutionService) finish(ctx context.Context, result domain.ExecutionResult, cause error) (domain.ExecutionResult, error) {
	result.FinishedAt = time.Now().UTC()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)
	// Cancellation must not discard the failure record; persistence is bounded.
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.config.Recorder.RecordExecution(recordCtx, result); err != nil {
		cause = errors.Join(cause, fmt.Errorf("record execution: %w", err))
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
