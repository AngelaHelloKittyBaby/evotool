package domain

// DependencyKind 表示能力所引用的依赖类型。
type DependencyKind string

const (
	// DependencyTool 表示能力依赖另一个 Tool。
	DependencyTool DependencyKind = "tool"
	// DependencyLibrary 表示能力依赖一个共享 Library。
	DependencyLibrary DependencyKind = "library"
	// DependencyPackage 表示能力依赖某个语言包。
	DependencyPackage DependencyKind = "package"
	// DependencyRuntime 表示能力依赖 python、node、go 等运行时。
	DependencyRuntime DependencyKind = "runtime"
	// DependencyMCP 表示能力依赖外部 MCP 能力。
	DependencyMCP DependencyKind = "mcp"
)

// DependencyRef 声明对另一个能力或外部包的依赖。
type DependencyRef struct {
	ID       string
	Name     string
	Kind     DependencyKind
	Version  string
	Optional bool
	// Digest binds an executable library reference to validated source contents.
	Digest string
}

// DependencyHealthStatus 表示依赖图的整体健康状态。
type DependencyHealthStatus string

const (
	// DependencyHealthy 表示所有必需依赖都可用。
	DependencyHealthy DependencyHealthStatus = "healthy"
	// DependencyDegraded 表示依赖可用但存在告警。
	DependencyDegraded DependencyHealthStatus = "degraded"
	// DependencyBroken 表示至少有一个必需依赖不可用。
	DependencyBroken DependencyHealthStatus = "broken"
	// DependencyUnknown 表示依赖健康状态尚未检查。
	DependencyUnknown DependencyHealthStatus = "unknown"
)

// DependencyGraph 表示 Tool、Library 与外部能力之间的依赖关系。
type DependencyGraph struct {
	Nodes              []DependencyNode
	Edges              []DependencyEdge
	VersionConstraints []VersionConstraint
	HealthStatus       DependencyHealthStatus
}

// DependencyNode 是依赖图中的一个节点。
type DependencyNode struct {
	ID      string
	Name    string
	Kind    DependencyKind
	Version string
}

// DependencyEdge 是依赖图中的一条边。
type DependencyEdge struct {
	FromID string
	ToID   string
	Kind   DependencyKind
}

// VersionConstraint 记录某个依赖期望的版本。
type VersionConstraint struct {
	DependencyID string
	Constraint   string
}

// DependencyIssue 描述依赖检查中发现的问题或告警。
type DependencyIssue struct {
	DependencyID string
	Kind         DependencyKind
	Status       DependencyHealthStatus
	Message      string
}

// DependencyCheckResult 汇总一张依赖图的健康检查结果。
type DependencyCheckResult struct {
	Graph   DependencyGraph
	Healthy bool
	Issues  []DependencyIssue
}
