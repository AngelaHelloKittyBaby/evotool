package service

import (
	"context"
	"errors"
	"testing"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

type executionFixture struct {
	tool                                                            domain.Tool
	health                                                          domain.DependencyHealthStatus
	decision                                                        domain.PolicyDecision
	loadErr, dependencyErr, policyErr, execErr, recordErr, auditErr error
	usageErr                                                        error
	executed                                                        domain.ExecutionResult
	calls                                                           []string
	results                                                         []domain.ExecutionResult
	usages                                                          []domain.ExecutionResult
	cancel                                                          context.CancelFunc
}

func (f *executionFixture) Save(context.Context, domain.Tool) error     { return nil }
func (f *executionFixture) Update(context.Context, domain.Tool) error   { return nil }
func (f *executionFixture) List(context.Context) ([]domain.Tool, error) { return nil, nil }
func (f *executionFixture) Get(context.Context, string) (domain.Tool, error) {
	f.calls = append(f.calls, "load")
	return f.tool, f.loadErr
}
func (f *executionFixture) ResolveTool(context.Context, domain.Tool) (domain.DependencyGraph, error) {
	f.calls = append(f.calls, "dependencies")
	return domain.DependencyGraph{HealthStatus: f.health}, f.dependencyErr
}
func (f *executionFixture) ResolveLibrary(context.Context, domain.Library) (domain.DependencyGraph, error) {
	return domain.DependencyGraph{}, nil
}
func (f *executionFixture) Evaluate(context.Context, domain.Tool, domain.TaskSpec) (domain.PolicyDecision, error) {
	f.calls = append(f.calls, "policy")
	return f.decision, f.policyErr
}
func (f *executionFixture) Execute(context.Context, domain.Tool, map[string]any) (domain.ExecutionResult, error) {
	f.calls = append(f.calls, "execute")
	if f.cancel != nil {
		f.cancel()
	}
	return f.executed, f.execErr
}
func (f *executionFixture) RecordExecution(ctx context.Context, r domain.ExecutionResult) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	f.calls = append(f.calls, "record")
	f.results = append(f.results, r)
	return f.recordErr
}
func (f *executionFixture) Log(_ context.Context, e domain.AuditEvent) error {
	f.calls = append(f.calls, e.Action)
	return f.auditErr
}

func (f *executionFixture) RecordUsage(_ context.Context, r domain.ExecutionResult) error {
	f.calls = append(f.calls, "usage")
	f.usages = append(f.usages, r)
	return f.usageErr
}

func newExecutionFixture() *executionFixture {
	return &executionFixture{tool: domain.Tool{ID: "csv_analyzer", CurrentVersion: "v1", Versions: []domain.ToolVersion{{Version: "v1"}}}, health: domain.DependencyHealthy, decision: domain.PolicyDecision{Allowed: true}, executed: domain.ExecutionResult{Success: true, Output: "report", ToolID: "spoof", Version: "wrong"}}
}
func (f *executionFixture) config() ExecutionServiceConfig {
	return ExecutionServiceConfig{ToolStore: f, Dependencies: f, Policy: f, Executor: f, Recorder: f, Audit: f, Usage: f}
}

func TestExecutionServicePaths(t *testing.T) {
	boom := errors.New("failure")
	for _, tt := range []struct {
		name                     string
		edit                     func(*executionFixture)
		execute, record, wantErr bool
	}{
		{"success", func(*executionFixture) {}, true, true, false},
		{"load failure", func(f *executionFixture) { f.loadErr = boom }, false, false, true},
		{"missing version", func(f *executionFixture) { f.tool.Versions = nil }, false, true, true},
		{"dependency failure", func(f *executionFixture) { f.dependencyErr = boom }, false, true, true},
		{"unhealthy", func(f *executionFixture) { f.health = domain.DependencyBroken }, false, true, true},
		{"unknown health", func(f *executionFixture) { f.health = domain.DependencyUnknown }, false, true, true},
		{"policy failure", func(f *executionFixture) { f.policyErr = boom }, false, true, true},
		{"denied", func(f *executionFixture) { f.decision.Allowed = false }, false, true, true},
		{"review despite allowed", func(f *executionFixture) { f.decision.RequiresReview = true }, false, true, true},
		{"start audit failure", func(f *executionFixture) { f.auditErr = boom }, false, true, true},
		{"executor error", func(f *executionFixture) { f.execErr = boom }, true, true, true},
		{"executor unsuccessful", func(f *executionFixture) { f.executed.Success = false }, true, true, true},
		{"record failure", func(f *executionFixture) { f.recordErr = boom }, true, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newExecutionFixture()
			tt.edit(f)
			s, err := NewExecutionService(f.config())
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.ExecuteTool(context.Background(), "csv_analyzer", domain.TaskSpec{}, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			didExecute := false
			for _, c := range f.calls {
				if c == "execute" {
					didExecute = true
				}
			}
			if didExecute != tt.execute || (len(f.results) > 0) != tt.record {
				t.Fatalf("unexpected calls: %v", f.calls)
			}
			// Usage statistics only move when the executor actually ran.
			if (len(f.usages) > 0) != tt.execute {
				t.Fatalf("usage recorded without execution: %v", f.calls)
			}
			if tt.record && (r.ToolID != "csv_analyzer" || r.Version != "v1" || r.StartedAt.IsZero() || r.FinishedAt.Before(r.StartedAt)) {
				t.Fatalf("invalid result: %+v", r)
			}
			if tt.execute && tt.record && f.recordErr == nil && f.execErr != nil && !errors.Is(err, boom) {
				t.Fatalf("lost executor error: %v", err)
			}
			if !tt.execute && tt.record && r.Success {
				t.Fatal("blocked execution marked successful")
			}
			if !tt.wantErr {
				want := []string{"load", "dependencies", "policy", "execution.started", "execute", "record", "usage", "execution.completed"}
				if len(want) != len(f.calls) {
					t.Fatalf("calls: %v", f.calls)
				}
				for i := range want {
					if want[i] != f.calls[i] {
						t.Fatalf("calls: %v", f.calls)
					}
				}
			}
		})
	}
}

