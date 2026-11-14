package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

// LearningService 接收外部生成的包，自身不内置大模型。
type LearningService struct {
	Validator ports.ToolValidator
	Memory    *CapabilityMemoryService
}

// LearnSubmittedTool 先验证，再批准并发布首个版本。
// 已存在的 ID 会被拒绝；版本升级需要单独的工作流。
func (s LearningService) LearnSubmittedTool(ctx context.Context, g domain.GeneratedTool) (domain.Tool, domain.ValidationResult, error) {
	if s.Validator == nil || s.Memory == nil {
		return domain.Tool{}, domain.ValidationResult{}, ErrMissingDependency
	}
	if _, err := s.Memory.GetTool(ctx, g.Manifest.Name); err == nil {
		return domain.Tool{}, domain.ValidationResult{}, fmt.Errorf("tool already exists")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return domain.Tool{}, domain.ValidationResult{}, err
	}
	report, err := s.Validator.ValidateGeneratedTool(ctx, g)
	if err != nil || !report.Valid {
		auditErr := s.Memory.logAudit(ctx, "tool.validation_failed", domain.AuditSubjectTool, g.Manifest.Name, fmt.Sprint(report.Errors, err))
		if err == nil {
			err = fmt.Errorf("tool validation failed: %v", report.Errors)
		}
		return domain.Tool{}, report, errors.Join(err, auditErr)
	}
	now := time.Now().UTC()
	m := g.Manifest
	files := append(append([]domain.SourceFile{}, g.SourceFiles...), g.Tests...)
	if g.Readme != "" {
		files = append(files, domain.SourceFile{Path: "README.md", Content: g.Readme})
	}
	t := domain.Tool{ID: m.Name, Name: m.Name, Description: m.Description, Category: m.Category, Manifest: m, CurrentVersion: "v1", Versions: []domain.ToolVersion{{Version: "v1", Files: files, CreatedAt: now}}, LifecycleStatus: domain.LifecycleApproved, TrustLevel: domain.TrustValidated, CreatedAt: now, UpdatedAt: now}
	t.Validation = &report
	t.GenerationReason = g.GenerationReason
	// 先注册空依赖图：中途失败不能留下一个可执行的工具。
	graph := domain.DependencyGraph{Nodes: []domain.DependencyNode{{ID: t.ID, Name: t.Name, Kind: domain.DependencyTool, Version: "v1"}}, HealthStatus: domain.DependencyHealthy}
	if err = s.Memory.registry.SaveDependencyGraph(ctx, graph); err != nil {
		return domain.Tool{}, report, err
	}
	if err = s.Memory.logAudit(ctx, "tool.validated", domain.AuditSubjectTool, t.ID, g.GenerationReason); err != nil {
		return domain.Tool{}, report, err
	}
	err = s.Memory.SaveTool(ctx, t)
	return t, report, err
}
