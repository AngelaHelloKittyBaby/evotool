package ports

import (
	"context"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// ToolExecutor 执行已批准的 Tool，并返回执行结果。
type ToolExecutor interface {
	Execute(ctx context.Context, tool domain.Tool, input map[string]any) (domain.ExecutionResult, error)
}

// Sandbox 在隔离边界内运行代码或测试。
type Sandbox interface {
	Run(ctx context.Context, request SandboxRequest) (SandboxResult, error)
}

// SandboxRequest 表示一次沙箱运行请求。
type SandboxRequest struct {
	Runtime  string
	Files    []domain.SourceFile
	Command  []string
	Input    map[string]any
	Policy   domain.ToolPolicy
	Metadata map[string]string
}

// SandboxResult 表示一次沙箱运行结果。
type SandboxResult struct {
	Success  bool
	Output   string
	Error    string
	Metadata map[string]string
}
