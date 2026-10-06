package ports

import (
	"context"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// BenchmarkRunner 运行 BenchmarkSuite 并生成评估结果。
type BenchmarkRunner interface {
	Run(ctx context.Context, suite domain.BenchmarkSuite) (domain.BenchmarkResult, error)
}

// BenchmarkRecorder 保存 Benchmark 结果，便于文档和报告复现。
type BenchmarkRecorder interface {
	RecordBenchmark(ctx context.Context, result domain.BenchmarkResult) error
}
