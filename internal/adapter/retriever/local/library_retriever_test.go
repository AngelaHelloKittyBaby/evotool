package local

import (
	"context"
	"testing"

	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

func TestLibraryRetrieverSearchesReusableLibrary(t *testing.T) {
	root := t.TempDir()
	registry := filesystem.NewRegistryStore(root)
	ctx := context.Background()

	mustSaveLibrary(t, registry, ctx, testLibrary("pdf_parser", "document_processing", "python", false, []string{"pdf", "parser", "table"}))
	mustSaveLibrary(t, registry, ctx, testLibrary("image_loader", "image_processing", "python", false, []string{"image", "loader"}))

	candidates, err := NewLibraryRetriever(root).SearchLibraries(ctx, domain.RetrievalQuery{
		Intent:     "parse pdf tables",
		Categories: []string{"document_processing"},
		Filters: domain.RetrievalFilters{
			Runtime: "python",
		},
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("search libraries: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(candidates))
	}
	if candidates[0].Library.Name != "pdf_parser" {
		t.Fatalf("candidate = %q, want pdf_parser", candidates[0].Library.Name)
	}
	if !candidates[0].Compatible {
		t.Fatal("expected compatible library")
	}
	if candidates[0].Score <= 0 {
		t.Fatal("expected positive score")
	}
}

func TestLibraryRetrieverAppliesNetworkFilter(t *testing.T) {
	root := t.TempDir()
	registry := filesystem.NewRegistryStore(root)
	ctx := context.Background()
	mustSaveLibrary(t, registry, ctx, testLibrary("remote_pdf_parser", "document_processing", "python", true, []string{"pdf", "parser"}))

	allowed := false
	candidates, err := NewLibraryRetriever(root).SearchLibraries(ctx, domain.RetrievalQuery{
		Intent: "pdf parser",
		Filters: domain.RetrievalFilters{
			NetworkAllowed: &allowed,
		},
	})
	if err != nil {
		t.Fatalf("search libraries: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidate count = %d, want 0", len(candidates))
	}
}

func TestLibraryRetrieverMissingRegistryReturnsEmpty(t *testing.T) {
	candidates, err := NewLibraryRetriever(t.TempDir()).SearchLibraries(context.Background(), domain.RetrievalQuery{Intent: "pdf"})
	if err != nil {
		t.Fatalf("search libraries: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidate count = %d, want 0", len(candidates))
	}
}

func mustSaveLibrary(t *testing.T, registry *filesystem.RegistryStore, ctx context.Context, library domain.Library) {
	t.Helper()
	if err := registry.UpsertLibraryMetadata(ctx, library); err != nil {
		t.Fatalf("upsert library metadata: %v", err)
	}
}

func testLibrary(name string, category string, runtime string, network bool, tags []string) domain.Library {
	return domain.Library{
		ID:              name,
		Name:            name,
		Description:     "Test library " + name,
		Category:        category,
		LifecycleStatus: domain.LifecycleApproved,
		TrustLevel:      domain.TrustValidated,
		Manifest: domain.LibraryManifest{
			Name:        name,
			Description: "Test manifest " + name,
			Category:    category,
			Runtime:     runtime,
			Tags:        tags,
			Permissions: domain.PermissionSet{Network: networkPolicy(network)},
		},
	}
}

func networkPolicy(network bool) domain.NetworkPolicy {
	if network {
		return domain.NetworkPolicyFull
	}
	return domain.NetworkPolicyNone
}
