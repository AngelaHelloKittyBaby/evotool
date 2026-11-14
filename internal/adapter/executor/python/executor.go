// Package python 把沙箱适配为 Python 工具执行器。
package python

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

// Executor 把所有生成代码交由沙箱执行。
type Executor struct{ Sandbox ports.Sandbox }

// Execute 运行当前版本的原始代码，并要求输出 JSON 对象。
func (e Executor) Execute(ctx context.Context, t domain.Tool, input map[string]any) (domain.ExecutionResult, error) {
	if e.Sandbox == nil {
		return domain.ExecutionResult{}, fmt.Errorf("sandbox required")
	}
	if err := domain.ValidateObject(t.Manifest.InputSchema, input); err != nil {
		return domain.ExecutionResult{}, err
	}
	var files []domain.SourceFile
	for _, v := range t.Versions {
		if v.Version == t.CurrentVersion {
			files = v.Files
		}
	}
	if len(files) == 0 {
		return domain.ExecutionResult{}, fmt.Errorf("current version source missing")
	}
	r, err := e.Sandbox.Run(ctx, ports.SandboxRequest{Runtime: t.Manifest.Runtime, Files: files, Command: []string{t.Manifest.EntryPoint}, Input: input, Policy: domain.ToolPolicy{Permissions: t.Manifest.Permissions, ResourceLimits: t.Manifest.ResourceLimits}})
	result := domain.ExecutionResult{Success: r.Success, Error: r.Error}
	if err != nil {
		return result, err
	}
	if !r.Success {
		return result, fmt.Errorf("sandbox reported failure")
	}
	var output map[string]any
	if err = json.Unmarshal([]byte(r.Output), &output); err != nil || output == nil {
		result.Success = false
		return result, fmt.Errorf("output must be a JSON object")
	}
	result.Output = r.Output
	if err := domain.ValidateObject(t.Manifest.OutputSchema, output); err != nil {
		result.Success = false
		return result, err
	}
	return result, nil
}
