# EvoTool

A capability memory layer that allows AI agents to discover, reuse, and evolve generated tools.

中文定位：一个让 Agent 学会积累、复用和演进能力的能力记忆层。

EvoTool helps AI agents turn one-off generated scripts into validated, searchable, reusable, and evolvable capabilities. Instead of regenerating the same helper code for similar tasks, an agent can build a capability memory and reuse proven tools and shared libraries across future work.

For the design reasoning, the progress as of the latest commit, and the tradeoffs still unresolved, read [the project write-up](docs/blog-capability-memory.md).

## Quick Start

External agents submit Python tool packages for sandbox validation and reuse. See [the learning workflow and one-command acceptance test](docs/learning.md). This path requires a running Docker Linux engine and a prepared Python image; it does not embed an LLM.

Learn a tool from a submitted package:

```bash
go run ./cmd/evotool learn --package examples/csv-analysis/generated.json --root .evotool
go run ./cmd/evotool search tools --query "csv numeric summary" --root .evotool
go run ./cmd/evotool execute --id csv_numeric_summary --input examples/csv-analysis/input.json --root .evotool
```

This creates a local structure like:

```text
.evotool/
  tools/csv_numeric_summary/versions/v1/main.py
  registry/metadata/tools/csv_numeric_summary.json
  registry/execution_history/csv_numeric_summary.jsonl
  registry/audit/events.jsonl
```

The search command runs a local retrieval pipeline: keyword recall, metadata filtering, capability ranking, and final selection.

Library retrieval is intentionally separate from Tool retrieval: a Tool is a top-level callable capability, while a Library is a reusable implementation dependency.

```bash
go run ./cmd/evotool search libraries --query "csv statistics" --runtime python --root .evotool
```

Inspect the Tool -> Library dependency graph:

```bash
go run ./cmd/evotool deps graph --id csv_numeric_summary --root .evotool
go run ./cmd/evotool deps check --id csv_numeric_summary --root .evotool
```

Measure reuse with the benchmark:

```bash
go run ./cmd/evotool benchmark run --suite examples/benchmark/csv-reporting-suite.json --root .evotool
```

Preview Tools that have been inactive for a long time:

```bash
go run ./cmd/evotool lifecycle scan --root .evotool
```

The default policy marks Tools unused for 90 days as `stale` candidates and Tools unused for 180 days as archive candidates. This command is read-only: it reports candidates but does not change or delete any files.

## Status

EvoTool is in the early design and MVP stage.

The execution policy service checks lifecycle, validation trust, version presence, and explicit host resource ceilings. `ExecutionService` coordinates dependency checks, policy checks, audit events, executor calls, and execution history. The external submission workflow now validates dependency-free Python tools and executes them through a Docker sandbox. The benchmark runner replays a task suite in both baseline and EvoTool mode and reports what reuse actually saved. Privileged submissions are rejected; model generation, executable shared-library dependencies, and a review approval interface are not implemented yet. See [the learning workflow](docs/learning.md), [the benchmark](docs/benchmark.md), and [the security model](docs/security.md).

## Core Idea

```text
Task
  -> Search learned tools
  -> Reuse if matched
  -> Generate if missing
  -> Detect reusable libraries
  -> Validate in sandbox
  -> Apply safety policy and review
  -> Save tool metadata and code
  -> Update registry and dependency graph
  -> Record execution results
  -> Evolve, refactor, or deprecate capabilities over time
```

## Capability Memory Architecture

EvoTool is not just a tool cache. It is designed as an Agent Capability Memory with three layers:

```text
Agent Capability Memory

  tools/        # User-callable capabilities
  libraries/    # Shared implementation building blocks
  registry/     # Metadata, embeddings, versions, dependencies, history, and policies
```

### Tools

Tools are capabilities an agent can directly select and execute:

```text
pdf_to_excel
csv_analyzer
sentiment_report_generator
webpage_to_markdown
```

Each tool should have a semantic name, a manifest, tests, documentation, execution history, and lifecycle state.

