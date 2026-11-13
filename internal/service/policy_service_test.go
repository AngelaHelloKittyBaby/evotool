package service

import (
	"context"
	"errors"
	"testing"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

var _ ports.PolicyEngine = (*PolicyService)(nil)

func TestPolicyServiceEvaluate(t *testing.T) {
	limits := domain.ResourceLimits{MaxRuntimeMs: 1000, MaxMemoryBytes: 1024, MaxOutputBytes: 512}
	s, err := NewPolicyService(PolicyServiceConfig{ResourceLimits: limits})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name            string
		edit            func(*domain.Tool)
		allowed, review bool
	}{
		{"validated approved tool", func(*domain.Tool) {}, true, false},
		{"stale remains usable", func(v *domain.Tool) { v.LifecycleStatus = domain.LifecycleStale }, true, false},
		{"draft", func(v *domain.Tool) { v.LifecycleStatus = domain.LifecycleDraft }, false, false},
		{"archived", func(v *domain.Tool) { v.LifecycleStatus = domain.LifecycleArchived }, false, false},
		{"quarantined", func(v *domain.Tool) { v.LifecycleStatus = domain.LifecycleQuarantined }, false, false},
		{"unknown lifecycle", func(v *domain.Tool) { v.LifecycleStatus = "future" }, false, false},
		{"untrusted", func(v *domain.Tool) { v.TrustLevel = domain.TrustUntrusted }, false, false},
		{"missing version", func(v *domain.Tool) { v.CurrentVersion = "" }, false, false},
		{"missing limits", func(v *domain.Tool) { v.Manifest.ResourceLimits.MaxRuntimeMs = 0 }, false, false},
		{"excess runtime", func(v *domain.Tool) { v.Manifest.ResourceLimits.MaxRuntimeMs++ }, false, false},
		{"excess memory", func(v *domain.Tool) { v.Manifest.ResourceLimits.MaxMemoryBytes++ }, false, false},
		{"excess output", func(v *domain.Tool) { v.Manifest.ResourceLimits.MaxOutputBytes++ }, false, false},
		{"network", func(v *domain.Tool) { v.Manifest.Permissions.Network = domain.NetworkPolicyRestricted }, false, true},
		{"unknown network", func(v *domain.Tool) { v.Manifest.Permissions.Network = "future" }, false, false},
		{"network mismatch", func(v *domain.Tool) { v.Manifest.Environment.Network = true }, false, false},
		{"filesystem", func(v *domain.Tool) { v.Manifest.Permissions.AllowedPaths = []string{"data"} }, false, true},
		{"commands", func(v *domain.Tool) { v.Manifest.Permissions.AllowedCommands = []string{"python"} }, false, true},
		{"secrets", func(v *domain.Tool) { v.Manifest.Permissions.Environment = []string{"API_KEY"} }, false, true},
		{"trust label cannot bypass review", func(v *domain.Tool) {
			v.TrustLevel = domain.TrustTrusted
			v.Manifest.Permissions.Network = domain.NetworkPolicyFull
		}, false, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			tool := domain.Tool{CurrentVersion: "v1", LifecycleStatus: domain.LifecycleApproved, TrustLevel: domain.TrustValidated,
				Manifest: domain.ToolManifest{ResourceLimits: limits}}
			tt.edit(&tool)
			d, err := s.Evaluate(context.Background(), tool, domain.TaskSpec{})
			if err != nil {
				t.Fatal(err)
			}
			if d.Allowed != tt.allowed || d.RequiresReview != tt.review {
				t.Fatalf("unexpected decision: %+v", d)
			}
			if !d.Allowed && (len(d.Reasons) == 0 || d.RiskLevel != domain.RiskHigh) {
				t.Fatalf("denial lacks explanation: %+v", d)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Evaluate(ctx, domain.Tool{}, domain.TaskSpec{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestPolicyServiceRequiresHostLimits(t *testing.T) {
	for _, limits := range []domain.ResourceLimits{{}, {MaxRuntimeMs: -1, MaxMemoryBytes: 1, MaxOutputBytes: 1}, {MaxRuntimeMs: 1, MaxOutputBytes: 1}, {MaxRuntimeMs: 1, MaxMemoryBytes: 1}} {
		if _, err := NewPolicyService(PolicyServiceConfig{ResourceLimits: limits}); err == nil {
			t.Fatal("expected invalid host limits to fail")
		}
	}
}
