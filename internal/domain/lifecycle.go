package domain

import "time"

// LifecycleStatus 表示 Tool 或 Library 的生命周期状态。
type LifecycleStatus string

const (
	// LifecycleDraft 表示能力尚未验证。
	LifecycleDraft LifecycleStatus = "draft"
	// LifecycleValidated 表示验证通过，但尚未批准自动复用。
	LifecycleValidated LifecycleStatus = "validated"
	// LifecycleApproved 表示能力可以被检索和执行。
	LifecycleApproved LifecycleStatus = "approved"
	// LifecycleActive 表示能力仍在被积极维护和使用。
	LifecycleActive LifecycleStatus = "active"
	// LifecycleStale 表示能力较长时间未被使用。
	LifecycleStale LifecycleStatus = "stale"
	// LifecycleArchived 表示能力被保留，但不参与默认检索。
	LifecycleArchived LifecycleStatus = "archived"
	// LifecycleQuarantined 表示能力因风险或反复失败而暂停使用。
	LifecycleQuarantined LifecycleStatus = "quarantined"
	// LifecycleDeprecated 表示已有更好的版本或替代能力。
	LifecycleDeprecated LifecycleStatus = "deprecated"
	// LifecycleDeleted 表示能力已从活跃存储中移除。
	LifecycleDeleted LifecycleStatus = "deleted"
)

// TrustLevel 表示 EvoTool 对某项能力的信任程度。
type TrustLevel string

const (
	// TrustUnknown 表示能力尚未被评估。
	TrustUnknown TrustLevel = "unknown"
	// TrustUntrusted 表示能力还不能安全复用。
	TrustUntrusted TrustLevel = "untrusted"
	// TrustValidated 表示能力通过了基础验证。
	TrustValidated TrustLevel = "validated"
	// TrustReviewed 表示能力通过了人工或策略审查。
	TrustReviewed TrustLevel = "reviewed"
	// TrustTrusted 表示能力有稳定的历史表现，可以被优先选择。
	TrustTrusted TrustLevel = "trusted"
)

// LifecycleAction 表示生命周期扫描给出的建议动作。
type LifecycleAction string

const (
	// LifecycleActionNone 表示无需变更生命周期状态。
	LifecycleActionNone LifecycleAction = "none"
	// LifecycleActionMarkStale 表示应将该 Tool 标记为 stale。
	LifecycleActionMarkStale LifecycleAction = "mark_stale"
	// LifecycleActionArchiveCandidate 表示该 Tool 可以进入归档复查。
	LifecycleActionArchiveCandidate LifecycleAction = "archive_candidate"
)

// LifecyclePolicy 定义生命周期扫描的时间阈值。
type LifecyclePolicy struct {
	StaleAfter   time.Duration
	ArchiveAfter time.Duration
}

// LifecycleCandidate 描述生命周期扫描识别出的一个 Tool。
type LifecycleCandidate struct {
	ToolID         string
	ToolName       string
	CurrentStatus  LifecycleStatus
	ProposedStatus LifecycleStatus
	Action         LifecycleAction
	LastActivityAt time.Time
	IdleFor        time.Duration
	Reason         string
}

// LifecycleScanReport 汇总一次只读生命周期扫描的结果。
type LifecycleScanReport struct {
	EvaluatedAt time.Time
	TotalTools  int
	Candidates  []LifecycleCandidate
}
