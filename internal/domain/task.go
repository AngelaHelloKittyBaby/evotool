package domain

// TaskSpec 是用户任务或 Agent 任务的标准化描述。
type TaskSpec struct {
	Intent         string
	InputSummary   string
	Constraints    []string
	ExpectedOutput string
	Domain         string
	Metadata       map[string]string
}

// ToolCandidate 表示一次检索得到的候选工具及其匹配信息。
type ToolCandidate struct {
	Tool                  Tool
	Score                 float64
	MatchReason           string
	Source                RetrievalSource
	RiskSummary           string
	DependencyHealth      DependencyHealthStatus
	HistoricalSuccessRate float64
}

// GeneratedTool 表示刚生成、尚未进入可信工具记忆的新工具。
type GeneratedTool struct {
	Manifest          ToolManifest
	SourceFiles       []SourceFile
	Tests             []SourceFile
	Readme            string
	Dependencies      []DependencyRef
	LibraryCandidates []LibraryCandidate
	GenerationReason  string
}
