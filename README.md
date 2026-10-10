# EvoTool

A capability memory layer that allows AI agents to discover, reuse, and evolve generated tools.

中文定位：一个让 Agent 学会积累、复用和演进能力的能力记忆层。

EvoTool helps AI agents turn one-off generated scripts into validated, searchable, reusable, and evolvable capabilities. Instead of regenerating the same helper code for similar tasks, an agent can build a capability memory and reuse proven tools and shared libraries across future work.

## Quick Start

Save a demo generated tool into local capability memory:

```bash
go run ./cmd/evotool demo save-tool --root .evotool
```

This creates a local structure like:

```text
.evotool/
  tools/pdf_to_excel/
  registry/metadata/tools/pdf_to_excel.json
  registry/audit/events.jsonl
```

Search the local capability memory:

```bash
go run ./cmd/evotool search tools --query "pdf table excel" --root .evotool
```

The search command runs a local retrieval pipeline: keyword recall, metadata filtering, capability ranking, and final selection.

Save the demo shared library and search it before generating another tool:

```bash
go run ./cmd/evotool demo save-library --root .evotool
go run ./cmd/evotool search libraries --query "pdf parser table" --runtime python --root .evotool
```

Library retrieval is intentionally separate from Tool retrieval: a Tool is a top-level callable capability, while a Library is a reusable implementation dependency.

Record and inspect the Tool -> Library dependency graph:

```bash
go run ./cmd/evotool demo link-tool-library --root .evotool
go run ./cmd/evotool deps graph --id pdf_to_excel --root .evotool
go run ./cmd/evotool deps check --id pdf_to_excel --root .evotool
```

This records `pdf_to_excel -> pdf_parser` in `.evotool/registry/dependency_graph/pdf_to_excel.json`, updates the tool metadata with a `LibraryRefs` entry, and verifies that the referenced Library exists with a compatible version.

## Status

EvoTool is in the early design and MVP stage.

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

默认检索只选择 `active`、`approved` 等可用状态。若正常检索没有结果，可以由 Agent 或管理员显式查询 `archived` 工具，避免低频但重要的年度报表、应急修复和季节性工具被永久丢失。

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
## What EvoTool Aims To Provide

- Tool Memory for generated tools
- Shared Library Memory for reusable implementation building blocks
- Registry for metadata, embeddings, versions, dependency graph, policies, and execution history
- Benchmarking to prove reduced token usage and task latency
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

## Benchmark Goal

EvoTool should prove its value with reproducible benchmarks comparing:

```text
Baseline: Agent regenerates tools for recurring tasks.
EvoTool: Agent searches learned tools and libraries first, then generates only when needed.
```

Key metrics:

- Tool generation count
- Duplicate generation count
- Tool reuse count
- Library reuse count
- Average task latency
- Average token usage
- Success and failure rates
- Safety policy blocks
- Dependency-related failures

## Roadmap

- Core domain models
- Tool and Library manifests
- Core ports and services
- Filesystem ToolStore and LibraryStore
- Semantic retrieval with metadata filtering and capability ranking
- Registry and dependency graph
- Validation and execution history
- Safety policy model
- Benchmark runner
- MCP server adapter
- Eino adapter first, then OpenHands adapter
- Semantic/vector retrieval
- Dependency graph
- Tool lifecycle retention and archive policy
- Code similarity detection
- Shared library extraction candidates
- Tool versioning and lifecycle management
- Failure analysis and automatic repair

## License

MIT License.
