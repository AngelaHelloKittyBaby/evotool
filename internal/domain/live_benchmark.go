package domain

// LiveBenchmarkReport separates measured model/execution runs from suite replay.
type LiveBenchmarkReport struct {
	SuiteName        string
	TokenUsageSource string
	Baseline         LiveBenchmarkMode
	EvoTool          LiveBenchmarkMode
}

// LiveBenchmarkMode records failures as well as successful task results.
type LiveBenchmarkMode struct {
	BenchmarkResult
	MissingUsageCount int
	Tasks             []LiveBenchmarkTask
}

// LiveBenchmarkTask records provider usage even if validation or execution fails.
type LiveBenchmarkTask struct {
	Name               string
	ToolID             string
	Generated          bool
	Reused             bool
	Success            bool
	Generation         GenerationInfo
	Output             string
	Error              string
	PolicyBlocked      bool
	LibraryReusedCount int
}
