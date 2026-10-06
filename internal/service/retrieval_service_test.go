package service

import (
	"context"
	"testing"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

func TestRetrievalServiceSearchRanksAndSelects(t *testing.T) {
	service, err := NewRetrievalService(RetrievalServiceConfig{
		IntentExtractor: fakeIntentExtractor{},
		ToolRetriever: fakeToolRetriever{
			candidates: []domain.ToolCandidate{
				{Tool: domain.Tool{ID: "slow", Name: "slow_tool"}},
				{Tool: domain.Tool{ID: "fast", Name: "fast_tool"}},
			},
		},
		Ranker:   fakeRanker{},
		Selector: fakeSelector{},
	})
	if err != nil {
		t.Fatalf("new retrieval service: %v", err)
	}

	result, err := service.Search(context.Background(), domain.TaskSpec{Intent: "extract pdf tables"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if result.Selected == nil {
		t.Fatal("expected selected tool")
	}
	if result.Selected.Tool.ID != "fast" {
		t.Fatalf("selected tool id = %q, want fast", result.Selected.Tool.ID)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(result.Candidates))
	}
	if len(result.StageReports) == 0 {
		t.Fatal("expected stage reports")
	}
}

func TestNewRetrievalServiceRequiresDependencies(t *testing.T) {
	_, err := NewRetrievalService(RetrievalServiceConfig{})
	if err == nil {
		t.Fatal("expected error")
	}
}

type fakeIntentExtractor struct{}

func (fakeIntentExtractor) Extract(_ context.Context, task domain.TaskSpec) (domain.RetrievalQuery, error) {
	return domain.RetrievalQuery{Intent: task.Intent, RankTopK: 1}, nil
}

type fakeToolRetriever struct {
	candidates []domain.ToolCandidate
}

func (r fakeToolRetriever) Retrieve(_ context.Context, query domain.RetrievalQuery) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{
		Query:      query,
		Candidates: r.candidates,
		StageReports: []domain.RetrievalStageReport{
			{
				Stage:       domain.RetrievalStageSemanticRetrieval,
				InputCount:  2,
				OutputCount: len(r.candidates),
			},
		},
	}, nil
}

type fakeRanker struct{}

func (fakeRanker) Rank(_ context.Context, _ domain.RetrievalQuery, candidates []domain.ToolCandidate) ([]domain.ToolCandidate, error) {
	return []domain.ToolCandidate{candidates[1], candidates[0]}, nil
}

type fakeSelector struct{}

func (fakeSelector) Select(_ context.Context, _ domain.TaskSpec, candidates []domain.ToolCandidate) (domain.ToolCandidate, error) {
	return candidates[0], nil
}
