package service_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	pythonexecutor "github.com/AngelaHelloKittyBaby/evotool/internal/adapter/executor/python"
	dockersandbox "github.com/AngelaHelloKittyBaby/evotool/internal/adapter/sandbox/docker"
	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
	"github.com/AngelaHelloKittyBaby/evotool/internal/service"
)

func libraryFixture() domain.Library {
	return domain.Library{ID: "numeric_core", Name: "numeric_core", CurrentVersion: "v1", LifecycleStatus: domain.LifecycleApproved, TrustLevel: domain.TrustValidated, Manifest: domain.LibraryManifest{Name: "numeric_core", Runtime: "python"}, Versions: []domain.LibraryVersion{{Version: "v1", Files: []domain.SourceFile{{Path: "source/stats.py", Content: "def total(values):\n    return sum(values)\n"}}}}}
}

func TestLibrarySourcesPinsAndRejectsChanges(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*domain.Library)
	}{
		{"source", func(l *domain.Library) { l.Versions[0].Files[0].Content = "changed" }},
		{"version", func(l *domain.Library) { l.CurrentVersion = "v2" }},
		{"quarantine", func(l *domain.Library) { l.LifecycleStatus = domain.LifecycleQuarantined }},
		{"privileged", func(l *domain.Library) { l.Manifest.Permissions.Network = domain.NetworkPolicyFull }},
		{"traversal", func(l *domain.Library) { l.Versions[0].Files[0].Path = "../outside.py" }},
		{"transitive", func(l *domain.Library) { l.Dependencies = []domain.DependencyRef{{ID: "another"}} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store := filesystem.NewStores(t.TempDir()).Libraries
			lib := libraryFixture()
			if err := store.Save(ctx, lib); err != nil {
				t.Fatal(err)
			}
			resolver := service.LibrarySources{Store: store}
			files, pins, err := resolver.Resolve(ctx, []domain.DependencyRef{{ID: lib.ID, Version: "v1"}})
			if err != nil || len(files) != 3 || pins[0].Digest == "" {
				t.Fatalf("files=%v pins=%v err=%v", files, pins, err)
			}
			tc.change(&lib)
			// Mutate metadata directly, so malicious paths are rejected by the resolver,
			// without asking the filesystem store to materialize unsafe source files.
			changed := &singleLibraryStore{LibraryStore: store, lib: lib}
			resolver.Store = changed
			if _, _, err = resolver.Resolve(ctx, pins); err == nil {
				t.Fatal("changed library accepted")
			}
			if tc.name != "source" {
				pins[0].Digest = ""
				if _, _, err = resolver.Resolve(ctx, pins); err == nil {
					t.Fatal("invalid unpinned library accepted")
				}
			}
		})
	}
}

type singleLibraryStore struct {
	ports.LibraryStore
	lib domain.Library
}

func (s *singleLibraryStore) Get(context.Context, string) (domain.Library, error) { return s.lib, nil }

func TestLibrarySharedByTwoTools(t *testing.T) {
	ctx := context.Background()
	stores := filesystem.NewStores(t.TempDir())
	lib := libraryFixture()
	if err := stores.Libraries.Save(ctx, lib); err != nil {
		t.Fatal(err)
	}
	listed, err := stores.Libraries.List(ctx)
	if err != nil || len(listed) != 1 {
		t.Fatalf("list=%v err=%v", listed, err)
	}
	resolver := service.LibrarySources{Store: stores.Libraries}
	var sandbox ports.Sandbox = &checkingSandbox{}
	if os.Getenv("EVOTOOL_DOCKER_TEST") == "1" {
		sandbox = dockersandbox.Sandbox{Image: "python:3.13-alpine"}
	}
	memory, err := service.NewCapabilityMemoryService(service.CapabilityMemoryServiceConfig{ToolStore: stores.Tools, LibraryStore: stores.Libraries, RegistryStore: stores.Registry, AuditLogger: stores.Audit})
	if err != nil {
		t.Fatal(err)
	}
	learning := service.LearningService{Memory: memory, Libraries: resolver, Validator: service.ValidationService{Sandbox: sandbox, Libraries: resolver}}
	executor := pythonexecutor.Executor{Sandbox: sandbox, Libraries: resolver}
	for _, name := range []string{"csv_shared_summary", "revenue_shared_summary"} {
		g := generatedFixture(t)
		g.Manifest.Name = name
		g.Manifest.LibraryRefs = []domain.DependencyRef{{ID: lib.ID, Version: "v1"}}
		g.SourceFiles[0].Content = strings.ReplaceAll(g.SourceFiles[0].Content, "import csv, io, json, sys", "import csv, io, json, sys\nfrom shared.numeric_core.stats import total")
		g.SourceFiles[0].Content = strings.ReplaceAll(g.SourceFiles[0].Content, "sum(values)", "total(values)")
		tool, report, err := learning.LearnSubmittedTool(ctx, g)
		if err != nil || !report.Valid {
			t.Fatalf("learn: %v %+v", err, report)
		}
		graph, err := stores.Registry.GetDependencyGraph(ctx, tool.ID)
		if err != nil || len(graph.Edges) != 1 {
			t.Fatalf("graph=%+v err=%v", graph, err)
		}
		var input map[string]any
		_ = json.Unmarshal([]byte(g.Manifest.Examples[0].Input), &input)
		result, err := executor.Execute(ctx, tool, input)
		if err != nil || !result.Success {
			t.Fatalf("execute=%+v err=%v", result, err)
		}
		// Source modification invalidates both tools without running any new code.
		changed := libraryFixture()
		changed.Versions[0].Files[0].Content = "raise Exception('must not execute')"
		executor.Libraries = service.LibrarySources{Store: &singleLibraryStore{LibraryStore: stores.Libraries, lib: changed}}
		if _, err := executor.Execute(ctx, tool, input); err == nil {
			t.Fatal("changed dependency executed")
		}
		executor.Libraries = resolver
	}
}
