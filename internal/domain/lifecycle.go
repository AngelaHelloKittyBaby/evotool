package domain

import "time"

// LifecycleStatus describes the lifecycle state of a Tool or Library.
type LifecycleStatus string

const (
	// LifecycleDraft means the capability has not been validated yet.
	LifecycleDraft LifecycleStatus = "draft"
	// LifecycleValidated means validation passed but automatic reuse is not approved yet.
	LifecycleValidated LifecycleStatus = "validated"
	// LifecycleApproved means the capability may be retrieved and executed.
	LifecycleApproved LifecycleStatus = "approved"
	// LifecycleActive means the capability is actively maintained and used.
	LifecycleActive LifecycleStatus = "active"
	// LifecycleStale means the capability has not been used recently.
	LifecycleStale LifecycleStatus = "stale"
	// LifecycleArchived means the capability is retained but excluded from default retrieval.
	LifecycleArchived LifecycleStatus = "archived"
	// LifecycleQuarantined means the capability is paused because of risk or repeated failures.
	LifecycleQuarantined LifecycleStatus = "quarantined"
	// LifecycleDeprecated means a better version or replacement is available.
	LifecycleDeprecated LifecycleStatus = "deprecated"
	// LifecycleDeleted means the capability has been removed from active storage.
	LifecycleDeleted LifecycleStatus = "deleted"
)

// TrustLevel describes how much EvoTool trusts a capability.
type TrustLevel string

const (
	// TrustUnknown means the capability has not been assessed.
	TrustUnknown TrustLevel = "unknown"
	// TrustUntrusted means the capability is not yet safe to reuse.
	TrustUntrusted TrustLevel = "untrusted"
	// TrustValidated means the capability passed baseline validation.
	TrustValidated TrustLevel = "validated"
	// TrustReviewed means the capability passed human or policy review.
	TrustReviewed TrustLevel = "reviewed"
	// TrustTrusted means the capability has a stable history and may be preferred.
	TrustTrusted TrustLevel = "trusted"
)

// LifecycleAction describes a recommendation produced by a lifecycle scan.
type LifecycleAction string

const (
	// LifecycleActionNone means no lifecycle change is recommended.
	LifecycleActionNone LifecycleAction = "none"
	// LifecycleActionMarkStale means the Tool should be marked stale.
	LifecycleActionMarkStale LifecycleAction = "mark_stale"
	// LifecycleActionArchiveCandidate means the Tool is ready for archive review.
	LifecycleActionArchiveCandidate LifecycleAction = "archive_candidate"
)

// LifecyclePolicy defines time thresholds for lifecycle scans.
type LifecyclePolicy struct {
	StaleAfter   time.Duration
	ArchiveAfter time.Duration
}

// LifecycleCandidate describes one Tool identified by a lifecycle scan.
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

// LifecycleScanReport summarizes a read-only lifecycle scan.
type LifecycleScanReport struct {
	EvaluatedAt time.Time
	TotalTools  int
	Candidates  []LifecycleCandidate
}
