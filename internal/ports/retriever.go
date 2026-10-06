package ports

import (
	"context"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// ToolRetriever 根据任务意图和过滤条件检索候选 Tool。
type ToolRetriever interface {
	Search(ctx context.Context, query domain.RetrievalQuery) ([]domain.ToolCandidate, error)
}

// LibraryRetriever 根据生成工具的需求检索可复用 Library。
type LibraryRetriever interface {
	SearchLibraries(ctx context.Context, query domain.RetrievalQuery) ([]domain.LibraryCandidate, error)
}
