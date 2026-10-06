package ports

import (
	"context"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// ToolValidator 验证生成工具是否满足 schema、测试、安全和依赖要求。
type ToolValidator interface {
	ValidateGeneratedTool(ctx context.Context, tool domain.GeneratedTool) (domain.ValidationResult, error)
}

// LibraryValidator 验证共享 Library 是否可复用。
type LibraryValidator interface {
	ValidateLibrary(ctx context.Context, library domain.Library) (domain.ValidationResult, error)
}
