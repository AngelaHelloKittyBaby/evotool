package domain

import "time"

// ExecutionResult 表示一次工具执行记录。
type ExecutionResult struct {
	ToolID     string
	Version    string
	Success    bool
	Output     string
	Error      string
	StartedAt  time.Time
	FinishedAt time.Time
	Duration   time.Duration
	TokenUsage TokenUsage
	Metadata   map[string]string
}

// TokenUsage 表示一次任务或工具执行产生的 token 用量。
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
