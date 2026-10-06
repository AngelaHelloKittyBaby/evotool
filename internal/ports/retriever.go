package ports

import (
	"context"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// IntentExtractor 从原始任务中提取检索需要的意图、输入输出和约束。
type IntentExtractor interface {
	Extract(ctx context.Context, task domain.TaskSpec) (domain.RetrievalQuery, error)
}

// ToolRetriever 执行工具候选召回，不能直接把全部工具交给 LLM。
type ToolRetriever interface {
	Retrieve(ctx context.Context, query domain.RetrievalQuery) (domain.RetrievalResult, error)
}

// CapabilityRanker 根据语义相似度、使用经验、可靠性、延迟和环境匹配对候选工具排序。
type CapabilityRanker interface {
	Rank(ctx context.Context, query domain.RetrievalQuery, candidates []domain.ToolCandidate) ([]domain.ToolCandidate, error)
}

// ToolSelector 从少量高质量候选中选择最终工具，可由规则或 LLM 实现。
type ToolSelector interface {
	Select(ctx context.Context, task domain.TaskSpec, candidates []domain.ToolCandidate) (domain.ToolCandidate, error)
}

// LibraryRetriever 根据生成工具的需求检索可复用 Library。
type LibraryRetriever interface {
	SearchLibraries(ctx context.Context, query domain.RetrievalQuery) ([]domain.LibraryCandidate, error)
}