## Tool Lifecycle and Retention

能力库会随着 Agent 持续生成工具而增长。如果所有历史工具都永久参与检索，候选数量、检索噪声和维护成本都会不断增加。因此 EvoTool 需要提供工具保留策略，根据使用情况治理长期未使用的 Tool。

第一版设计不采用“长时间没用过就立即删除”，而是采用分阶段生命周期：

```text
active
  -> stale
  -> archived
  -> deleted
```

- `active`：正常参与检索和执行。
- `stale`：长时间未使用，降低检索优先级并等待复查。
- `archived`：默认不参与检索，但保留 Tool 文件、版本和元数据，可以恢复。
- `deleted`：满足安全条件后才允许物理删除。

生命周期判断应综合以下信息，而不是只看最后使用时间：

- `last_used_at`
- `usage_count`
- `last_successful_at`
- `success_rate`
- `last_validated_at`
- 是否被其他 Tool 或 Library 依赖
- 是否被标记为 `pinned`
- 是否仍在保留期限内
- 是否存在可替代的新版本

推荐治理流程：

```text
定期扫描
  -> 识别长期未使用 Tool
  -> 标记 stale
  -> 到期后归档
  -> 从默认检索集合移除
  -> 保留恢复入口
  -> 满足全部条件后才允许人工确认删除
```

默认检索只选择 `approved`、`active`、`stale` 三种状态，实现位于 `domain.DefaultRetrievableStatuses()`。`archived`、`quarantined`、`deleted`、`draft`、`validated` 不会出现在默认搜索结果中，调用方可以用 `RetrievalFilters.AllowedStatuses` 显式覆盖。若正常检索没有结果，可以由 Agent 或管理员显式查询 `archived` 工具，避免低频但重要的年度报表、应急修复和季节性工具被永久丢失。

删除前必须确认：

- 没有其他能力依赖该 Tool。
- 没有 `pinned` 或 `retain_until` 保护。
- 已经超过归档保留期限。
- 没有正在使用的版本。
- 删除操作经过显式确认并记录审计日志。

这项能力目前属于设计和后续实现阶段，当前 MVP 还不会自动归档或删除工具。详细方案见 [`docs/lifecycle.md`](docs/lifecycle.md)。

### Libraries

Libraries are shared lower-level capabilities used by multiple tools:

```text
pdf_parser
excel_writer
text_processor
data_cleaner
```

A tool can depend on libraries instead of copying the same helper code again and again.

### Registry

The registry is the capability index. It stores metadata, embeddings, categories, versions, execution history, safety policies, benchmark records, and the dependency graph between tools and libraries.

## Why Not Just Skills?

Skills are usually developer-defined capabilities:

```yaml
Skill:
  creator: Developer
  flow: Developer creates capability -> Agent uses capability
```

EvoTool focuses on learned tools generated during agent work:

```yaml
Learned Tool:
  creator: Agent
  flow: Agent creates capability -> EvoTool validates and stores it -> Future agents reuse it
```

EvoTool is not only a skill registry or tool registry. It manages the lifecycle of generated capabilities: naming, validation, safety review, retrieval, dependency reuse, execution history, benchmarking, versioning, and future evolution.

## Tool Naming Matters

Agents should not save tools as `tool1.py`, `test.py`, or `helper.py`.

A learned tool should have a semantic manifest:

```json
{
  "name": "pdf_table_extractor",
  "description": "Extract tables from PDF documents and export to Excel format",
  "category": "document_processing",
  "input": {
    "file": "pdf"
  },
  "output": {
    "file": "xlsx"
  }
}
```

Good names improve retrieval because future agents search by intent, description, metadata, embeddings, and execution history.

## Retrieval Strategy

EvoTool should not rely on tool names alone or send every tool to an LLM. Retrieval is staged:

```text
User Task
  -> Intent Extraction
  -> Semantic Retrieval, for example 20,000 tools to 50 candidates
  -> Metadata Filter, for example 50 candidates to 10 candidates
  -> Capability Rank, for example 10 candidates to 3 candidates
  -> LLM or rule selection, final 1 tool
```

