package domain

import "time"

// Tool 表示 EvoTool 学到或注册的一个可复用顶层能力。
type Tool struct {
	ID              string
	Name            string
	Description     string
	Category        string
	Manifest        ToolManifest
	CurrentVersion  string
	Versions        []ToolVersion
	Dependencies    []DependencyRef
	LifecycleStatus LifecycleStatus
	TrustLevel      TrustLevel
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ToolManifest 描述工具的机器可读契约。
type ToolManifest struct {
	Name           string
	Description    string
	Category       string
	Runtime        string
	EntryPoint     string
	Inputs         []Parameter
	Outputs        []Parameter
	Capabilities   []string
	Dependencies   []DependencyRef
	LibraryRefs    []DependencyRef
	Tags           []string
	Permissions    PermissionSet
	ResourceLimits ResourceLimits
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
