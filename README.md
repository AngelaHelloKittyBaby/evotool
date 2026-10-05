# EvoTool

A self-evolving tool memory system for AI agents.

EvoTool helps agents save, retrieve, validate, and reuse tools they generate during task execution. Instead of regenerating the same helper code again and again, an agent can gradually build a learned tool library and reuse proven capabilities across future tasks.

## Status

EvoTool is in the early design and MVP stage.

## Core Idea

Agents often create temporary scripts or helper tools to complete tasks. EvoTool turns those one-off tools into reusable capabilities:

```text
Task
  -> Search learned tools
  -> Reuse if matched
  -> Generate if missing
  -> Validate in sandbox
  -> Save tool metadata and code
  -> Record execution results
  -> Improve future reuse
```

## Planned Features

- Tool manifest and metadata storage
- Semantic retrieval for learned tools
- Tool input/output schema validation
- Sandbox-based tool execution
- Execution history and success/failure tracking
- Tool versioning
- Agent framework adapters, such as Eino or MCP-based runtimes

## Project Goals

EvoTool aims to provide a reusable infrastructure layer for AI agents:

- Reduce repeated tool generation
- Lower token and execution cost for recurring tasks
- Make generated tools inspectable and testable
- Help agents accumulate reusable capabilities over time

## License

MIT License.
