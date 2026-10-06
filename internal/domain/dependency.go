package domain

// DependencyKind 表示依赖类型。
type DependencyKind string

const (
	// DependencyTool 表示依赖另一个 Tool。
	DependencyTool DependencyKind = "tool"
	// DependencyLibrary 表示依赖一个 Library。
	DependencyLibrary DependencyKind = "library"
	// DependencyPackage 表示依赖语言生态包，例如 Python package。
	DependencyPackage DependencyKind = "package"
	// DependencyRuntime 表示依赖运行时，例如 python、node、go。
	DependencyRuntime DependencyKind = "runtime"
	// DependencyMCP 表示依赖外部 MCP 能力。
	DependencyMCP DependencyKind = "mcp"
)

// DependencyRef 表示 Tool 或 Library 对另一个能力或外部包的依赖声明。
type DependencyRef struct {
	ID       string
	Name     string
	Kind     DependencyKind
	Version  string
	Optional bool
}

// DependencyHealthStatus 表示依赖图整体健康状态。
type DependencyHealthStatus string

const (
	// DependencyHealthy 表示依赖健康。
	DependencyHealthy DependencyHealthStatus = "healthy"
	// DependencyDegraded 表示依赖可用但存在风险或警告。
	DependencyDegraded DependencyHealthStatus = "degraded"
	// DependencyBroken 表示依赖不可用。
	DependencyBroken DependencyHealthStatus = "broken"
	// DependencyUnknown 表示依赖状态未知。
	DependencyUnknown DependencyHealthStatus = "unknown"
)

// DependencyGraph 表示 Tool 和 Library 之间的依赖关系。
type DependencyGraph struct {
	Nodes              []DependencyNode
	Edges              []DependencyEdge
	VersionConstraints []VersionConstraint
	HealthStatus       DependencyHealthStatus
}

// DependencyNode 表示依赖图中的一个节点。
type DependencyNode struct {
	ID      string
	Name    string
	Kind    DependencyKind
	Version string
}

// DependencyEdge 表示依赖图中的一条边。
type DependencyEdge struct {
	FromID string
	ToID   string
	Kind   DependencyKind
}

// VersionConstraint 表示依赖版本约束。
type VersionConstraint struct {
	DependencyID string
	Constraint   string
}
