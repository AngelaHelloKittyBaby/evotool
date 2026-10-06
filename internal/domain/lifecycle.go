package domain

// LifecycleStatus 表示 Tool 或 Library 从生成到废弃的生命周期状态。
type LifecycleStatus string

const (
	// LifecycleDraft 表示刚生成，尚未验证。
	LifecycleDraft LifecycleStatus = "draft"
	// LifecycleValidated 表示验证通过，但尚未批准自动复用。
	LifecycleValidated LifecycleStatus = "validated"
	// LifecycleApproved 表示允许被自动检索和执行。
	LifecycleApproved LifecycleStatus = "approved"
	// LifecycleQuarantined 表示发现风险或多次失败，暂停使用。
	LifecycleQuarantined LifecycleStatus = "quarantined"
	// LifecycleDeprecated 表示已有更好版本，不再推荐。
	LifecycleDeprecated LifecycleStatus = "deprecated"
	// LifecycleDeleted 表示能力已被移除。
	LifecycleDeleted LifecycleStatus = "deleted"
)

// TrustLevel 表示 EvoTool 对一个能力的信任等级。
type TrustLevel string

const (
	// TrustUnknown 表示尚未评估信任等级。
	TrustUnknown TrustLevel = "unknown"
	// TrustUntrusted 表示能力尚不可信。
	TrustUntrusted TrustLevel = "untrusted"
	// TrustValidated 表示能力已通过基础验证。
	TrustValidated TrustLevel = "validated"
	// TrustReviewed 表示能力已通过人工或策略审查。
	TrustReviewed TrustLevel = "reviewed"
	// TrustTrusted 表示能力长期表现稳定，可作为可信能力优先复用。
	TrustTrusted TrustLevel = "trusted"
)
