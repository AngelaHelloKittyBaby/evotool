package ports

import (
	"context"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// ToolStore 保存和读取 Tool 本体，不负责检索排序。
type ToolStore interface {
	Save(ctx context.Context, tool domain.Tool) error
	Get(ctx context.Context, id string) (domain.Tool, error)
	Update(ctx context.Context, tool domain.Tool) error
}

// LibraryStore 保存和读取 Library 本体，不负责检索排序。
type LibraryStore interface {
	Save(ctx context.Context, library domain.Library) error
	Get(ctx context.Context, id string) (domain.Library, error)
	Update(ctx context.Context, library domain.Library) error
}

// RegistryStore 保存能力目录、索引、依赖图和管理型元数据。
type RegistryStore interface {
	UpsertToolMetadata(ctx context.Context, tool domain.Tool) error
	UpsertLibraryMetadata(ctx context.Context, library domain.Library) error
	SaveDependencyGraph(ctx context.Context, graph domain.DependencyGraph) error
	GetDependencyGraph(ctx context.Context, rootID string) (domain.DependencyGraph, error)
}

// ExecutionRecorder 记录工具执行结果。
type ExecutionRecorder interface {
	RecordExecution(ctx context.Context, result domain.ExecutionResult) error
}

// AuditLogger 记录生成、验证、执行、审查、升级等审计事件。
type AuditLogger interface {
	Log(ctx context.Context, event domain.AuditEvent) error
}
