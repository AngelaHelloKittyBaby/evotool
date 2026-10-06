package domain

import "time"

// Library 表示多个 Tool 共享的底层能力或公共代码。
type Library struct {
	ID              string
	Name            string
	Description     string
	Category        string
	Manifest        LibraryManifest
	CurrentVersion  string
	Versions        []LibraryVersion
	Dependencies    []DependencyRef
	UsedByTools     []string
	LifecycleStatus LifecycleStatus
	TrustLevel      TrustLevel
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// LibraryManifest 描述 Library 的机器可读契约。
type LibraryManifest struct {
	Name           string
	Description    string
	Category       string
	Runtime        string
	Exports        []LibraryExport
	Dependencies   []DependencyRef
	Tags           []string
	Permissions    PermissionSet
	ResourceLimits ResourceLimits
}

// LibraryVersion 表示 Library 的一次可复用版本。
type LibraryVersion struct {
	Version   string
	Files     []SourceFile
	CreatedAt time.Time
}

// LibraryExport 描述 Library 对外暴露的函数、类型或模块。
type LibraryExport struct {
	Name        string
	Kind        string
	Description string
}

// LibraryCandidate 表示生成工具时发现的可复用 Library 候选项。
type LibraryCandidate struct {
	Library    Library
	Score      float64
	Reason     string
	Compatible bool
}
