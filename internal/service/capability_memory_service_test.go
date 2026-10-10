package service

import (
	"context"
	"testing"

	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

func TestCapabilityMemoryServiceSearchLibraries(t *testing.T) {
	stores := filesystem.NewStores(t.TempDir())
	memory, err := NewCapabilityMemoryService(CapabilityMemoryServiceConfig{
		ToolStore:        stores.Tools,
		LibraryStore:     stores.Libraries,
		RegistryStore:    stores.Registry,
		LibraryRetriever: fakeLibraryRetriever{},
	})
	if err != nil {
		t.Fatalf("new capability memory service: %v", err)
	}

	candidates, err := memory.SearchLibraries(context.Background(), domain.RetrievalQuery{Intent: "parse pdf"})
	if err != nil {
		t.Fatalf("search libraries: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Library.Name != "pdf_parser" {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
}

func TestCapabilityMemoryServiceSearchLibrariesRequiresRetriever(t *testing.T) {
	stores := filesystem.NewStores(t.TempDir())
	memory, err := NewCapabilityMemoryService(CapabilityMemoryServiceConfig{
		ToolStore:     stores.Tools,
		LibraryStore:  stores.Libraries,
		RegistryStore: stores.Registry,
	})
	if err != nil {
		t.Fatalf("new capability memory service: %v", err)
	}

	if _, err := memory.SearchLibraries(context.Background(), domain.RetrievalQuery{}); err == nil {
		t.Fatal("expected missing library retriever error")
	}
}

func TestCapabilityMemoryServiceLinkToolLibraries(t *testing.T) {
	stores := filesystem.NewStores(t.TempDir())
	memory, err := NewCapabilityMemoryService(CapabilityMemoryServiceConfig{
		ToolStore:     stores.Tools,
		LibraryStore:  stores.Libraries,
		RegistryStore: stores.Registry,
		AuditLogger:   stores.Audit,
	})
	if err != nil {
		t.Fatalf("new capability memory service: %v", err)
	}

	ctx := context.Background()
	tool := domain.Tool{
		ID:             "pdf_to_excel",
		Name:           "pdf_to_excel",
		CurrentVersion: "v1",
		Manifest: domain.ToolManifest{
			Name:       "pdf_to_excel",
			Runtime:    "python",
			EntryPoint: "main.py",
		},
	}
	library := domain.Library{
		ID:             "pdf_parser",
		Name:           "pdf_parser",
		CurrentVersion: "v1",
		Manifest: domain.LibraryManifest{
			Name:    "pdf_parser",
			Runtime: "python",
		},
	}

	graph, err := memory.LinkToolLibraries(ctx, tool, []domain.Library{library})
	if err != nil {
		t.Fatalf("link tool libraries: %v", err)
	}
	if len(graph.Nodes) != 2 {
		t.Fatalf("node count = %d, want 2", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("edge count = %d, want 1", len(graph.Edges))
	}
	if graph.Edges[0].FromID != "pdf_to_excel" || graph.Edges[0].ToID != "pdf_parser" {
		t.Fatalf("unexpected edge: %+v", graph.Edges[0])
	}

	loadedGraph, err := memory.GetDependencyGraph(ctx, "pdf_to_excel")
	if err != nil {
		t.Fatalf("get dependency graph: %v", err)
	}
	if len(loadedGraph.Edges) != 1 {
		t.Fatalf("loaded graph edge count = %d, want 1", len(loadedGraph.Edges))
	}

	loadedTool, err := memory.GetTool(ctx, "pdf_to_excel")
	if err != nil {
		t.Fatalf("get tool: %v", err)
	}
	if len(loadedTool.Manifest.LibraryRefs) != 1 {
		t.Fatalf("library ref count = %d, want 1", len(loadedTool.Manifest.LibraryRefs))
	}
	if loadedTool.Manifest.LibraryRefs[0].ID != "pdf_parser" {
		t.Fatalf("library ref id = %q, want pdf_parser", loadedTool.Manifest.LibraryRefs[0].ID)
	}
}

func TestCapabilityMemoryServiceLinkToolLibrariesRequiresLibrary(t *testing.T) {
	stores := filesystem.NewStores(t.TempDir())
	memory, err := NewCapabilityMemoryService(CapabilityMemoryServiceConfig{
		ToolStore:     stores.Tools,
		LibraryStore:  stores.Libraries,
		RegistryStore: stores.Registry,
	})
	if err != nil {
		t.Fatalf("new capability memory service: %v", err)
	}

	_, err = memory.LinkToolLibraries(context.Background(), domain.Tool{ID: "pdf_to_excel"}, nil)
	if err == nil {
		t.Fatal("expected missing library error")
	}
}

type fakeLibraryRetriever struct{}

func (fakeLibraryRetriever) SearchLibraries(_ context.Context, _ domain.RetrievalQuery) ([]domain.LibraryCandidate, error) {
	return []domain.LibraryCandidate{
		{
			Library:    domain.Library{Name: "pdf_parser"},
			Score:      1,
			Compatible: true,
		},
	}, nil
}
