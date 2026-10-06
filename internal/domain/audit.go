package domain

import "time"

// AuditSubjectKind 表示审计事件关联的对象类型。
type AuditSubjectKind string

const (
	// AuditSubjectTool 表示审计对象是 Tool。
	AuditSubjectTool AuditSubjectKind = "tool"
	// AuditSubjectLibrary 表示审计对象是 Library。
	AuditSubjectLibrary AuditSubjectKind = "library"
	// AuditSubjectExecution 表示审计对象是一次执行。
	AuditSubjectExecution AuditSubjectKind = "execution"
	// AuditSubjectPolicy 表示审计对象是一次策略评估。
	AuditSubjectPolicy AuditSubjectKind = "policy"
	// AuditSubjectBenchmark 表示审计对象是一次 Benchmark。
	AuditSubjectBenchmark AuditSubjectKind = "benchmark"
)

// AuditEvent 表示 EvoTool 中一次可追踪的审计事件。
type AuditEvent struct {
	ID          string
	Action      string
	SubjectKind AuditSubjectKind
	SubjectID   string
	Actor       string
	Message     string
	Metadata    map[string]string
	CreatedAt   time.Time
}
