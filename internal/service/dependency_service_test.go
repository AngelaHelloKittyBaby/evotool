package service

import (
	"context"
	"testing"

	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

func TestDependencyServiceCheckToolHealthy(t *testing.T) {
	stores := filesystem.NewStores(t.TempDir())
	service := newTestDependencyService(t, stores)
	ctx := context.Background()

	library := testDependencyLibrary("pdf_parser", "v1")
	if err := stores.Libraries.Save(ctx, library); err != nil {
		t.Fatalf("save library: %v", err)
	}
	if err := stores.Registry.SaveDependencyGraph(ctx, testDependencyGraph("pdf_to_excel", "pdf_parser", "v1")); err != nil {
		t.Fatalf("save dependency graph: %v", err)
	}

	result, err := service.CheckTool(ctx, "pdf_to_excel")
	if err != nil {
		t.Fatalf("check tool: %v", err)
	}
	if !result.Healthy {
		t.Fatalf("healthy = false, issues = %+v", result.Issues)
	}
	if result.Graph.HealthStatus != domain.DependencyHealthy {
		t.Fatalf("health status = %q, want healthy", result.Graph.HealthStatus)
	}
}

func TestDependencyServiceCheckToolMissingLibraryIsBroken(t *testing.T) {
	stores := filesystem.NewStores(t.TempDir())
	service := newTestDependencyService(t, stores)
	ctx := context.Background()

	if err := stores.Registry.SaveDependencyGraph(ctx, testDependencyGraph("pdf_to_excel", "pdf_parser", "v1")); err != nil {
		t.Fatalf("save dependency graph: %v", err)
	}

	result, err := service.CheckTool(ctx, "pdf_to_excel")
	if err != nil {
		t.Fatalf("check tool: %v", err)
	}
	if result.Healthy {
		t.Fatal("expected broken dependency check")
	}
	if result.Graph.HealthStatus != domain.DependencyBroken {
		t.Fatalf("health status = %q, want broken", result.Graph.HealthStatus)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("issue count = %d, want 1", len(result.Issues))
	}
}

func TestDependencyServiceCheckToolVersionMismatchIsBroken(t *testing.T) {
	stores := filesystem.NewStores(t.TempDir())
	service := newTestDependencyService(t, stores)
	ctx := context.Background()

	library := testDependencyLibrary("pdf_parser", "v2")
	if err := stores.Libraries.Save(ctx, library); err != nil {
		t.Fatalf("save library: %v", err)
	}
	if err := stores.Registry.SaveDependencyGraph(ctx, testDependencyGraph("pdf_to_excel", "pdf_parser", "v1")); err != nil {
		t.Fatalf("save dependency graph: %v", err)
	}

	result, err := service.CheckTool(ctx, "pdf_to_excel")
	if err != nil {
		t.Fatalf("check tool: %v", err)
	}
	if result.Healthy {
		t.Fatal("expected version mismatch")
	}
	if result.Issues[0].Status != domain.DependencyBroken {
		t.Fatalf("issue status = %q, want broken", result.Issues[0].Status)
	}
}

func TestCapabilityMemoryServiceCheckToolDependencies(t *testing.T) {
	stores := filesystem.NewStores(t.TempDir())
	memory, err := NewCapabilityMemoryService(CapabilityMemoryServiceConfig{
		ToolStore:     stores.Tools,
		LibraryStore:  stores.Libraries,
		RegistryStore: stores.Registry,
	})
	if err != nil {
		t.Fatalf("new capability memory service: %v", err)
	}

	ctx := context.Background()
	library := testDependencyLibrary("pdf_parser", "v1")
	if err := stores.Libraries.Save(ctx, library); err != nil {
		t.Fatalf("save library: %v", err)
	}
	if err := stores.Registry.SaveDependencyGraph(ctx, testDependencyGraph("pdf_to_excel", "pdf_parser", "v1")); err != nil {
		t.Fatalf("save dependency graph: %v", err)
	}

	result, err := memory.CheckToolDependencies(ctx, "pdf_to_excel")
	if err != nil {
		t.Fatalf("check tool dependencies: %v", err)
	}
	if !result.Healthy {
		t.Fatalf("healthy = false, issues = %+v", result.Issues)
	}
}

func newTestDependencyService(t *testing.T, stores filesystem.Stores) *DependencyService {
	t.Helper()
	service, err := NewDependencyService(DependencyServiceConfig{
		RegistryStore: stores.Registry,
		LibraryStore:  stores.Libraries,
	})
	if err != nil {
		t.Fatalf("new dependency service: %v", err)
	}
	return service
}

func testDependencyGraph(toolID string, libraryID string, version string) domain.DependencyGraph {
	return domain.DependencyGraph{
		Nodes: []domain.DependencyNode{
			{ID: toolID, Name: toolID, Kind: domain.DependencyTool, Version: "v1"},
			{ID: libraryID, Name: libraryID, Kind: domain.DependencyLibrary, Version: version},
		},
		Edges: []domain.DependencyEdge{
			{FromID: toolID, ToID: libraryID, Kind: domain.DependencyLibrary},
		},
		VersionConstraints: []domain.VersionConstraint{
			{DependencyID: libraryID, Constraint: version},
		},
		HealthStatus: domain.DependencyUnknown,
	}
}

func testDependencyLibrary(id string, version string) domain.Library {
	return domain.Library{
		ID:             id,
		Name:           id,
		CurrentVersion: version,
		Manifest: domain.LibraryManifest{
			Name:    id,
			Runtime: "python",
		},
	}
}