Tool embeddings should be built from:

```text
name + description + tags + input/output schema + examples + capabilities
```

Ranking should combine:

- Semantic similarity
- Success rate
- Reliability
- Average latency
- Usage history
- Environment match
- Dependency health

This means EvoTool should know not only which tool is similar, but which tool is more reliable.

Execution outcomes now feed back into that ranking. After an executor actually runs, `ExecutionService` folds the result into the stored `UsageStats` and both metadata copies, so repeated successes raise a tool's score and repeated failures lower it. Calls blocked by policy or dependency checks are recorded in history and audit but do not count as usage experience.

## What EvoTool Aims To Provide

- Tool Memory for generated tools
- Shared Library Memory for reusable implementation building blocks
- Registry for metadata, embeddings, versions, dependency graph, policies, and execution history
- Benchmarking to measure how much generation reuse replaces
- Safety mechanisms, including sandboxing, permissions, review, execution limits, and dependency isolation
- MCP integration for agent ecosystems
- Eino adapter first, then OpenHands adapter
- Complete documentation and reproducible examples
- Real task cases such as PDF-to-Excel, webpage-to-Markdown, CSV analysis, and report generation

## Safety First

Generated tools are untrusted by default. EvoTool is designed around explicit safety boundaries:

- Sandbox execution
- Permission policies
- Human or policy-based review for high-risk tools
- Execution time, memory, output, file, and network limits
- Dependency isolation
- Audit logs for generation, validation, execution, failure, and version changes

A generated tool must not become an automatically reusable capability until it has passed validation and policy checks.

## Benchmark

EvoTool proves its value with reproducible benchmarks comparing:

```text
Baseline: Agent regenerates tools for recurring tasks.
EvoTool: Agent searches learned tools and libraries first, then generates only when needed.
```

```bash
go run ./cmd/evotool benchmark run --suite examples/benchmark/csv-reporting-suite.json --root .evotool
```

On the bundled suite of three recurring CSV reporting tasks, once `csv_numeric_summary` is stored with healthy dependencies:

```text
metric                         baseline    evotool      delta
tools generated                       3          0         -3
tools reused                          0          3         +3
duplicate generations                 2          0         -2
total tokens                       3600          0      -3600
```

With an empty capability memory the same suite reports every delta as `+0`. That zero is the point: the runner does not hardcode a saving.

EvoTool embeds no model, so the token counts are the generation cost each task declares in the suite. The runner measures how many generations reuse replaced, not model generation quality, and it does not execute the generated tools. The boundary is documented in [`docs/benchmark.md`](docs/benchmark.md).

Key metrics:

- Tool generation count
- Duplicate generation count
- Tool reuse count
- Library reuse count
- Average decision latency
- Average token usage
- Success and failure rates, once execution is wired into the harness
- Safety policy blocks
- Dependency-related failures

## Roadmap

Implemented:

- Core domain models, ports, and services
- Tool and Library manifests with naming validation
- Filesystem ToolStore, LibraryStore, and RegistryStore
- Staged retrieval: keyword recall, metadata filtering, capability ranking
- Execution outcomes fed back into ranking via `UsageRecorder`
- Registry, dependency graph, and dependency health checks
- Sandbox validation and execution history through a Docker sandbox
- Safety policy model with fail-closed privileged permissions
- Read-only lifecycle scan for stale and archive candidates
- Benchmark runner comparing baseline and EvoTool reuse

Not yet implemented:

- Model generation: `ToolGenerator` has no implementation, so tools are submitted by an external agent
- Executable shared-library dependencies: `LibraryRefs` are recorded and health-checked but rejected at validation
- A review approval interface for high-risk tools
- Lifecycle archive, restore, and purge commands
- MCP server adapter
- Eino adapter first, then OpenHands adapter
- Semantic/vector retrieval, replacing the current keyword retriever
- Code similarity detection and shared library extraction candidates
- Failure analysis and automatic repair

## License

MIT License.
