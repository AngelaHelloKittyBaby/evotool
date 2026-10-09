package service

import (
	"context"
	"fmt"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

// CapabilityMemoryServiceConfig contains dependencies for the capability memory facade.
type CapabilityMemoryServiceConfig struct {
	ToolStore         ports.ToolStore
	LibraryStore      ports.LibraryStore
	RegistryStore     ports.RegistryStore
	ExecutionRecorder ports.ExecutionRecorder
	AuditLogger       ports.AuditLogger
	RetrievalService  *RetrievalService
	LibraryRetriever  ports.LibraryRetriever
}

// CapabilityMemoryService provides the main application operations for EvoTool memory.
type CapabilityMemoryService struct {
	tools            ports.ToolStore
	libraries        ports.LibraryStore
	registry         ports.RegistryStore
	recorder         ports.ExecutionRecorder
	audit            ports.AuditLogger
	retrieval        *RetrievalService
	libraryRetriever ports.LibraryRetriever
}

// NewCapabilityMemoryService creates the capability memory facade.
func NewCapabilityMemoryService(config CapabilityMemoryServiceConfig) (*CapabilityMemoryService, error) {
	if config.ToolStore == nil {
		return nil, fmt.Errorf("%w: tool store", ErrMissingDependency)
	}
	if config.LibraryStore == nil {
		return nil, fmt.Errorf("%w: library store", ErrMissingDependency)
	}
	if config.RegistryStore == nil {
		return nil, fmt.Errorf("%w: registry store", ErrMissingDependency)
	}

	return &CapabilityMemoryService{
		tools:            config.ToolStore,
		libraries:        config.LibraryStore,
		registry:         config.RegistryStore,
		recorder:         config.ExecutionRecorder,
		audit:            config.AuditLogger,
		retrieval:        config.RetrievalService,
		libraryRetriever: config.LibraryRetriever,
	}, nil
}

// SearchTools searches reusable tools for a task.
func (s *CapabilityMemoryService) SearchTools(ctx context.Context, task domain.TaskSpec) (domain.RetrievalResult, error) {
	if s.retrieval == nil {
		return domain.RetrievalResult{}, fmt.Errorf("%w: retrieval service", ErrMissingDependency)
	}
	return s.retrieval.Search(ctx, task)
}

// SearchLibraries searches reusable libraries for a generated tool dependency query.
func (s *CapabilityMemoryService) SearchLibraries(ctx context.Context, query domain.RetrievalQuery) ([]domain.LibraryCandidate, error) {
	if s.libraryRetriever == nil {
		return nil, fmt.Errorf("%w: library retriever", ErrMissingDependency)
	}
	candidates, err := s.libraryRetriever.SearchLibraries(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search libraries: %w", err)
	}
	return candidates, nil
}

// SaveTool saves a tool and synchronizes its registry metadata.
func (s *CapabilityMemoryService) SaveTool(ctx context.Context, tool domain.Tool) error {
	if err := s.tools.Save(ctx, tool); err != nil {
		return fmt.Errorf("save tool: %w", err)
	}
	if err := s.registry.UpsertToolMetadata(ctx, tool); err != nil {
		return fmt.Errorf("upsert tool metadata: %w", err)
	}
	return s.logAudit(ctx, "tool.saved", domain.AuditSubjectTool, tool.ID, tool.Name)
}

// GetTool returns a stored tool by ID.
func (s *CapabilityMemoryService) GetTool(ctx context.Context, id string) (domain.Tool, error) {
	tool, err := s.tools.Get(ctx, id)
	if err != nil {
		return domain.Tool{}, fmt.Errorf("get tool: %w", err)
	}
	return tool, nil
}

// SaveLibrary saves a library and synchronizes its registry metadata.
func (s *CapabilityMemoryService) SaveLibrary(ctx context.Context, library domain.Library) error {
	if err := s.libraries.Save(ctx, library); err != nil {
		return fmt.Errorf("save library: %w", err)
	}
	if err := s.registry.UpsertLibraryMetadata(ctx, library); err != nil {
		return fmt.Errorf("upsert library metadata: %w", err)
	}
	return s.logAudit(ctx, "library.saved", domain.AuditSubjectLibrary, library.ID, library.Name)
}

// GetLibrary returns a stored library by ID.
func (s *CapabilityMemoryService) GetLibrary(ctx context.Context, id string) (domain.Library, error) {
	library, err := s.libraries.Get(ctx, id)
	if err != nil {
		return domain.Library{}, fmt.Errorf("get library: %w", err)
	}
	return library, nil
}

// RecordExecution records a tool execution result.
func (s *CapabilityMemoryService) RecordExecution(ctx context.Context, result domain.ExecutionResult) error {
	if s.recorder == nil {
		return fmt.Errorf("%w: execution recorder", ErrMissingDependency)
	}
	if err := s.recorder.RecordExecution(ctx, result); err != nil {
		return fmt.Errorf("record execution: %w", err)
	}
	return s.logAudit(ctx, "execution.recorded", domain.AuditSubjectExecution, result.ToolID, result.Version)
}

func (s *CapabilityMemoryService) logAudit(ctx context.Context, action string, subjectKind domain.AuditSubjectKind, subjectID, message string) error {
	if s.audit == nil {
		return nil
	}
	if err := s.audit.Log(ctx, domain.AuditEvent{
		Action:      action,
		SubjectKind: subjectKind,
		SubjectID:   subjectID,
		Message:     message,
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("log audit: %w", err)
	}
	return nil
}
