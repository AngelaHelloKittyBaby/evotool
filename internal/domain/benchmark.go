package domain

import "time"

// BenchmarkSuite 表示一组用于评估 EvoTool 效果的任务集合。
type BenchmarkSuite struct {
	Name        string
	Description string
	Tasks       []BenchmarkTask
	Metadata    map[string]string
}

// BenchmarkTask 是一个基准任务，以及 Agent 在缺少可复用能力时本该生成的包。
// EvoTool 不内置大模型，因此生成产物由任务固定提供；基准度量的是
// “有多少次生成被已有能力替代”，而不是模型生成质量。
type BenchmarkTask struct {
	Name       string
	Spec       TaskSpec
	Package    GeneratedTool
	TokenUsage TokenUsage
	Input      map[string]any
}

// BenchmarkMode 表示基准的一次运行模式。
type BenchmarkMode string

const (
	// BenchmarkModeBaseline 表示没有能力记忆，每个任务都重新生成工具。
	BenchmarkModeBaseline BenchmarkMode = "baseline"
	// BenchmarkModeEvoTool 表示先检索能力记忆，未命中才生成工具。
	BenchmarkModeEvoTool BenchmarkMode = "evotool"
)

// BenchmarkResult 表示 EvoTool 对任务成本和复用效果的评估结果。
// TokenUsage 来自任务提供的生成成本，不是实测的模型调用用量。
type BenchmarkResult struct {
	Mode                     BenchmarkMode
	SuiteName                string
	TotalTasks               int
	ToolGeneratedCount       int
	ToolReusedCount          int
	LibraryReusedCount       int
	DuplicateGenerationCount int
	AverageLatency           time.Duration
	AverageTokenUsage        TokenUsage
	TotalTokenUsage          TokenUsage
	SuccessRate              float64
	FailureRate              float64
	SafetyPolicyBlockCount   int
	DependencyFailureCount   int
}

// BenchmarkReport 比较同一套任务在两种模式下的表现。
type BenchmarkReport struct {
	SuiteName string
	Baseline  BenchmarkResult
	EvoTool   BenchmarkResult
}

// GenerationSaved 返回 EvoTool 相比 baseline 少生成的能力数量。
// 负数表示 EvoTool 反而多生成了，调用方不应假设它一定为正。
func (r BenchmarkReport) GenerationSaved() int {
	return r.Baseline.ToolGeneratedCount - r.EvoTool.ToolGeneratedCount
}

// DuplicateGenerationSaved 返回 EvoTool 相比 baseline 少做的重复生成数量。
func (r BenchmarkReport) DuplicateGenerationSaved() int {
	return r.Baseline.DuplicateGenerationCount - r.EvoTool.DuplicateGenerationCount
}

// TokenSaved 返回 EvoTool 相比 baseline 节省的生成 token。
func (r BenchmarkReport) TokenSaved() TokenUsage {
	return TokenUsage{
		PromptTokens:     r.Baseline.TotalTokenUsage.PromptTokens - r.EvoTool.TotalTokenUsage.PromptTokens,
		CompletionTokens: r.Baseline.TotalTokenUsage.CompletionTokens - r.EvoTool.TotalTokenUsage.CompletionTokens,
		TotalTokens:      r.Baseline.TotalTokenUsage.TotalTokens - r.EvoTool.TotalTokenUsage.TotalTokens,
	}
}
