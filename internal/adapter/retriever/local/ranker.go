package local

import (
	"context"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

type CapabilityRanker struct{}

func NewCapabilityRanker() CapabilityRanker {
	return CapabilityRanker{}
}

func (CapabilityRanker) Rank(ctx context.Context, query domain.RetrievalQuery, candidates []domain.ToolCandidate) ([]domain.ToolCandidate, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	ranked := make([]domain.ToolCandidate, len(candidates))
	copy(ranked, candidates)
	for i := range ranked {
		ranked[i] = scoreCandidate(query, ranked[i])
	}
	sortCandidates(ranked)
	return ranked, nil
}

func scoreCandidate(query domain.RetrievalQuery, candidate domain.ToolCandidate) domain.ToolCandidate {
	semantic := candidate.RankScore.SemanticSimilarity
	if semantic == 0 {
		semantic = candidate.Score
	}
	success := successRate(candidate.UsageStats)
	reliability := reliabilityScore(candidate.UsageStats)
	latency := latencyScore(candidate.UsageStats.AverageLatency)
	usage := usageScore(candidate.UsageStats.UsageCount)
	environment := candidate.RankScore.EnvironmentMatch
	if environment == 0 {
		environment = environmentMatch(query, candidate.Tool)
	}

	final := 0.55*semantic + 0.15*success + 0.10*reliability + 0.10*latency + 0.05*usage + 0.05*environment
	candidate.RankScore = domain.CapabilityRankScore{
		SemanticSimilarity: semantic,
		SuccessRate:        success,
		Reliability:        reliability,
		Latency:            latency,
		UsageHistory:       usage,
		EnvironmentMatch:   environment,
		FinalScore:         final,
	}
	candidate.Score = final
	return candidate
}

func successRate(stats domain.ToolUsageStats) float64 {
	if stats.SuccessRate > 0 {
		return clamp01(stats.SuccessRate)
	}
	if stats.UsageCount <= 0 {
		return 0.5
	}
	return clamp01(float64(stats.SuccessCount) / float64(stats.UsageCount))
}

func reliabilityScore(stats domain.ToolUsageStats) float64 {
	if stats.UsageCount <= 0 {
		return 0
	}
	confidence := clamp01(float64(stats.UsageCount) / 20)
	return successRate(stats) * confidence
}

func latencyScore(latency time.Duration) float64 {
	if latency <= 0 {
		return 0.5
	}
	seconds := latency.Seconds()
	return clamp01(1 / (1 + seconds))
}

func usageScore(count int) float64 {
	if count <= 0 {
		return 0
	}
	return clamp01(float64(count) / 100)
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
