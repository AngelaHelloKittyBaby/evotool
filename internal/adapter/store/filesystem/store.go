package filesystem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

const (
	// DefaultRoot 是 EvoTool 本地能力记忆的默认目录。
	DefaultRoot = ".evotool"
)

var safeSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// ToolStore 使用文件系统保存和读取 Tool。
type ToolStore struct {
	root string
}

// LibraryStore 使用文件系统保存和读取 Library。
type LibraryStore struct {
	root string
}

// RegistryStore 使用文件系统保存 Registry 元数据和依赖图。
type RegistryStore struct {
	root string
}

// ExecutionRecorder 使用 JSONL 文件记录工具执行历史。
type ExecutionRecorder struct {
	root string
}

// AuditLogger 使用 JSONL 文件记录审计事件。
type AuditLogger struct {
	root string
}

// Stores 聚合文件系统 Adapter 的所有存储实现。
type Stores struct {
	Tools      *ToolStore
	Libraries  *LibraryStore
	Registry   *RegistryStore
	Executions *ExecutionRecorder
	Audit      *AuditLogger
}

// NewStores 创建一组共享相同 root 的文件系统存储实现。
func NewStores(root string) Stores {
	root = normalizeRoot(root)
	return Stores{
		Tools:      NewToolStore(root),
		Libraries:  NewLibraryStore(root),
		Registry:   NewRegistryStore(root),
		Executions: NewExecutionRecorder(root),
		Audit:      NewAuditLogger(root),
	}
}

// NewToolStore 创建 ToolStore。
func NewToolStore(root string) *ToolStore {
	return &ToolStore{root: normalizeRoot(root)}
}

// NewLibraryStore 创建 LibraryStore。
func NewLibraryStore(root string) *LibraryStore {
	return &LibraryStore{root: normalizeRoot(root)}
}

// NewRegistryStore 创建 RegistryStore。
func NewRegistryStore(root string) *RegistryStore {
	return &RegistryStore{root: normalizeRoot(root)}
}

// NewExecutionRecorder 创建 ExecutionRecorder。
func NewExecutionRecorder(root string) *ExecutionRecorder {
	return &ExecutionRecorder{root: normalizeRoot(root)}
}

// NewAuditLogger 创建 AuditLogger。
func NewAuditLogger(root string) *AuditLogger {
	return &AuditLogger{root: normalizeRoot(root)}
}

// Save 保存 Tool 本体、manifest 和版本文件。
func (s *ToolStore) Save(ctx context.Context, tool domain.Tool) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	key, err := objectKey(tool.ID, tool.Name)
	if err != nil {
		return fmt.Errorf("tool key: %w", err)
	}

	dir := filepath.Join(s.root, "tools", key)
	if err := writeJSON(filepath.Join(dir, "manifest.json"), tool.Manifest); err != nil {
		return fmt.Errorf("write tool manifest: %w", err)
	}
	if err := writeJSON(filepath.Join(dir, "metadata.json"), tool); err != nil {
		return fmt.Errorf("write tool metadata: %w", err)
	}
	if err := writeToolVersions(filepath.Join(dir, "versions"), tool.Versions); err != nil {
		return fmt.Errorf("write tool versions: %w", err)
	}
	return nil
}

// Get 读取 Tool 本体。
func (s *ToolStore) Get(ctx context.Context, id string) (domain.Tool, error) {
	if err := checkContext(ctx); err != nil {
		return domain.Tool{}, err
	}
	key, err := safeSegment(id)
	if err != nil {
		return domain.Tool{}, fmt.Errorf("tool key: %w", err)
	}

	var tool domain.Tool
	if err := readJSON(filepath.Join(s.root, "tools", key, "metadata.json"), &tool); err != nil {
		return domain.Tool{}, fmt.Errorf("read tool metadata: %w", err)
	}
	return tool, nil
}

// Update 更新 Tool，本地文件系统实现与 Save 相同。
func (s *ToolStore) Update(ctx context.Context, tool domain.Tool) error {
	return s.Save(ctx, tool)
}

// Save 保存 Library 本体、manifest 和版本文件。
func (s *LibraryStore) Save(ctx context.Context, library domain.Library) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	key, err := objectKey(library.ID, library.Name)
	if err != nil {
		return fmt.Errorf("library key: %w", err)
	}

	dir := filepath.Join(s.root, "libraries", key)
	if err := writeJSON(filepath.Join(dir, "manifest.json"), library.Manifest); err != nil {
		return fmt.Errorf("write library manifest: %w", err)
	}
	if err := writeJSON(filepath.Join(dir, "metadata.json"), library); err != nil {
		return fmt.Errorf("write library metadata: %w", err)
	}
	if err := writeLibraryVersions(filepath.Join(dir, "versions"), library.Versions); err != nil {
		return fmt.Errorf("write library versions: %w", err)
	}
	return nil
}

