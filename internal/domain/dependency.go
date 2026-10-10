package domain

// DependencyKind describes the kind of dependency referenced by a capability.
type DependencyKind string

const (
	// DependencyTool means a capability depends on another Tool.
	DependencyTool DependencyKind = "tool"
	// DependencyLibrary means a capability depends on a shared Library.
	DependencyLibrary DependencyKind = "library"
	// DependencyPackage means a capability depends on a language package.
	DependencyPackage DependencyKind = "package"
	// DependencyRuntime means a capability depends on a runtime such as python, node, or go.
	DependencyRuntime DependencyKind = "runtime"
	// DependencyMCP means a capability depends on an external MCP capability.
	DependencyMCP DependencyKind = "mcp"
)

// DependencyRef declares a dependency on another capability or external package.
type DependencyRef struct {
	ID       string
	Name     string
	Kind     DependencyKind
	Version  string
	Optional bool
}

// DependencyHealthStatus describes the overall health of a dependency graph.
type DependencyHealthStatus string

const (
	// DependencyHealthy means all required dependencies are available.
	DependencyHealthy DependencyHealthStatus = "healthy"
	// DependencyDegraded means dependencies are available but have warnings.
	DependencyDegraded DependencyHealthStatus = "degraded"
	// DependencyBroken means at least one required dependency is unavailable.
	DependencyBroken DependencyHealthStatus = "broken"
	// DependencyUnknown means dependency health has not been checked.
	DependencyUnknown DependencyHealthStatus = "unknown"
)

// DependencyGraph represents dependencies between Tools, Libraries, and external capabilities.
type DependencyGraph struct {
	Nodes              []DependencyNode
	Edges              []DependencyEdge
	VersionConstraints []VersionConstraint
	HealthStatus       DependencyHealthStatus
}

// DependencyNode is a node in a dependency graph.
type DependencyNode struct {
	ID      string
	Name    string
	Kind    DependencyKind
	Version string
}

// DependencyEdge is an edge in a dependency graph.
type DependencyEdge struct {
	FromID string
	ToID   string
	Kind   DependencyKind
}

// VersionConstraint records the expected version for a dependency.
type VersionConstraint struct {
	DependencyID string
	Constraint   string
}

// DependencyIssue describes a problem or warning discovered during dependency checking.
type DependencyIssue struct {
	DependencyID string
	Kind         DependencyKind
	Status       DependencyHealthStatus
	Message      string
}

// DependencyCheckResult summarizes the dependency health check for a graph.
type DependencyCheckResult struct {
	Graph   DependencyGraph
	Healthy bool
	Issues  []DependencyIssue
}
