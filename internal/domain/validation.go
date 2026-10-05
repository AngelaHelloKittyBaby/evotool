package domain

// ValidationResult 表示工具验证结果。
type ValidationResult struct {
	Valid         bool
	Errors        []string
	Warnings      []string
	TestOutput    string
	SandboxReport string
}
