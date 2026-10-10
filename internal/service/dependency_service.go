package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

// DependencyServiceConfig contains dependencies for dependency graph health checks.
type DependencyServiceConfig struct {
	RegistryStore ports.RegistryStore
	LibraryStore  ports.LibraryStore
}

// DependencyService checks dependency graphs before a Tool is executed.
type DependencyService struct {
	registry  ports.RegistryStore
	libraries ports.LibraryStore
}

// NewDependencyService creates a dependency health checker.
func NewDependencyService(config DependencyServiceConfig) (*DependencyService, error) {
	if config.RegistryStore == nil {
		return nil, fmt.Errorf("%w: registry store", ErrMissingDependency)
	}
	if config.LibraryStore == nil {
		return nil, fmt.Errorf("%w: library store", ErrMissingDependency)
	}
	return &DependencyService{registry: config.RegistryStore, libraries: config.LibraryStore}, nil
}

// CheckTool loads and checks the dependency graph for a Tool root ID.
func (s *DependencyService) CheckTool(ctx context.Context, toolID string) (domain.DependencyCheckResult, error) {
	if strings.TrimSpace(toolID) == "" {
		return domain.DependencyCheckResult{}, fmt.Errorf("tool id is required")
	}
	graph, err := s.registry.GetDependencyGraph(ctx, toolID)
	if err != nil {
		return domain.DependencyCheckResult{}, fmt.Errorf("load dependency graph: %w", err)
	}
	return s.CheckGraph(ctx, graph)
}

// CheckGraph verifies that required Library dependencies are available and version-compatible.
func (s *DependencyService) CheckGraph(ctx context.Context, graph domain.DependencyGraph) (domain.DependencyCheckResult, error) {
	issues := make([]domain.DependencyIssue, 0)
	constraints := versionConstraintMap(graph.VersionConstraints)
	for _, node := range graph.Nodes {
		if err := checkContext(ctx); err != nil {
			return domain.DependencyCheckResult{}, err
		}
		if node.Kind != domain.DependencyLibrary {
			continue
		}

		library, err := s.libraries.Get(ctx, node.ID)
		if err != nil {
			issues = append(issues, domain.DependencyIssue{
				DependencyID: node.ID,
				Kind:         node.Kind,
				Status:       domain.DependencyBroken,
				Message:      fmt.Sprintf("library %q is missing or unreadable: %v", node.ID, err),
			})
			continue
		}

		expectedVersion := firstNonEmpty(constraints[node.ID], node.Version)
		if expectedVersion != "" && library.CurrentVersion != "" && expectedVersion != library.CurrentVersion {
			issues = append(issues, domain.DependencyIssue{
				DependencyID: node.ID,
				Kind:         node.Kind,
				Status:       domain.DependencyBroken,
				Message:      fmt.Sprintf("library %q version mismatch: expected %s, got %s", node.ID, expectedVersion, library.CurrentVersion),
			})
			continue
		}

		if library.CurrentVersion == "" {
			issues = append(issues, domain.DependencyIssue{
				DependencyID: node.ID,
				Kind:         node.Kind,
				Status:       domain.DependencyDegraded,
				Message:      fmt.Sprintf("library %q has no current version", node.ID),
			})
		}
	}

	graph.HealthStatus = dependencyHealthFromIssues(issues)
	return domain.DependencyCheckResult{
		Graph:   graph,
		Healthy: graph.HealthStatus == domain.DependencyHealthy,
		Issues:  issues,
	}, nil
}

// ResolveTool implements ports.DependencyResolver for Tool dependency graphs.
func (s *DependencyService) ResolveTool(ctx context.Context, tool domain.Tool) (domain.DependencyGraph, error) {
	result, err := s.CheckTool(ctx, capabilityID(tool.ID, tool.Name))
	if err != nil {
		return domain.DependencyGraph{}, err
	}
	return result.Graph, nil
}

// ResolveLibrary implements ports.DependencyResolver for Library dependency graphs.
func (s *DependencyService) ResolveLibrary(_ context.Context, library domain.Library) (domain.DependencyGraph, error) {
	libraryID := capabilityID(library.ID, library.Name)
	if libraryID == "" {
		return domain.DependencyGraph{}, fmt.Errorf("library id or name is required")
	}
	return domain.DependencyGraph{
		Nodes: []domain.DependencyNode{
			{ID: libraryID, Name: firstNonEmpty(library.Name, libraryID), Kind: domain.DependencyLibrary, Version: library.CurrentVersion},
		},
		HealthStatus: domain.DependencyUnknown,
	}, nil
}

func versionConstraintMap(constraints []domain.VersionConstraint) map[string]string {
	mapped := make(map[string]string, len(constraints))
	for _, constraint := range constraints {
		id := strings.TrimSpace(constraint.DependencyID)
		if id != "" {
			mapped[id] = strings.TrimSpace(constraint.Constraint)
		}
	}
	return mapped
}

func dependencyHealthFromIssues(issues []domain.DependencyIssue) domain.DependencyHealthStatus {
	if len(issues) == 0 {
		return domain.DependencyHealthy
	}
	for _, issue := range issues {
		if issue.Status == domain.DependencyBroken {
			return domain.DependencyBroken
		}
	}
	return domain.DependencyDegraded
}

func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

var _ ports.DependencyResolver = (*DependencyService)(nil)
