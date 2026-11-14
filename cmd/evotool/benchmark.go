package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// runBenchmarkSuite loads a suite, runs both modes, and prints the comparison.
func runBenchmarkSuite(root, suitePath, outPath string) error {
	var suite domain.BenchmarkSuite
	if err := readWorkflowJSON(suitePath, &suite); err != nil {
		return fmt.Errorf("read benchmark suite: %w", err)
	}

	benchmark, err := newBenchmarkService(root)
	if err != nil {
		return err
	}
	report, err := benchmark.Run(context.Background(), suite)
	if err != nil {
		return err
	}

	printBenchmarkReport(report)
	if outPath == "" {
		return nil
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode benchmark report: %w", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write benchmark report: %w", err)
	}
	fmt.Printf("wrote report to %s\n", outPath)
	return nil
}

func printBenchmarkReport(report domain.BenchmarkReport) {
	fmt.Printf("benchmark: %s\n", report.SuiteName)
	fmt.Printf("%-28s %10s %10s %10s\n", "metric", "baseline", "evotool", "delta")
	printBenchmarkRow("tools generated", report.Baseline.ToolGeneratedCount, report.EvoTool.ToolGeneratedCount)
	printBenchmarkRow("tools reused", report.Baseline.ToolReusedCount, report.EvoTool.ToolReusedCount)
	printBenchmarkRow("duplicate generations", report.Baseline.DuplicateGenerationCount, report.EvoTool.DuplicateGenerationCount)
	printBenchmarkRow("libraries reused", report.Baseline.LibraryReusedCount, report.EvoTool.LibraryReusedCount)
	printBenchmarkRow("policy blocks", report.Baseline.SafetyPolicyBlockCount, report.EvoTool.SafetyPolicyBlockCount)
	printBenchmarkRow("dependency failures", report.Baseline.DependencyFailureCount, report.EvoTool.DependencyFailureCount)
	printBenchmarkRow("prompt tokens", report.Baseline.TotalTokenUsage.PromptTokens, report.EvoTool.TotalTokenUsage.PromptTokens)
	printBenchmarkRow("completion tokens", report.Baseline.TotalTokenUsage.CompletionTokens, report.EvoTool.TotalTokenUsage.CompletionTokens)
	printBenchmarkRow("total tokens", report.Baseline.TotalTokenUsage.TotalTokens, report.EvoTool.TotalTokenUsage.TotalTokens)
}

func printBenchmarkRow(label string, baseline, evotool int) {
	fmt.Printf("%-28s %10d %10d %+10d\n", label, baseline, evotool, evotool-baseline)
}

func printBenchmarkUsage() {
	fmt.Println(`EvoTool benchmark commands

Usage:
  evotool benchmark run --suite benchmark.json [--root .evotool] [--out result.json]

The suite replays each task twice: baseline always generates a tool, evotool
searches capability memory first. Token counts come from the suite, not from a
model call; this benchmark measures reuse, not generation quality.`)
}