// Get 读取 Library 本体。
func (s *LibraryStore) Get(ctx context.Context, id string) (domain.Library, error) {
	if err := checkContext(ctx); err != nil {
		return domain.Library{}, err
	}
	key, err := safeSegment(id)
	if err != nil {
		return domain.Library{}, fmt.Errorf("library key: %w", err)
	}

	var library domain.Library
	if err := readJSON(filepath.Join(s.root, "libraries", key, "metadata.json"), &library); err != nil {
		return domain.Library{}, fmt.Errorf("read library metadata: %w", err)
	}
	return library, nil
}

// Update 更新 Library，本地文件系统实现与 Save 相同。
func (s *LibraryStore) Update(ctx context.Context, library domain.Library) error {
	return s.Save(ctx, library)
}

// UpsertToolMetadata 保存 Tool 检索和治理所需元数据。
func (s *RegistryStore) UpsertToolMetadata(ctx context.Context, tool domain.Tool) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	key, err := objectKey(tool.ID, tool.Name)
	if err != nil {
		return fmt.Errorf("tool metadata key: %w", err)
	}
	return writeJSON(filepath.Join(s.root, "registry", "metadata", "tools", key+".json"), tool)
}

// UpsertLibraryMetadata 保存 Library 检索和治理所需元数据。
func (s *RegistryStore) UpsertLibraryMetadata(ctx context.Context, library domain.Library) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	key, err := objectKey(library.ID, library.Name)
	if err != nil {
		return fmt.Errorf("library metadata key: %w", err)
	}
	return writeJSON(filepath.Join(s.root, "registry", "metadata", "libraries", key+".json"), library)
}

// SaveDependencyGraph 保存依赖图。
func (s *RegistryStore) SaveDependencyGraph(ctx context.Context, graph domain.DependencyGraph) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	key, err := dependencyGraphKey(graph)
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(s.root, "registry", "dependency_graph", key+".json"), graph)
}

// GetDependencyGraph 读取依赖图。
func (s *RegistryStore) GetDependencyGraph(ctx context.Context, rootID string) (domain.DependencyGraph, error) {
	if err := checkContext(ctx); err != nil {
		return domain.DependencyGraph{}, err
	}
	key, err := safeSegment(rootID)
	if err != nil {
		return domain.DependencyGraph{}, fmt.Errorf("dependency graph key: %w", err)
	}

	var graph domain.DependencyGraph
	if err := readJSON(filepath.Join(s.root, "registry", "dependency_graph", key+".json"), &graph); err != nil {
		return domain.DependencyGraph{}, fmt.Errorf("read dependency graph: %w", err)
	}
	return graph, nil
}

// RecordExecution 追加记录一次执行结果。
func (r *ExecutionRecorder) RecordExecution(ctx context.Context, result domain.ExecutionResult) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	key, err := safeSegment(result.ToolID)
	if err != nil {
		return fmt.Errorf("execution key: %w", err)
	}
	return appendJSONLine(filepath.Join(r.root, "registry", "execution_history", key+".jsonl"), result)
}

// Log 追加记录一次审计事件。
func (l *AuditLogger) Log(ctx context.Context, event domain.AuditEvent) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	return appendJSONLine(filepath.Join(l.root, "registry", "audit", "events.jsonl"), event)
}

func normalizeRoot(root string) string {
	if strings.TrimSpace(root) == "" {
		return DefaultRoot
	}
	return root
}

func objectKey(id, name string) (string, error) {
	if strings.TrimSpace(name) != "" {
		return safeSegment(name)
	}
	return safeSegment(id)
}

func safeSegment(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("empty segment")
	}
	if !safeSegmentPattern.MatchString(value) {
		return "", fmt.Errorf("unsafe segment %q", value)
	}
	return value, nil
}

func safeRelativePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("empty relative path")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("absolute path %q is not allowed", value)
	}
	clean := filepath.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path traversal %q is not allowed", value)
	}
	return clean, nil
}

func writeToolVersions(root string, versions []domain.ToolVersion) error {
	for _, version := range versions {
		key, err := safeSegment(version.Version)
		if err != nil {
			return fmt.Errorf("version key: %w", err)
		}
		if err := writeSourceFiles(filepath.Join(root, key), version.Files); err != nil {
			return err
		}
	}
	return nil
}

func writeLibraryVersions(root string, versions []domain.LibraryVersion) error {
	for _, version := range versions {
		key, err := safeSegment(version.Version)
		if err != nil {
			return fmt.Errorf("version key: %w", err)
		}
		if err := writeSourceFiles(filepath.Join(root, key), version.Files); err != nil {
			return err
		}
	}
	return nil
}

func writeSourceFiles(root string, files []domain.SourceFile) error {
	for _, file := range files {
		rel, err := safeRelativePath(file.Path)
		if err != nil {
			return err
		}
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(file.Content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func dependencyGraphKey(graph domain.DependencyGraph) (string, error) {
	if len(graph.Nodes) == 0 {
		return "", errors.New("dependency graph has no root node")
	}
	root := graph.Nodes[0]
	if strings.TrimSpace(root.ID) != "" {
		return safeSegment(root.ID)
	}
	return safeSegment(root.Name)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func appendJSONLine(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
