package service

import (
	"context"
	"fmt"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// PolicyServiceConfig defines the maximum resources permitted by the host.
type PolicyServiceConfig struct {
	ResourceLimits domain.ResourceLimits
}

// PolicyService checks execution eligibility without running generated code.
// Its decisions do not replace enforcement by a sandbox.
type PolicyService struct {
	limits domain.ResourceLimits
}

// NewPolicyService creates a fail-closed policy engine with explicit host limits.
func NewPolicyService(config PolicyServiceConfig) (*PolicyService, error) {
	l := config.ResourceLimits
	if l.MaxRuntimeMs <= 0 || l.MaxMemoryBytes <= 0 || l.MaxOutputBytes <= 0 {
		return nil, fmt.Errorf("host runtime, memory, and output limits must be positive")
	}
	return &PolicyService{limits: l}, nil
}

// Evaluate implements ports.PolicyEngine. Privileged declarations require review
// even when the Tool already has a reviewed trust label: the model does not yet
// carry a review bound to a particular version and permission set.
func (s *PolicyService) Evaluate(ctx context.Context, tool domain.Tool, _ domain.TaskSpec) (domain.PolicyDecision, error) {
	if err := checkContext(ctx); err != nil {
		return domain.PolicyDecision{}, err
	}
	d := domain.PolicyDecision{Allowed: true, RiskLevel: domain.RiskLow}
	deny := func(reason string) {
		d.Allowed = false
		d.RiskLevel = domain.RiskHigh
		d.Reasons = append(d.Reasons, reason)
	}
	switch tool.LifecycleStatus {
	case domain.LifecycleApproved, domain.LifecycleActive, domain.LifecycleStale:
	default:
		deny("tool lifecycle does not permit execution")
	}
	switch tool.TrustLevel {
	case domain.TrustValidated, domain.TrustReviewed, domain.TrustTrusted:
	default:
		deny("tool has not passed validation")
	}
	if tool.CurrentVersion == "" {
		deny("current version is required")
	}
	l := tool.Manifest.ResourceLimits
	if l.MaxRuntimeMs <= 0 || l.MaxMemoryBytes <= 0 || l.MaxOutputBytes <= 0 {
		deny("tool runtime, memory, and output limits must be positive")
	}
	if l.MaxRuntimeMs > s.limits.MaxRuntimeMs || l.MaxMemoryBytes > s.limits.MaxMemoryBytes || l.MaxOutputBytes > s.limits.MaxOutputBytes {
		deny("tool resource limits exceed host policy")
	}
	p := tool.Manifest.Permissions
	switch p.Network {
	case "", domain.NetworkPolicyNone:
	case domain.NetworkPolicyRestricted, domain.NetworkPolicyFull:
		d.RequiresReview = true
		d.Reasons = append(d.Reasons, "network access requires review")
	default:
		deny("unknown network policy")
	}
	if tool.Manifest.Environment.Network && (p.Network == "" || p.Network == domain.NetworkPolicyNone) {
		deny("runtime network requirement conflicts with declared permissions")
	}
	if len(p.AllowedPaths) > 0 {
		d.RequiresReview = true
		d.Reasons = append(d.Reasons, "host filesystem access requires review")
	}
	if len(p.AllowedCommands) > 0 {
		d.RequiresReview = true
		d.Reasons = append(d.Reasons, "system command access requires review")
	}
	if len(p.Environment) > 0 {
		d.RequiresReview = true
		d.Reasons = append(d.Reasons, "host environment access requires review")
	}
	if d.RequiresReview {
		d.Allowed = false
		d.RiskLevel = domain.RiskHigh
	}
	return d, nil
}
