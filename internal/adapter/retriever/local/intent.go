package local

import (
	"context"
	"strings"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

const (
	defaultSemanticTopK = 50
	defaultFilterTopK   = 10
	defaultRankTopK     = 3
)

type IntentExtractor struct {
	SemanticTopK int
	FilterTopK   int
	RankTopK     int
	Filters      domain.RetrievalFilters
}

func (e IntentExtractor) Extract(ctx context.Context, task domain.TaskSpec) (domain.RetrievalQuery, error) {
	if err := checkContext(ctx); err != nil {
		return domain.RetrievalQuery{}, err
	}

	metadata := copyMetadata(task.Metadata)
	filters := e.Filters
	if runtime := strings.TrimSpace(metadata["runtime"]); runtime != "" {
		filters.Runtime = runtime
	}

	intent := joinNonEmpty(task.Intent, task.InputSummary, task.ExpectedOutput)
	if intent == "" {
		intent = strings.TrimSpace(metadata["query"])
	}

	query := domain.RetrievalQuery{
		Intent:       intent,
		Categories:   splitList(metadata["category"]),
		InputTypes:   splitList(metadata["input"]),
		OutputTypes:  splitList(metadata["output"]),
		Domain:       task.Domain,
		SemanticTopK: positiveOrDefault(e.SemanticTopK, defaultSemanticTopK),
		FilterTopK:   positiveOrDefault(e.FilterTopK, defaultFilterTopK),
		RankTopK:     positiveOrDefault(e.RankTopK, defaultRankTopK),
		Limit:        positiveOrDefault(e.RankTopK, defaultRankTopK),
		Filters:      filters,
		Metadata:     metadata,
	}
	return query, nil
}

func copyMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	copied := make(map[string]string, len(metadata))
	for key, value := range metadata {
		copied[key] = value
	}
	return copied
}

func joinNonEmpty(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, "\n")
}

func splitList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t'
	})
	items := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			items = append(items, field)
		}
	}
	return items
}

func positiveOrDefault(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
