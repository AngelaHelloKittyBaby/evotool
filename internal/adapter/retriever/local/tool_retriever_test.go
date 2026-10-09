package local

import (
	"context"
	"testing"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

func TestToolRetrieverRetrieveByKeyword(t *testing.T) {
	root := t.TempDir()
	registry := filesystem.NewRegistryStore(root)
	ctx := context.Background()

	mustSaveTool(t, registry, ctx, testTool("pdf_to_excel", "document_processing", "python", false, []string{"pdf", "table", "excel"}))
	mustSaveTool(t, registry, ctx, testTool("image_compressor", "image_processing", "python", false, []string{"image", "compress"}))

	result, err := NewToolRetriever(root).Retrieve(ctx, domain.RetrievalQuery{
		Intent:       "extract pdf table into excel",
		SemanticTopK: 10,
		FilterTopK:   10,
	})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(result.Candidates) == 0 {
		t.Fatal("expected candidates")
	}
	if result.Candidates[0].Tool.Name != "pdf_to_excel" {
		t.Fatalf("first candidate = %q, want pdf_to_excel", result.Candidates[0].Tool.Name)
	}
	if len(result.StageReports) != 2 {
		t.Fatalf("stage report count = %d, want 2", len(result.StageReports))
	}
}

func TestToolRetrieverAppliesMetadataFilters(t *testing.T) {
	root := t.TempDir()
	registry := filesystem.NewRegistryStore(root)
	ctx := context.Background()

	mustSaveTool(t, registry, ctx, testTool("local_pdf_tool", "document_processing", "python", false, []string{"pdf", "table"}))
	mustSaveTool(t, registry, ctx, testTool("web_pdf_tool", "document_processing", "node", true, []string{"pdf", "web"}))

	networkAllowed := false
	result, err := NewToolRetriever(root).Retrieve(ctx, domain.RetrievalQuery{
		Intent:      "pdf table",
		Categories:  []string{"document_processing"},
		OutputTypes: []string{"xlsx"},
		Filters: domain.RetrievalFilters{
			Runtime:        "python",
			NetworkAllowed: &networkAllowed,
		},
		SemanticTopK: 10,
		FilterTopK:   10,
	})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(result.Candidates))
	}
	if result.Candidates[0].Tool.Name != "local_pdf_tool" {
		t.Fatalf("candidate = %q, want local_pdf_tool", result.Candidates[0].Tool.Name)
	}
}

func TestToolRetrieverMissingRegistryReturnsEmpty(t *testing.T) {
	result, err := NewToolRetriever(t.TempDir()).Retrieve(context.Background(), domain.RetrievalQuery{Intent: "pdf"})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("candidate count = %d, want 0", len(result.Candidates))
	}
}

func TestCapabilityRankerUsesExecutionExperience(t *testing.T) {
	candidates := []domain.ToolCandidate{
		{
			Tool:  domain.Tool{Name: "similar_but_unreliable"},
			Score: 0.95,
			RankScore: domain.CapabilityRankScore{
				SemanticSimilarity: 0.95,
				EnvironmentMatch:   1,
			},
			UsageStats: domain.ToolUsageStats{UsageCount: 10, FailureCount: 10},
		},
		{
			Tool:  domain.Tool{Name: "reliable"},
			Score: 0.85,
			RankScore: domain.CapabilityRankScore{
				SemanticSimilarity: 0.85,
				EnvironmentMatch:   1,
			},
			UsageStats: domain.ToolUsageStats{UsageCount: 50, SuccessCount: 50, AverageLatency: 100 * time.Millisecond},
		},
	}

	ranked, err := NewCapabilityRanker().Rank(context.Background(), domain.RetrievalQuery{}, candidates)
	if err != nil {
		t.Fatalf("rank: %v", err)
	}
	if ranked[0].Tool.Name != "reliable" {
		t.Fatalf("first ranked tool = %q, want reliable", ranked[0].Tool.Name)
	}
	if ranked[0].RankScore.FinalScore == 0 {
		t.Fatal("expected final score")
	}
}

func mustSaveTool(t *testing.T, registry *filesystem.RegistryStore, ctx context.Context, tool domain.Tool) {
	t.Helper()
	if err := registry.UpsertToolMetadata(ctx, tool); err != nil {
		t.Fatalf("upsert tool metadata: %v", err)
	}
}

func testTool(name string, category string, runtime string, network bool, tags []string) domain.Tool {
	return domain.Tool{
		ID:              name,
		Name:            name,
		Description:     "Test tool " + name,
		Category:        category,
		LifecycleStatus: domain.LifecycleApproved,
		TrustLevel:      domain.TrustValidated,
		Manifest: domain.ToolManifest{
			Name:         name,
			Description:  "Test manifest " + name,
			Category:     category,
			Runtime:      runtime,
			Capabilities: tags,
			Tags:         tags,
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"xlsx_path": map[string]any{"type": "string"},
				},
			},
			Environment: domain.EnvironmentSpec{Runtime: runtime, Network: network},
		},
	}
}
