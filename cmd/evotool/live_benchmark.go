package main

import (
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
	"os"
)

func runLiveBenchmark(args []string) error {
	flags := flag.NewFlagSet("benchmark live", flag.ContinueOnError)
	root := flags.String("root", "", "dedicated benchmark memory root (required)")
	suiteFile := flags.String("suite", "", "live suite JSON with independent Expected outputs")
	out := flags.String("out", "", "optional JSON report file")
	image := flags.String("image", "python:3.13-alpine", "prepared sandbox image")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *root == "" {
		return fmt.Errorf("dedicated --root required; no positional arguments")
	}
	var suite domain.BenchmarkSuite
	if err := readWorkflowJSON(*suiteFile, &suite); err != nil {
		return err
	}
	generator, err := chatgenerator.New(chatgenerator.Config{Endpoint: os.Getenv("EVOTOOL_MODEL_ENDPOINT"), APIKey: os.Getenv("EVOTOOL_MODEL_API_KEY"), Model: os.Getenv("EVOTOOL_MODEL")})
	if err != nil {
		return err
	}
	stores := filesystem.NewStores(*root)
	memory, err := newCapabilityMemory(*root)
	if err != nil {
		return err
	}
	retrieval, err := newRetrievalService(*root, searchToolOptions{runtime: "python", limit: 3})
	if err != nil {
		return err
	}
	libraries := service.LibrarySources{Store: stores.Libraries}
	sandbox := dockersandbox.Sandbox{Image: *image}
	learning := service.LearningService{Memory: memory, Generator: generator, Libraries: libraries, Validator: service.ValidationService{Sandbox: sandbox, Libraries: libraries}, LibraryRetriever: localretriever.NewLibraryRetriever(*root)}
	dependencies, err := service.NewDependencyService(service.DependencyServiceConfig{RegistryStore: stores.Registry, LibraryStore: stores.Libraries})
	if err != nil {
		return err
	}
	policy, err := service.NewPolicyService(service.PolicyServiceConfig{ResourceLimits: domain.ResourceLimits{MaxRuntimeMs: 30000, MaxMemoryBytes: 268435456, MaxOutputBytes: 1048576}})
	if err != nil {
		return err
	}
	executor := pythonexecutor.Executor{Sandbox: sandbox, Libraries: libraries}
	execution, err := service.NewExecutionService(service.ExecutionServiceConfig{ToolStore: stores.Tools, Dependencies: dependencies, Policy: policy, Executor: executor, Recorder: stores.Executions, Audit: stores.Audit, Usage: stores.Usage})
	if err != nil {
		return err
	}
	benchmark := service.LiveBenchmarkService{Learning: &learning, Retrieval: retrieval, Execution: execution, BaselineExecutor: executor, Policy: policy}
	report, err := benchmark.Run(context.Background(), suite)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if *out != "" {
		if err = os.WriteFile(*out, data, 0600); err != nil {
			return err
		}
	}
	fmt.Println(string(data))
	return nil
}
