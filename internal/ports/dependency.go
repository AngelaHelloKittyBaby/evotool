package ports

import (
	"context"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// DependencyResolver 解析 Tool 或 Library 的依赖关系和健康状态。
type DependencyResolver interface {
	ResolveTool(ctx context.Context, tool domain.Tool) (domain.DependencyGraph, error)
	ResolveLibrary(ctx context.Context, library domain.Library) (domain.DependencyGraph, error)
}
