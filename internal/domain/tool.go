package domain

import "time"

// Tool 表示 EvoTool 学到或注册的一个可复用工具能力。
type Tool struct {
	ID             string
	Name           string
	Description    string
	Manifest       ToolManifest
	CurrentVersion string
	Versions       []ToolVersion
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ToolManifest 描述工具的机器可读契约。
type ToolManifest struct {
	Name         string
	Description  string
	Runtime      string
	EntryPoint   string
	Inputs       []Parameter
	Outputs      []Parameter
	Capabilities []string
	Dependencies []string
	Tags         []string
}

// ToolVersion 表示工具的一次可执行版本。
type ToolVersion struct {
	Version   string
	Files     []SourceFile
	CreatedAt time.Time
}

// Parameter 描述工具的输入或输出参数。
type Parameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// SourceFile 表示一个生成工具中的源码、测试或文档文件。
type SourceFile struct {
	Path    string
	Content string
}
