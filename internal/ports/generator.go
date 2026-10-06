package ports

import (
	"context"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// ToolGenerator 根据任务描述生成候选工具。
type ToolGenerator interface {
	Generate(ctx context.Context, task domain.TaskSpec) (domain.GeneratedTool, error)
}
