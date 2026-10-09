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