func TestExecutionServiceCancellationRecordsFailure(t *testing.T) {
	f := newExecutionFixture()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.cancel = cancel
	f.execErr = context.Canceled
	s, _ := NewExecutionService(f.config())
	r, err := s.ExecuteTool(ctx, "csv_analyzer", domain.TaskSpec{}, nil)
	if !errors.Is(err, context.Canceled) || r.Success || len(f.results) != 1 {
		t.Fatalf("result=%+v records=%d error=%v", r, len(f.results), err)
	}
}

func TestExecutionServiceRecordsExecutorFailureAsUsage(t *testing.T) {
	f := newExecutionFixture()
	f.execErr = errors.New("sandbox died")
	s, _ := NewExecutionService(f.config())
	if _, err := s.ExecuteTool(context.Background(), "csv_analyzer", domain.TaskSpec{}, nil); err == nil {
		t.Fatal("expected executor error")
	}
	if len(f.usages) != 1 || f.usages[0].Success {
		t.Fatalf("expected one failed usage record, got %+v", f.usages)
	}
}

func TestExecutionServiceUsageFailureDoesNotDropResult(t *testing.T) {
	boom := errors.New("usage store down")
	f := newExecutionFixture()
	f.usageErr = boom
	s, _ := NewExecutionService(f.config())
	result, err := s.ExecuteTool(context.Background(), "csv_analyzer", domain.TaskSpec{}, nil)
	if !errors.Is(err, boom) {
		t.Fatalf("usage error not propagated: %v", err)
	}
	if !result.Success || result.Output != "report" {
		t.Fatalf("execution result lost: %+v", result)
	}
}

func TestExecutionServiceUsageIsOptional(t *testing.T) {
	f := newExecutionFixture()
	config := f.config()
	config.Usage = nil
	s, err := NewExecutionService(config)
	if err != nil {
		t.Fatalf("usage recorder must be optional: %v", err)
	}
	if _, err := s.ExecuteTool(context.Background(), "csv_analyzer", domain.TaskSpec{}, nil); err != nil {
		t.Fatalf("execute without usage recorder: %v", err)
	}
	if len(f.usages) != 0 || len(f.results) != 1 {
		t.Fatalf("usages=%d records=%d", len(f.usages), len(f.results))
	}
}

func TestExecutionServiceRequiredDependencies(t *testing.T) {
	f := newExecutionFixture()
	for i := 0; i < 6; i++ {
		c := f.config()
		switch i {
		case 0:
			c.ToolStore = nil
		case 1:
			c.Dependencies = nil
		case 2:
			c.Policy = nil
		case 3:
			c.Executor = nil
		case 4:
			c.Recorder = nil
		case 5:
			c.Audit = nil
		}
		if _, err := NewExecutionService(c); !errors.Is(err, ErrMissingDependency) {
			t.Fatalf("dependency %d: %v", i, err)
		}
	}
}

func TestCapabilityMemoryExecutionDelegation(t *testing.T) {
	f := newExecutionFixture()
	execution, err := NewExecutionService(f.config())
	if err != nil {
		t.Fatal(err)
	}
	memory := &CapabilityMemoryService{execution: execution}
	result, err := memory.ExecuteTool(context.Background(), "csv_analyzer", domain.TaskSpec{}, nil)
	if err != nil || !result.Success || len(f.results) != 1 {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	memory.execution = nil
	if _, err := memory.ExecuteTool(context.Background(), "csv_analyzer", domain.TaskSpec{}, nil); !errors.Is(err, ErrMissingDependency) {
		t.Fatalf("expected missing execution service: %v", err)
	}
}
