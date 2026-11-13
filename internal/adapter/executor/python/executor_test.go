package python

import (
	"context"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
	"testing"
)

type fakeSandbox struct {
	calls  int
	output string
}

func (s *fakeSandbox) Run(context.Context, ports.SandboxRequest) (ports.SandboxResult, error) {
	s.calls++
	return ports.SandboxResult{Success: true, Output: s.output}, nil
}
func TestExecutorChecksSchemas(t *testing.T) {
	schema := map[string]any{"type": "object", "required": []string{"text"}, "properties": map[string]any{"text": map[string]any{"type": "string"}}}
	tool := domain.Tool{CurrentVersion: "v1", Versions: []domain.ToolVersion{{Version: "v1", Files: []domain.SourceFile{{Path: "main.py"}}}}, Manifest: domain.ToolManifest{Runtime: "python", EntryPoint: "main.py", InputSchema: schema, OutputSchema: schema}}
	for _, tc := range []struct {
		name, output string
		input        map[string]any
		wantError    bool
		calls        int
	}{
		{"success", `{"text":"ok"}`, map[string]any{"text": "input"}, false, 1},
		{"missing input", `{}`, map[string]any{}, true, 0},
		{"wrong output type", `{"text":5}`, map[string]any{"text": "input"}, true, 1},
		{"invalid JSON", `not json`, map[string]any{"text": "input"}, true, 1},
		{"null output", `null`, map[string]any{"text": "input"}, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sandbox := &fakeSandbox{output: tc.output}
			r, err := (Executor{Sandbox: sandbox}).Execute(context.Background(), tool, tc.input)
			if (err != nil) != tc.wantError || sandbox.calls != tc.calls || (!tc.wantError && !r.Success) {
				t.Fatalf("result=%+v err=%v calls=%d", r, err, sandbox.calls)
			}
		})
	}
}
