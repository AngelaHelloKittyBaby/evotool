package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
	"github.com/AngelaHelloKittyBaby/evotool/internal/service"
	"os"
	"testing"
)

type checkingSandbox struct {
	calls int
	fail  bool
}

func (s *checkingSandbox) Run(_ context.Context, r ports.SandboxRequest) (ports.SandboxResult, error) {
	s.calls++
	if s.fail {
		return ports.SandboxResult{}, errors.New("sandbox test failed")
	}
	return ports.SandboxResult{Success: true, Output: `{"count":3,"total":60,"average":20}`}, nil
}
func generatedFixture(t *testing.T) domain.GeneratedTool {
	t.Helper()
	data, err := os.ReadFile("../../examples/csv-analysis/generated.json")
	if err != nil {
		t.Fatal(err)
	}
	var g domain.GeneratedTool
	if err = json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestValidationRejectsBeforeExecuting(t *testing.T) {
	cases := []struct {
		name   string
		change func(*domain.GeneratedTool)
	}{
		{"name", func(g *domain.GeneratedTool) { g.Manifest.Name = "helper" }},
		{"traversal", func(g *domain.GeneratedTool) { g.SourceFiles[0].Path = "../main.py" }},
		{"duplicate", func(g *domain.GeneratedTool) { g.Tests = append(g.Tests, g.SourceFiles[0]) }},
		{"network", func(g *domain.GeneratedTool) { g.Manifest.Permissions.Network = domain.NetworkPolicyFull }},
		{"host paths", func(g *domain.GeneratedTool) { g.Manifest.Permissions.AllowedPaths = []string{"/"} }},
		{"dependencies", func(g *domain.GeneratedTool) { g.Manifest.Dependencies = []domain.DependencyRef{{ID: "untrusted"}} }},
		{"missing tests", func(g *domain.GeneratedTool) { g.Tests = nil }},
		{"unsupported schema", func(g *domain.GeneratedTool) { g.Manifest.InputSchema["oneOf"] = []any{} }},
		{"resource limit", func(g *domain.GeneratedTool) { g.Manifest.ResourceLimits.MaxRuntimeMs = 0 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := generatedFixture(t)
			tc.change(&g)
			sandbox := &checkingSandbox{}
			r, err := (service.ValidationService{Sandbox: sandbox}).ValidateGeneratedTool(context.Background(), g)
			if err != nil || r.Valid || sandbox.calls != 0 {
				t.Fatalf("report=%+v error=%v calls=%d", r, err, sandbox.calls)
			}
		})
	}
}

func TestLearnSubmittedToolPublishesOnlyValidatedPackages(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "valid", true: "test failure"}[fail], func(t *testing.T) {
			ctx := context.Background()
			stores := filesystem.NewStores(t.TempDir())
			memory, err := service.NewCapabilityMemoryService(service.CapabilityMemoryServiceConfig{ToolStore: stores.Tools, LibraryStore: stores.Libraries, RegistryStore: stores.Registry, AuditLogger: stores.Audit})
			if err != nil {
				t.Fatal(err)
			}
			sandbox := &checkingSandbox{fail: fail}
			learning := service.LearningService{Memory: memory, Validator: service.ValidationService{Sandbox: sandbox}}
			g := generatedFixture(t)
			tool, report, err := learning.LearnSubmittedTool(ctx, g)
			if fail {
				if err == nil || report.Valid {
					t.Fatal("failed tests accepted")
				}
				if _, err = stores.Tools.Get(ctx, g.Manifest.Name); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("failed tool persisted")
				}
				return
			}
			if err != nil || !report.Valid || tool.LifecycleStatus != domain.LifecycleApproved {
				t.Fatalf("report=%+v err=%v", report, err)
			}
			if _, err = stores.Registry.GetDependencyGraph(ctx, tool.ID); err != nil {
				t.Fatal(err)
			}
			calls := sandbox.calls
			if _, _, err = learning.LearnSubmittedTool(ctx, g); err == nil {
				t.Fatal("existing tool overwritten")
			}
			if sandbox.calls != calls {
				t.Fatal("duplicate submission executed")
			}
		})
	}
}
