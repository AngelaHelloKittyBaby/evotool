# Benchmark：度量复用省下了什么

本文下方介绍 `benchmark run` 的静态重放模式。另有 `benchmark live`，已接入 ToolGenerator、沙箱验证、实际执行及独立输出核验，使用 provider 返回的 token；详见 [模型生成与实测 Benchmark](generation.md)。不要把两种模式的结果混用。

EvoTool 的核心主张是“已有能力被复用，就不再重复生成”。论文不需要，但需要可复现的数字。这份文档说明当前 benchmark 实测什么、不实测什么，以及怎么跑。

对应实现：`internal/service/benchmark_service.go`、`cmd/evotool/benchmark.go`、`internal/domain/benchmark.go`。
示例 suite：`examples/benchmark/csv-reporting-suite.json`。

## 两种模式

同一组任务跑两遍，唯一的差别是有没有先查能力记忆：

```text
baseline：每个任务都生成一个工具（模拟没有 Tool Memory 的 Agent）
evotool ：先检索能力记忆；命中的工具通过依赖健康检查就复用，不命中才生成
```

两种模式共用同一份已保存的能力记忆，且本轮 run 中新“生成”的能力不会回头被后续任务复用。否则度量的是“同一轮里边生成边学习”，那不是这里要证明的东西。

## 度量的是什么

静态 `benchmark run` 不调用大模型。任务的 `Package`（本该生成的产物）和 `TokenUsage`（本该花掉的生成成本）都由 suite 提供。

因此这个 benchmark 度量的是**复用替代了多少次生成**，不是模型生成质量，也不是真实 token 账单。

```text
Task
  -> baseline: 计入一次生成，累加该任务的 TokenUsage
  -> evotool : 检索 -> 命中且依赖健康 -> 计入一次复用，token 计 0
  -> evotool : 检索 -> 未命中或不健康 -> 计入一次生成，累加 TokenUsage
```

这一点在 `BenchmarkResult` 上有注释，在 CLI 输出末尾也会打印一次。任何引用这些数字的地方都必须保留这个边界，否则就是拿估算冒充实测。

## 指标

| 指标 | 含义 |
| --- | --- |
| `ToolGeneratedCount` | 本模式一共生成多少次工具 |
| `ToolReusedCount` | 本模式复用已有工具多少次（baseline 恒为 0） |
| `DuplicateGenerationCount` | 同一个包名在一轮里被生成第 2 次及以后的数量 |
| `LibraryReusedCount` | 任务包声明的 Library 引用中，能在能力记忆里解析到的数量 |
| `SafetyPolicyBlockCount` | 会被策略拒绝或需要 review 的任务数 |
| `DependencyFailureCount` | 检索命中但依赖不健康、因此不能复用的次数 |
| `TotalTokenUsage` / `AverageTokenUsage` | 只统计“发生生成”的那些任务 |
| `AverageLatency` | 该模式处理每个任务的平均耗时 |

`AverageLatency` 记的是本地检索、依赖检查和策略评估的耗时。它**不**包含工具执行时间，也不包含模型生成时间（这里没有模型）。所以它适合做回归对比，不适合对外宣称“任务快了多少”。

`SuccessRate` 和 `FailureRate` 保持为 0。这个 benchmark 重放的是复用决策，它没有真正执行生成的工具；填 1.0 就是一个没人测量过的断言。

## 报告

`BenchmarkReport` 给出三个派生量，允许为负：EvoTool 反而多生成时，`GenerationSaved()` 会是负数，调用方不能假设它一定为正。

```go
report.GenerationSaved()          // baseline 生成数 - evotool 生成数
report.DuplicateGenerationSaved() // baseline 重复生成数 - evotool 重复生成数
report.TokenSaved()               // 差值 token
```

## 运行

```bash
go run ./cmd/evotool benchmark run --suite examples/benchmark/csv-reporting-suite.json --root .evotool
go run ./cmd/evotool benchmark run --suite examples/benchmark/csv-reporting-suite.json --root .evotool --out result.json
```

suite 的结构：

```json
{
  "Name": "csv_reporting_reuse",
  "Tasks": [
    {
      "Name": "monthly_revenue_summary",
      "Spec": { "Intent": "...", "InputSummary": "...", "ExpectedOutput": "...", "Domain": "data_processing" },
      "Package": { "Manifest": {}, "SourceFiles": [], "Tests": [] },
      "TokenUsage": { "PromptTokens": 900, "CompletionTokens": 300, "TotalTokens": 1200 },
      "Input": {}
    }
  ]
}
```

校验规则：suite 必须有名字、至少一个任务；每个任务必须有名字、Package 必须有 `Manifest.Name`。空记忆下跑出来的必须是零收益，这条有测试守着（`TestBenchmarkWithEmptyMemoryShowsNoSavings`）。

## 实测结果

`examples/benchmark/csv-reporting-suite.json` 是三个报表任务（月度、季度、小时），输入列不同、需要的能力相同，每个任务假定生成成本 1200 token。

**空能力记忆**（没有可复用工具）：

```text
metric                         baseline    evotool      delta
tools generated                       3          3         +0
tools reused                          0          0         +0
duplicate generations                 2          2         +0
total tokens                       3600       3600         +0
```

**已存有 `csv_numeric_summary` 且依赖健康**：

```text
metric                         baseline    evotool      delta
tools generated                       3          0         -3
tools reused                          0          3         +3
duplicate generations                 2          0         -2
total tokens                       3600          0      -3600
```

三行数字要一起看：

- 空记忆时收益必须是 0。这一条比“省了 3600 token”更重要，它证明数字不是写死的。
- 已有能力时，3 次生成全部被复用替代，重复生成从 2 降到 0。
- 3600 token 是“本该花掉”的生成成本，按 suite 声明计，不是实测账单。

依赖不健康时不会复用。若工具引用的 Library 缺失，同样 suite 会得到 `tools reused = 0`、`dependency failures = 3`、`GenerationSaved() = 0`——检索命中不等于能跑，benchmark 不会替一个坏依赖记功（`TestBenchmarkDoesNotReuseUnhealthyDependencies`）。

## 这个 benchmark 不覆盖什么

- 没有模型调用，所以没有真实的 prompt/completion 账单差异。
- 没有执行生成的工具，所以没有成功率和失败率。
- 没有包含工具执行耗时，`AverageLatency` 只是本地决策耗时。
- 没有覆盖跨轮学习。任务序列内的学习效应不在度量范围内。

这些指标已由独立的 `benchmark live` 路径实现；静态 `benchmark run` 保持现有重放语义，不会冒充现场模型调用。真实费用结论仍需使用者在自己的模型配置下运行并核对账单。

## 与其他指标的关系

静态 `benchmark run` 只读能力记忆：不写执行历史、不更新 `UsageStats`、不影响后续检索排序。`benchmark live` 则会学习工具、记录执行并更新统计，需使用专门的实验目录。正常执行的统计流程见 [learning.md](learning.md) 的“使用统计与检索排序”。

## 历史验证

2026-11-18，已保存 CSV 工具时，静态重放的生成次数由 3 降为 0、复用次数由 0 升为 3。3600 token 来自 suite 估算，不是模型 API 实测。当前生成与执行闭环的验证证据见 [verification.md](verification.md)。
