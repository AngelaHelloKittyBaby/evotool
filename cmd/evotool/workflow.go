package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	pythonexecutor "github.com/AngelaHelloKittyBaby/evotool/internal/adapter/executor/python"
	chatgenerator "github.com/AngelaHelloKittyBaby/evotool/internal/adapter/generator/chat"
	localretriever "github.com/AngelaHelloKittyBaby/evotool/internal/adapter/retriever/local"
	dockersandbox "github.com/AngelaHelloKittyBaby/evotool/internal/adapter/sandbox/docker"
	"github.com/AngelaHelloKittyBaby/evotool/internal/adapter/store/filesystem"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/service"
	"io"
	"os"
)

func runToolWorkflow(command string, args []string) error {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	root := flags.String("root", filesystem.DefaultRoot, "capability memory directory")
	image := flags.String("image", "python:3.13-alpine", "locally installed Python sandbox image")
	pkg := flags.String("package", "", "generated package JSON file")
	id := flags.String("id", "", "stored tool ID")
	input := flags.String("input", "", "input JSON file")
	taskFile := flags.String("task", "", "TaskSpec JSON file for model generation")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	stores := filesystem.NewStores(*root)
	memory, err := service.NewCapabilityMemoryService(service.CapabilityMemoryServiceConfig{ToolStore: stores.Tools, LibraryStore: stores.Libraries, RegistryStore: stores.Registry, ExecutionRecorder: stores.Executions, AuditLogger: stores.Audit})
	if err != nil {
		return err
	}
	sandbox := dockersandbox.Sandbox{Image: *image}
	libraries := service.LibrarySources{Store: stores.Libraries}
	ctx := context.Background()
	if command == "generate" {
		var task domain.TaskSpec
		if err = readWorkflowJSON(*taskFile, &task); err != nil {
			return err
		}
		generator, err := chatgenerator.New(chatgenerator.Config{Endpoint: os.Getenv("EVOTOOL_MODEL_ENDPOINT"), APIKey: os.Getenv("EVOTOOL_MODEL_API_KEY"), Model: os.Getenv("EVOTOOL_MODEL")})
		if err != nil {
			return err
		}
		learning := service.LearningService{Memory: memory, Validator: service.ValidationService{Sandbox: sandbox, Libraries: libraries}, Libraries: libraries, Generator: generator, LibraryRetriever: localretriever.NewLibraryRetriever(*root)}
		tool, generated, report, err := learning.GenerateAndLearn(ctx, task)
		if encodeErr := json.NewEncoder(os.Stdout).Encode(map[string]any{"tool_id": tool.ID, "generation": generated.Generation, "validation": report}); encodeErr != nil {
			return encodeErr
		}
		return err
	}
	if command == "learn" {
		var generated domain.GeneratedTool
		if err = readWorkflowJSON(*pkg, &generated); err != nil {
			return err
		}
		learning := service.LearningService{Validator: service.ValidationService{Sandbox: sandbox, Libraries: libraries}, Memory: memory, Libraries: libraries}
		tool, report, err := learning.LearnSubmittedTool(ctx, generated)
		if err != nil {
			_ = json.NewEncoder(os.Stdout).Encode(report)
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"tool_id": tool.ID, "version": tool.CurrentVersion, "validation": report})
	}
	if *id == "" {
		return fmt.Errorf("tool id required")
	}
	var payload map[string]any
	if err = readWorkflowJSON(*input, &payload); err != nil {
		return err
	}
	if payload == nil {
		return fmt.Errorf("input must be a JSON object")
	}
	dependencies, err := service.NewDependencyService(service.DependencyServiceConfig{RegistryStore: stores.Registry, LibraryStore: stores.Libraries})
	if err != nil {
		return err
	}
	policy, err := service.NewPolicyService(service.PolicyServiceConfig{ResourceLimits: domain.ResourceLimits{MaxRuntimeMs: 30000, MaxMemoryBytes: 268435456, MaxOutputBytes: 1048576}})
	if err != nil {
		return err
	}
	execution, err := service.NewExecutionService(service.ExecutionServiceConfig{ToolStore: stores.Tools, Dependencies: dependencies, Policy: policy, Executor: pythonexecutor.Executor{Sandbox: sandbox, Libraries: libraries}, Recorder: stores.Executions, Audit: stores.Audit, Usage: stores.Usage})
	if err != nil {
		return err
	}
	result, err := execution.ExecuteTool(ctx, *id, domain.TaskSpec{}, payload)
	if encodeErr := json.NewEncoder(os.Stdout).Encode(result); encodeErr != nil {
		return encodeErr
	}
	return err
}

func readWorkflowJSON(file string, target any) error {
	if file == "" {
		return fmt.Errorf("JSON file path required")
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 1048577))
	if err != nil {
		return err
	}
	if len(data) > 1048576 {
		return fmt.Errorf("JSON file exceeds 1 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(target); err != nil {
		return err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("unexpected trailing JSON")
	}
	return nil
}
