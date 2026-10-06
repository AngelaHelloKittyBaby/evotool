# EvoTool

A capability memory layer that allows AI agents to discover, reuse, and evolve generated tools.

中文定位：一个让 Agent 学会积累、复用和演进能力的能力记忆层。

EvoTool helps AI agents turn one-off generated scripts into validated, searchable, reusable, and evolvable capabilities. Instead of regenerating the same helper code for similar tasks, an agent can build a capability memory and reuse proven tools and shared libraries across future work.

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
- Dependency graph`r`n- Code similarity detection`r`n- Shared library extraction candidates
- Tool versioning and lifecycle management
- Failure analysis and automatic repair

## License

MIT License.

