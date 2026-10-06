package service

import (
	"context"
	"fmt"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

// RetrievalServiceConfig 描述 RetrievalService 的依赖。
type RetrievalServiceConfig struct {
	IntentExtractor ports.IntentExtractor
	ToolRetriever   ports.ToolRetriever
	Ranker          ports.CapabilityRanker
	Selector        ports.ToolSelector
}

// RetrievalService 编排 Intent Extraction、候选召回、能力排序和最终选择。
type RetrievalService struct {
	extractor ports.IntentExtractor
	retriever ports.ToolRetriever
	ranker    ports.CapabilityRanker
	selector  ports.ToolSelector
}

// NewRetrievalService 创建 RetrievalService。
func NewRetrievalService(config RetrievalServiceConfig) (*RetrievalService, error) {
	if config.IntentExtractor == nil {
		return nil, fmt.Errorf("%w: intent extractor", ErrMissingDependency)
	}
	if config.ToolRetriever == nil {
		return nil, fmt.Errorf("%w: tool retriever", ErrMissingDependency)
	}

	return &RetrievalService{
		extractor: config.IntentExtractor,
		retriever: config.ToolRetriever,
		ranker:    config.Ranker,
		selector:  config.Selector,
	}, nil
}

// Search 执行完整工具检索流水线。
func (s *RetrievalService) Search(ctx context.Context, task domain.TaskSpec) (domain.RetrievalResult, error) {
	query, err := s.extractor.Extract(ctx, task)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("extract intent: %w", err)
	}

	result, err := s.retriever.Retrieve(ctx, query)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("retrieve tools: %w", err)
	}
	result.Query = query
	result.StageReports = append([]domain.RetrievalStageReport{
		{
			Stage:       domain.RetrievalStageIntentExtraction,
			InputCount:  1,
			OutputCount: 1,
			Reason:      "task normalized into retrieval query",
		},
	}, result.StageReports...)

	if s.ranker != nil && len(result.Candidates) > 0 {
		inputCount := len(result.Candidates)
		ranked, err := s.ranker.Rank(ctx, query, result.Candidates)
		if err != nil {
			return domain.RetrievalResult{}, fmt.Errorf("rank candidates: %w", err)
		}
		result.Candidates = limitCandidates(ranked, query.RankTopK)
		result.StageReports = append(result.StageReports, domain.RetrievalStageReport{
			Stage:       domain.RetrievalStageCapabilityRank,
			InputCount:  inputCount,
			OutputCount: len(result.Candidates),
			Reason:      "ranked by semantic similarity and usage experience",
		})
	}

	if len(result.Candidates) == 0 {
		return result, nil
	}

	if s.selector != nil {
		selected, err := s.selector.Select(ctx, task, result.Candidates)
		if err != nil {
			return domain.RetrievalResult{}, fmt.Errorf("select tool: %w", err)
		}
		result.Selected = &selected
		result.StageReports = append(result.StageReports, domain.RetrievalStageReport{
			Stage:       domain.RetrievalStageLLMSelection,
			InputCount:  len(result.Candidates),
			OutputCount: 1,
			Reason:      "selected final tool from ranked candidates",
		})
		return result, nil
	}

	selected := result.Candidates[0]
	result.Selected = &selected
	return result, nil
}

func limitCandidates(candidates []domain.ToolCandidate, limit int) []domain.ToolCandidate {
	if limit <= 0 || len(candidates) <= limit {
		return candidates
	}
	return candidates[:limit]
}
