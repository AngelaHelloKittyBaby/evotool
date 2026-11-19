package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

// ValidationService 在沙箱运行测试之前检查提交上来的包。
type ValidationService struct {
	Sandbox   ports.Sandbox
	Libraries ports.LibrarySources
}

// ValidateGeneratedTool 校验 MVP 的 Python 契约；暂不支持安装依赖。
func (s ValidationService) ValidateGeneratedTool(ctx context.Context, g domain.GeneratedTool) (domain.ValidationResult, error) {
	r := domain.ValidationResult{}
	if err := checkContext(ctx); err != nil {
		return r, err
	}
	reject := func(msg string) (domain.ValidationResult, error) { r.Errors = append(r.Errors, msg); return r, nil }
	m := g.Manifest
	if err := domain.ValidateSchema(m.InputSchema); err != nil {
		return reject("input schema: " + err.Error())
	}
	if err := domain.ValidateSchema(m.OutputSchema); err != nil {
		return reject("output schema: " + err.Error())
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)+$`).MatchString(m.Name) || m.Name == "abc_tool" || m.Name == "main_tool" {
		return reject("semantic snake_case name required")
	}
	if strings.TrimSpace(m.Description) == "" || m.Category == "" || len(m.Capabilities) == 0 || m.InputSchema == nil || m.OutputSchema == nil || g.GenerationReason == "" {
		return reject("description, category, capabilities, schemas and generation reason required")
	}
	if m.Runtime != "python" || m.Environment.Network || (m.Environment.Runtime != "" && m.Environment.Runtime != "python") {
		return reject("only offline Python packages supported")
	}
	if len(m.Dependencies)+len(g.Dependencies) > 0 {
		return reject("dependency installation and library execution not supported in this adapter yet")
	}
	policy, err := NewPolicyService(PolicyServiceConfig{ResourceLimits: domain.ResourceLimits{MaxRuntimeMs: 30000, MaxMemoryBytes: 268435456, MaxOutputBytes: 1048576}})
	if err != nil {
		return r, err
	}
	decision, err := policy.Evaluate(ctx, domain.Tool{Manifest: m, CurrentVersion: "v1", LifecycleStatus: domain.LifecycleApproved, TrustLevel: domain.TrustValidated}, domain.TaskSpec{})
	if err != nil {
		return r, err
	}
	if !decision.Allowed {
		return reject(strings.Join(decision.Reasons, "; "))
	}
	files := append(append([]domain.SourceFile{}, g.SourceFiles...), g.Tests...)
	if g.Readme != "" {
		files = append(files, domain.SourceFile{Path: "README.md", Content: g.Readme})
	}
	if len(m.LibraryRefs) > 0 {
		if s.Libraries == nil {
			return reject("library resolver required")
		}
		for _, f := range files {
			if f.Path == "shared" || strings.HasPrefix(f.Path, "shared/") {
				return reject("shared namespace reserved for libraries")
			}
		}
		resolved, _, err := s.Libraries.Resolve(ctx, m.LibraryRefs)
		if err != nil {
			return reject(err.Error())
		}
		files = append(files, resolved...)
	}
	seen := map[string]bool{}
	size := 0
	for _, f := range files {
		if f.Path == "" || path.IsAbs(f.Path) || path.Clean(f.Path) != f.Path || strings.ContainsAny(f.Path, "\\:") || f.Path == ".." || strings.HasPrefix(f.Path, "../") || seen[f.Path] {
			return reject("unsafe or duplicate source path: " + f.Path)
		}
		seen[f.Path] = true
		size += len(f.Content)
	}
	if size > 524288 || len(files) > 100 || !seen[m.EntryPoint] || len(g.Tests) == 0 || len(m.Examples) == 0 {
		return reject("entry point, tests and examples required; package limit 512 KiB / 100 files")
	}
	if s.Sandbox == nil {
		return r, fmt.Errorf("%w: sandbox", ErrMissingDependency)
	}
	run := func(entry string, input map[string]any) (ports.SandboxResult, error) {
		return s.Sandbox.Run(ctx, ports.SandboxRequest{Runtime: m.Runtime, Files: files, Command: []string{entry}, Input: input, Policy: domain.ToolPolicy{Permissions: m.Permissions, ResourceLimits: m.ResourceLimits}})
	}
	for _, test := range g.Tests {
		tested, e := run(test.Path, map[string]any{})
		r.TestOutput += tested.Output
		if e != nil {
			r.Errors = append(r.Errors, e.Error())
			return r, fmt.Errorf("test %s: %w", test.Path, e)
		}
		if !tested.Success {
			return reject("test failed: " + test.Path)
		}
	}
	for _, example := range m.Examples {
		var input, expected, actual map[string]any
		if json.Unmarshal([]byte(example.Input), &input) != nil || input == nil || json.Unmarshal([]byte(example.Output), &expected) != nil || expected == nil {
			return reject("examples must contain JSON objects")
		}
		if err := domain.ValidateObject(m.InputSchema, input); err != nil {
			return reject(err.Error())
		}
		if err := domain.ValidateObject(m.OutputSchema, expected); err != nil {
			return reject(err.Error())
		}
		tested, e := run(m.EntryPoint, input)
		if e != nil {
			r.Errors = append(r.Errors, e.Error())
			return r, fmt.Errorf("example execution: %w", e)
		}
		if !tested.Success || json.Unmarshal([]byte(tested.Output), &actual) != nil || !reflect.DeepEqual(actual, expected) {
			return reject("example output mismatch")
		}
	}
	r.Valid = true
	r.SandboxReport = "tests and examples passed in configured sandbox"
	r.PolicyReport = "offline, no host permissions"
	r.DependencyReport = "no external dependencies"
	if len(m.LibraryRefs) > 0 {
		r.DependencyReport = "exact library versions and source digests checked"
	}
	return r, nil
}
