# EvoTool

A capability memory layer that allows AI agents to discover, reuse, and evolve generated tools.

中文定位：一个让 Agent 学会积累、复用和演进能力的能力记忆层。

EvoTool helps AI agents turn one-off generated scripts into validated, searchable, reusable, and evolvable tools. Instead of regenerating the same helper code for similar tasks, an agent can build a learned tool memory and reuse proven capabilities across future work.

## Status

EvoTool is in the early design and MVP stage.

## Core Idea

```text
Task
  -> Search learned tools
  -> Reuse if matched
  -> Generate if missing
  -> Validate in sandbox
  -> Apply safety policy and review
  -> Save tool metadata and code
  -> Record execution results
  -> Evolve or deprecate tools over time
```

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

EvoTool is not only a tool registry. It manages the lifecycle of generated tools: discovery, validation, safety review, retrieval, execution history, benchmarking, versioning, and future evolution.

## What EvoTool Aims To Provide

- Tool Memory for generated tools
- Benchmarking to prove reduced token usage and task latency
- Safety mechanisms, including sandboxing, permissions, review, execution limits, and dependency isolation
- MCP integration for agent ecosystems
- Eino/OpenHands adapters
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
EvoTool: Agent searches learned tools first and generates only when needed.
```

Key metrics:

- Tool generation count
- Duplicate generation count
- Tool reuse count
- Average task latency
- Average token usage
- Success and failure rates
- Safety policy blocks

## Roadmap

- Core domain models
- Core ports and services
- Filesystem ToolStore
- Keyword-based retrieval
- Validation and execution history
- Safety policy model
- Benchmark runner
- MCP server adapter
- Eino/OpenHands adapters
- Semantic/vector retrieval
- Tool versioning and lifecycle management
- Failure analysis and automatic repair

## License

MIT License.
