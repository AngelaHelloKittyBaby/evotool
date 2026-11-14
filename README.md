# EvoTool

**English** | [简体中文](README.zh-CN.md)

A capability memory layer that allows AI agents to discover, reuse, and evolve generated tools.

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

A capability library keeps growing as an agent generates more tools. If every historical tool stayed in retrieval forever, the candidate pool, retrieval noise, and maintenance cost would all keep climbing. EvoTool therefore needs a retention policy that governs Tools left unused for a long time.

The first version does not delete a tool the moment it stops being used. It applies a staged lifecycle:

```text
active
  -> stale
  -> archived
  -> deleted
```

- `active`: participates in retrieval and execution normally.
- `stale`: unused for a long time; demoted in retrieval priority and awaiting review.
- `archived`: excluded from retrieval by default, but Tool files, versions, and metadata are kept and can be restored.
- `deleted`: physical deletion only after the safety conditions are met.

Lifecycle decisions should combine the following signals instead of last-used time alone:

- `last_used_at`
- `usage_count`
- `last_successful_at`
- `success_rate`
- `last_validated_at`
- Whether other Tools or Libraries depend on it
- Whether it is marked `pinned`
- Whether it is still inside its retention window
- Whether a replacement version exists

Recommended governance flow:

```text
Periodic scan
  -> Identify Tools unused for a long time
  -> Mark stale
  -> Archive once the window elapses
  -> Remove from the default retrieval set
  -> Keep a restore entry point
  -> Allow manual confirmed deletion only when every condition holds
```

Default retrieval selects only the `approved`, `active`, and `stale` states; the implementation lives in `domain.DefaultRetrievableStatuses()`. `archived`, `quarantined`, `deleted`, `draft`, and `validated` never appear in default search results, and callers can override this explicitly through `RetrievalFilters.AllowedStatuses`. When normal retrieval returns nothing, an Agent or administrator can explicitly query `archived` tools, so low-frequency but important annual reports, emergency fixes, and seasonal tools are not lost for good.

Before deletion, all of the following must hold:

- No other capability depends on the Tool.
- No `pinned` or `retain_until` protection is in force.
- The archive retention window has elapsed.
- No version is still in use.
- The deletion was explicitly confirmed and written to the audit log.

This capability is still in the design and pending-implementation stage; the current MVP does not archive or delete tools automatically. See [`docs/lifecycle.md`](docs/lifecycle.md) for the full design.

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
