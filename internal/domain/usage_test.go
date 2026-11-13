package domain

import (
	"testing"
	"time"
)

func TestApplyExecutionResultFoldsCountsRateAndLatency(t *testing.T) {
	base := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name   string
		stats  ToolUsageStats
		result ExecutionResult
		want   ToolUsageStats
	}{
		{
			name:   "first success",
			result: ExecutionResult{Success: true, Duration: 2 * time.Second, FinishedAt: base},
			want:   ToolUsageStats{UsageCount: 1, SuccessCount: 1, SuccessRate: 1, AverageLatency: 2 * time.Second, LastUsedAt: base},
		},
		{
			name:   "first failure",
			result: ExecutionResult{Success: false, Duration: time.Second, FinishedAt: base},
			want:   ToolUsageStats{UsageCount: 1, FailureCount: 1, SuccessRate: 0, AverageLatency: time.Second, LastUsedAt: base},
		},
		{
			name:   "success after failure",
			stats:  ToolUsageStats{UsageCount: 1, FailureCount: 1, SuccessRate: 0, AverageLatency: time.Second, LastUsedAt: base},
			result: ExecutionResult{Success: true, Duration: 3 * time.Second, FinishedAt: base.Add(time.Hour)},
			want:   ToolUsageStats{UsageCount: 2, SuccessCount: 1, FailureCount: 1, SuccessRate: 0.5, AverageLatency: 2 * time.Second, LastUsedAt: base.Add(time.Hour)},
		},
		{
			name:   "older result does not rewind last used",
			stats:  ToolUsageStats{UsageCount: 1, SuccessCount: 1, SuccessRate: 1, LastUsedAt: base},
			result: ExecutionResult{Success: true, Duration: 4 * time.Second, FinishedAt: base.Add(-time.Hour)},
			want:   ToolUsageStats{UsageCount: 2, SuccessCount: 2, SuccessRate: 1, AverageLatency: 2 * time.Second, LastUsedAt: base},
		},
		{
			name:   "zero duration keeps previous average",
			stats:  ToolUsageStats{UsageCount: 1, SuccessCount: 1, SuccessRate: 1, AverageLatency: 2 * time.Second, LastUsedAt: base},
			result: ExecutionResult{Success: true, FinishedAt: base.Add(time.Minute)},
			want:   ToolUsageStats{UsageCount: 2, SuccessCount: 2, SuccessRate: 1, AverageLatency: 2 * time.Second, LastUsedAt: base.Add(time.Minute)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyExecutionResult(tt.stats, tt.result)
			if got != tt.want {
				t.Fatalf("stats = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDefaultRetrievableStatusesExcludeInactiveCapabilities(t *testing.T) {
	allowed := make(map[LifecycleStatus]bool)
	for _, status := range DefaultRetrievableStatuses() {
		allowed[status] = true
	}
	for _, status := range []LifecycleStatus{LifecycleApproved, LifecycleActive, LifecycleStale} {
		if !allowed[status] {
			t.Fatalf("%s must be retrievable by default", status)
		}
	}
	for _, status := range []LifecycleStatus{
		LifecycleDraft, LifecycleValidated, LifecycleArchived, LifecycleQuarantined, LifecycleDeprecated, LifecycleDeleted,
	} {
		if allowed[status] {
			t.Fatalf("%s must not be retrievable by default", status)
		}
	}
}
