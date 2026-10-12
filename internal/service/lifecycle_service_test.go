package service

import (
	"context"
	"testing"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

func TestLifecycleServiceScanFindsStaleAndArchiveCandidates(t *testing.T) {
	root := t.TempDir()
	stores := filesystem.NewStores(root)
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	if err := stores.Tools.Save(context.Background(), lifecycleTestTool("stale_tool", domain.LifecycleApproved, now.Add(-100*24*time.Hour))); err != nil {
		t.Fatalf("save stale tool: %v", err)
	}
	if err := stores.Tools.Save(context.Background(), lifecycleTestTool("archived_candidate", domain.LifecycleStale, now.Add(-200*24*time.Hour))); err != nil {
		t.Fatalf("save archive candidate: %v", err)
	}
	if err := stores.Tools.Save(context.Background(), lifecycleTestTool("active_tool", domain.LifecycleActive, now.Add(-10*24*time.Hour))); err != nil {
		t.Fatalf("save active tool: %v", err)
	}

	service, err := NewLifecycleService(LifecycleServiceConfig{
		ToolStore: stores.Tools,
		Policy: domain.LifecyclePolicy{
			StaleAfter:   90 * 24 * time.Hour,
			ArchiveAfter: 180 * 24 * time.Hour,
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	report, err := service.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan lifecycle: %v", err)
	}
	if report.TotalTools != 3 {
		t.Fatalf("total tools = %d, want 3", report.TotalTools)
	}
	if len(report.Candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(report.Candidates))
	}
	if report.Candidates[0].ToolID != "archived_candidate" {
		t.Fatalf("first candidate = %q, want archived_candidate", report.Candidates[0].ToolID)
	}
	if report.Candidates[0].Action != domain.LifecycleActionArchiveCandidate {
		t.Fatalf("first action = %q, want archive_candidate", report.Candidates[0].Action)
	}
	if report.Candidates[1].ProposedStatus != domain.LifecycleStale {
		t.Fatalf("second proposed status = %q, want stale", report.Candidates[1].ProposedStatus)
	}
}

func TestLifecycleServiceScanUsesLastUsedAtBeforeMetadataDates(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	tool := lifecycleTestTool("recently_used", domain.LifecycleApproved, now.Add(-200*24*time.Hour))
	tool.UsageStats.LastUsedAt = now.Add(-2 * 24 * time.Hour)

	stores := filesystem.NewStores(t.TempDir())
	if err := stores.Tools.Save(context.Background(), tool); err != nil {
		t.Fatalf("save tool: %v", err)
	}
	service, err := NewLifecycleService(LifecycleServiceConfig{
		ToolStore: stores.Tools,
		Policy: domain.LifecyclePolicy{
			StaleAfter:   90 * 24 * time.Hour,
			ArchiveAfter: 180 * 24 * time.Hour,
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	report, err := service.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan lifecycle: %v", err)
	}
	if len(report.Candidates) != 0 {
		t.Fatalf("candidate count = %d, want 0", len(report.Candidates))
	}
}

func TestLifecycleServiceScanSkipsProtectedLifecycleStates(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	stores := filesystem.NewStores(t.TempDir())
	for _, status := range []domain.LifecycleStatus{
		domain.LifecycleDraft,
		domain.LifecycleValidated,
		domain.LifecycleArchived,
		domain.LifecycleQuarantined,
		domain.LifecycleDeprecated,
		domain.LifecycleDeleted,
	} {
		tool := lifecycleTestTool("tool_"+string(status), status, now.Add(-365*24*time.Hour))
		if err := stores.Tools.Save(context.Background(), tool); err != nil {
			t.Fatalf("save %s tool: %v", status, err)
		}
	}

	service, err := NewLifecycleService(LifecycleServiceConfig{
		ToolStore: stores.Tools,
		Policy: domain.LifecyclePolicy{
			StaleAfter:   90 * 24 * time.Hour,
			ArchiveAfter: 180 * 24 * time.Hour,
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	report, err := service.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan lifecycle: %v", err)
	}
	if len(report.Candidates) != 0 {
		t.Fatalf("candidate count = %d, want 0", len(report.Candidates))
	}
}

func TestLifecycleServiceRequiresValidThresholds(t *testing.T) {
	stores := filesystem.NewStores(t.TempDir())
	_, err := NewLifecycleService(LifecycleServiceConfig{
		ToolStore: stores.Tools,
		Policy: domain.LifecyclePolicy{
			StaleAfter:   180 * 24 * time.Hour,
			ArchiveAfter: 90 * 24 * time.Hour,
		},
	})
	if err == nil {
		t.Fatal("expected invalid threshold error")
	}
}

func lifecycleTestTool(id string, status domain.LifecycleStatus, updatedAt time.Time) domain.Tool {
	return domain.Tool{
		ID:              id,
		Name:            id,
		CurrentVersion:  "v1",
		LifecycleStatus: status,
		UpdatedAt:       updatedAt,
		CreatedAt:       updatedAt,
		Manifest:        domain.ToolManifest{Name: id, EntryPoint: "main.py"},
	}
}
