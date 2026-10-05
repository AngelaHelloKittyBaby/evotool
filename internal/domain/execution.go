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
	Metadata   map[string]string
}
