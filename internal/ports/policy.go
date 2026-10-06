package ports

import (
	"context"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// PolicyEngine 评估某个 Tool 在指定任务下是否允许执行。
type PolicyEngine interface {
	Evaluate(ctx context.Context, tool domain.Tool, task domain.TaskSpec) (domain.PolicyDecision, error)
}

// ToolReviewer 对高风险生成工具执行人工或策略审查。
type ToolReviewer interface {
	Review(ctx context.Context, tool domain.GeneratedTool) (domain.ReviewResult, error)
}
