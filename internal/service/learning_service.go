package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

// LearningService 接收外部生成的包，自身不内置大模型。
type LearningService struct {
	Validator        ports.ToolValidator
	Memory           *CapabilityMemoryService
	Libraries        ports.LibrarySources
	Generator        ports.ToolGenerator
	LibraryRetriever ports.LibraryRetriever
}

// GenerateAndLearn searches library contracts before calling the generator and
// validates the untrusted output through the normal submission workflow.
func (s LearningService) GenerateAndLearn(ctx context.Context, task domain.TaskSpec) (domain.Tool, domain.GeneratedTool, domain.ValidationResult, error) {
	if s.Validator == nil || s.Memory == nil {
		return domain.Tool{}, domain.GeneratedTool{}, domain.ValidationResult{}, ErrMissingDependency
	}
	g, err := s.GenerateCandidate(ctx, task)
	if err != nil {
		return domain.Tool{}, g, domain.ValidationResult{}, err
	}
	t, report, err := s.LearnSubmittedTool(ctx, g)
	return t, g, report, err
}

// GenerateCandidate searches library contracts but never executes model output.
func (s LearningService) GenerateCandidate(ctx context.Context, task domain.TaskSpec) (domain.GeneratedTool, error) {
	if s.Generator == nil {
		return domain.GeneratedTool{}, fmt.Errorf("generator required")
	}
	if strings.TrimSpace(task.Intent) == "" {
		return domain.GeneratedTool{}, fmt.Errorf("task intent required")
	}
	if s.LibraryRetriever != nil {
		libs, err := s.LibraryRetriever.SearchLibraries(ctx, domain.RetrievalQuery{Intent: task.Intent, Limit: 5, Environment: domain.EnvironmentSpec{Runtime: "python"}})
		if err != nil {
			return domain.GeneratedTool{}, err
		}
		task.LibraryCandidates = libs
	}
	return s.Generator.Generate(ctx, task)
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
	if len(g.Manifest.LibraryRefs) > 0 {
		if s.Libraries == nil {
			return domain.Tool{}, domain.ValidationResult{}, fmt.Errorf("library resolver required")
		}
		_, pins, err := s.Libraries.Resolve(ctx, g.Manifest.LibraryRefs)
		if err != nil {
			return domain.Tool{}, domain.ValidationResult{}, err
		}
		g.Manifest.LibraryRefs = pins
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
	t.Generation = g.Generation
	// 先注册空依赖图：中途失败不能留下一个可执行的工具。
	graph := domain.DependencyGraph{Nodes: []domain.DependencyNode{{ID: t.ID, Name: t.Name, Kind: domain.DependencyTool, Version: "v1"}}, HealthStatus: domain.DependencyHealthy}
	for _, ref := range m.LibraryRefs {
		graph.Nodes = append(graph.Nodes, domain.DependencyNode{ID: ref.ID, Name: ref.Name, Kind: domain.DependencyLibrary, Version: ref.Version})
		graph.Edges = append(graph.Edges, domain.DependencyEdge{FromID: t.ID, ToID: ref.ID, Kind: domain.DependencyLibrary})
		graph.VersionConstraints = append(graph.VersionConstraints, domain.VersionConstraint{DependencyID: ref.ID, Constraint: ref.Version})
	}
	if err = s.Memory.registry.SaveDependencyGraph(ctx, graph); err != nil {
		return domain.Tool{}, report, err
	}
	if err = s.Memory.logAudit(ctx, "tool.validated", domain.AuditSubjectTool, t.ID, g.GenerationReason); err != nil {
		return domain.Tool{}, report, err
	}
	err = s.Memory.SaveTool(ctx, t)
	return t, report, err
}
