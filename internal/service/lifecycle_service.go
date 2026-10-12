package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

// LifecycleServiceConfig contains dependencies and policy for lifecycle scans.
type LifecycleServiceConfig struct {
	ToolStore ports.ToolStore
	Policy    domain.LifecyclePolicy
	Now       func() time.Time
}

// LifecycleService evaluates Tool retention candidates without mutating storage.
type LifecycleService struct {
	tools  ports.ToolStore
	policy domain.LifecyclePolicy
	now    func() time.Time
}

// NewLifecycleService creates a lifecycle scan service.
func NewLifecycleService(config LifecycleServiceConfig) (*LifecycleService, error) {
	if config.ToolStore == nil {
		return nil, fmt.Errorf("%w: tool store", ErrMissingDependency)
	}
	if config.Policy.StaleAfter <= 0 {
		return nil, fmt.Errorf("stale threshold must be greater than zero")
	}
	if config.Policy.ArchiveAfter <= config.Policy.StaleAfter {
		return nil, fmt.Errorf("archive threshold must be greater than stale threshold")
	}

	now := config.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &LifecycleService{
		tools:  config.ToolStore,
		policy: config.Policy,
		now:    now,
	}, nil
}

// Scan returns lifecycle candidates without changing Tool metadata or files.
func (s *LifecycleService) Scan(ctx context.Context) (domain.LifecycleScanReport, error) {
	tools, err := s.tools.List(ctx)
	if err != nil {
		return domain.LifecycleScanReport{}, fmt.Errorf("list tools for lifecycle scan: %w", err)
	}

	now := s.now().UTC()
	report := domain.LifecycleScanReport{
		EvaluatedAt: now,
		TotalTools:  len(tools),
		Candidates:  make([]domain.LifecycleCandidate, 0),
	}
	for _, tool := range tools {
		if err := checkContext(ctx); err != nil {
			return domain.LifecycleScanReport{}, err
		}
		candidate, ok := s.evaluate(tool, now)
		if ok {
			report.Candidates = append(report.Candidates, candidate)
		}
	}

	sort.SliceStable(report.Candidates, func(i, j int) bool {
		if report.Candidates[i].IdleFor == report.Candidates[j].IdleFor {
			return report.Candidates[i].ToolID < report.Candidates[j].ToolID
		}
		return report.Candidates[i].IdleFor > report.Candidates[j].IdleFor
	})
	return report, nil
}

func (s *LifecycleService) evaluate(tool domain.Tool, now time.Time) (domain.LifecycleCandidate, bool) {
	if !lifecycleScanEligible(tool.LifecycleStatus) {
		return domain.LifecycleCandidate{}, false
	}

	lastActivity, source := lastActivityAt(tool)
	if lastActivity.IsZero() || lastActivity.After(now) {
		return domain.LifecycleCandidate{}, false
	}

	idleFor := now.Sub(lastActivity)
	action := domain.LifecycleActionNone
	proposedStatus := tool.LifecycleStatus
	switch {
	case idleFor >= s.policy.ArchiveAfter:
		action = domain.LifecycleActionArchiveCandidate
		proposedStatus = domain.LifecycleArchived
	case idleFor >= s.policy.StaleAfter && tool.LifecycleStatus != domain.LifecycleStale:
		action = domain.LifecycleActionMarkStale
		proposedStatus = domain.LifecycleStale
	default:
		return domain.LifecycleCandidate{}, false
	}

	toolID := firstNonEmpty(tool.ID, tool.Name)
	toolName := firstNonEmpty(tool.Name, tool.ID)
	return domain.LifecycleCandidate{
		ToolID:         toolID,
		ToolName:       toolName,
		CurrentStatus:  tool.LifecycleStatus,
		ProposedStatus: proposedStatus,
		Action:         action,
		LastActivityAt: lastActivity,
		IdleFor:        idleFor,
		Reason:         fmt.Sprintf("no recorded use for %s; activity source=%s", idleFor.Round(time.Second), source),
	}, true
}

func lifecycleScanEligible(status domain.LifecycleStatus) bool {
	switch status {
	case "", domain.LifecycleApproved, domain.LifecycleActive, domain.LifecycleStale:
		return true
	default:
		return false
	}
}

func lastActivityAt(tool domain.Tool) (time.Time, string) {
	if !tool.UsageStats.LastUsedAt.IsZero() {
		return tool.UsageStats.LastUsedAt, "last_used_at"
	}
	if !tool.UpdatedAt.IsZero() {
		return tool.UpdatedAt, "updated_at"
	}
	if !tool.CreatedAt.IsZero() {
		return tool.CreatedAt, "created_at"
	}
	return time.Time{}, ""
}
