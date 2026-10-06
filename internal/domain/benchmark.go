package domain

import "time"

// BenchmarkSuite 表示一组用于评估 EvoTool 效果的任务集合。
type BenchmarkSuite struct {
	Name        string
	Description string
	Tasks       []TaskSpec
	Metadata    map[string]string
}

// BenchmarkResult 表示 EvoTool 对任务成本和复用效果的评估结果。
type BenchmarkResult struct {
	SuiteName                string
	TotalTasks               int
	ToolGeneratedCount       int
	ToolReusedCount          int
	LibraryReusedCount       int
	DuplicateGenerationCount int
	AverageLatency           time.Duration
	AverageTokenUsage        TokenUsage
	SuccessRate              float64
	FailureRate              float64
	SafetyPolicyBlockCount   int
	DependencyFailureCount   int
}
