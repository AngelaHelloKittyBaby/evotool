package ports

import (
	"context"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// SimilarityDetector 检测源码或能力之间的相似度。
type SimilarityDetector interface {
	Detect(ctx context.Context, files []domain.SourceFile) ([]domain.SimilarityReport, error)
}
