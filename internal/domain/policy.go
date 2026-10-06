package domain

import "time"

// NetworkPolicy 表示工具运行时的网络访问策略。
type NetworkPolicy string

const (
	// NetworkPolicyNone 表示不允许网络访问。
	NetworkPolicyNone NetworkPolicy = "none"
	// NetworkPolicyRestricted 表示只允许受限网络访问。
	NetworkPolicyRestricted NetworkPolicy = "restricted"
	// NetworkPolicyFull 表示允许完整网络访问。
	NetworkPolicyFull NetworkPolicy = "full"
)

// RiskLevel 表示 Tool 或 Library 的安全风险等级。
type RiskLevel string

const (
	// RiskLow 表示低风险能力。
	RiskLow RiskLevel = "low"
	// RiskMedium 表示中风险能力。
	RiskMedium RiskLevel = "medium"
	// RiskHigh 表示高风险能力。
	RiskHigh RiskLevel = "high"
	// RiskCritical 表示极高风险能力，默认必须拒绝或人工审查。
	RiskCritical RiskLevel = "critical"
)

// PermissionSet 表示 Tool 或 Library 声明的权限边界。
type PermissionSet struct {
	AllowedPaths    []string
	DeniedPaths     []string
	AllowedCommands []string
	DeniedCommands  []string
	Network         NetworkPolicy
	Environment     []string
}

// ResourceLimits 表示 Tool 或 Library 的资源限制。
type ResourceLimits struct {
	MaxRuntimeMs   int64
	MaxMemoryBytes int64
	MaxOutputBytes int64
}

// ToolPolicy 表示工具的权限、安全和资源限制策略。
type ToolPolicy struct {
	Permissions    PermissionSet
	ResourceLimits ResourceLimits
	RequiresReview bool
	ReviewReasons  []string
}

// PolicyDecision 表示一次安全策略评估结果。
type PolicyDecision struct {
	Allowed        bool
	RequiresReview bool
	RiskLevel      RiskLevel
	Reasons        []string
}

// ReviewResult 表示人工或策略审查结果。
type ReviewResult struct {
	Approved   bool
	Reviewer   string
	Reasons    []string
	ReviewedAt time.Time
}
