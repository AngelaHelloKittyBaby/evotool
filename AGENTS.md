# EvoTool Agent 开发指南

本文档用于约束后续 Coding Agent 在 EvoTool 仓库中的设计和编码行为。任何 Agent 在编写或修改代码前，都应该先阅读并遵守本文档。

EvoTool 是一个面向 AI Agent 的自进化工具记忆系统。它的核心职责是：让 Agent 能够在任务执行过程中保存、检索、验证、执行并复用自己生成过的工具，而不是每次遇到相似任务都重新生成代码。

## 架构目标

EvoTool 必须设计成与具体 Agent 框架无关的基础设施库。

核心模块不应该绑定任何特定 Agent 框架、LLM Provider、数据库、向量数据库、沙箱实现或 MCP 实现。这些外部技术都应该通过 Adapter 接入，并且可以被替换。

本项目需要使用 OOAD 思想，但不要做成过度设计的继承体系。重点是职责清晰、边界稳定、接口抽象合理。

设计原则：

- 建模清晰的领域对象。
- 把行为放到高内聚的服务中。
- 依赖接口，而不是依赖具体实现。
- 优先组合，而不是继承。
- 包要小而专注。
- 不为不存在的变化提前抽象。
- 只有当抽象保护了真实边界时才引入抽象。

目标质量：

- 可扩展
- 可复用
- 可维护
- 高内聚
- 低耦合
- 默认可测试

## 推荐项目结构

默认按以下方向组织项目：

```text
evotool/
  cmd/
    evotool/                 # CLI 入口

  internal/
    domain/                  # 核心领域对象和领域规则
      tool.go
      manifest.go
      task.go
      execution.go
      validation.go
      version.go

    ports/                   # EvoTool 核心拥有的接口定义
      store.go
      retriever.go
      generator.go
      validator.go
      executor.go
      sandbox.go
      embedder.go
      mcp.go

    service/                 # 应用服务和用例编排
      memory_service.go
      learning_service.go
      execution_service.go
      version_service.go

    adapter/                 # 可替换的基础设施实现
      store/
        filesystem/
        sqlite/
      retriever/
        keyword/
        vector/
      generator/
        openai/
      executor/
        python/
        shell/
      sandbox/
        local/
        docker/
      mcp/
        server/
        client/

    api/                     # 对外门面，供应用、示例、Adapter 使用
      evotool.go

  examples/
    simple-agent/
    mcp-server/
    eino-adapter/

  docs/
    architecture.md
    mcp.md
    roadmap.md
```

不要在一开始就创建所有目录。随着功能真实出现，再逐步扩展结构。

## 依赖规则

依赖方向必须向内流动：

```text
cmd/examples/adapters -> api -> service -> ports/domain
adapter implementations -> ports/domain
```

具体规则：

- `domain` 不能导入 `service`、`adapter`、`api`、`cmd` 或任何外部 Agent 框架包。
- `ports` 可以依赖 `domain`，但不能依赖具体 Adapter。
- `service` 只能依赖 `domain` 和 `ports`。
- `adapter` 负责实现 `ports` 中定义的接口，可以依赖外部库。
- `api` 是对外门面，负责组织公开用法，但不能隐藏核心错误，也不能写死基础设施实现。
- `cmd` 和 `examples` 是装配层，负责把具体实现组合起来。

核心代码禁止直接依赖：

- Eino
- OpenHands
- LangChain
- OpenAI SDK
- MCP SDK
- SQLite Driver
- 向量数据库 Client
- Docker Client
- Shell/Python 执行库

这些依赖只能出现在 Adapter 层。

## 核心领域概念

后续代码中应统一使用以下概念。

### Tool

表示一个 EvoTool 学到或注册的可复用能力。

建议字段：

- ID
- Name
- Description
- Manifest
- CurrentVersion
- Versions
- CreatedAt
- UpdatedAt

### ToolManifest

表示工具的机器可读契约。

建议字段：

- Name
- Description
- Runtime
- EntryPoint
- Inputs
- Outputs
- Capabilities
- Dependencies
- Tags

### TaskSpec

表示用户任务或 Agent 任务的标准化描述。

建议字段：

- Intent
- InputSummary
- Constraints
- ExpectedOutput
- Metadata

### ToolCandidate

表示一次检索得到的候选工具及其匹配信息。

建议字段：

- Tool
- Score
- MatchReason
- Source

### GeneratedTool

表示刚生成、还没有进入可信工具记忆的新工具。

建议字段：

- Manifest
- SourceFiles
- Tests
- Readme
- Dependencies

### ValidationResult

表示工具验证结果。

建议字段：

- Valid
- Errors
- Warnings
- TestOutput
- SandboxReport

### ExecutionResult

表示一次工具执行记录。

建议字段：

- ToolID
- Version
- Success
- Output
- Error
- StartedAt
- FinishedAt
- Duration
- Metadata

## 核心接口

核心接口统一放在 `internal/ports` 中。接口要小而专注，避免大而全。

接口设计方向示例：

```go
type ToolStore interface {
    Save(ctx context.Context, tool domain.Tool) error
    Get(ctx context.Context, id string) (domain.Tool, error)
    Update(ctx context.Context, tool domain.Tool) error
}

type ToolRetriever interface {
    Search(ctx context.Context, intent string, limit int) ([]domain.ToolCandidate, error)
}

type ToolGenerator interface {
    Generate(ctx context.Context, task domain.TaskSpec) (domain.GeneratedTool, error)
}

type ToolValidator interface {
    Validate(ctx context.Context, tool domain.GeneratedTool) (domain.ValidationResult, error)
}

type ToolExecutor interface {
    Execute(ctx context.Context, tool domain.Tool, input map[string]any) (domain.ExecutionResult, error)
}

type ExecutionRecorder interface {
    RecordExecution(ctx context.Context, result domain.ExecutionResult) error
}
```

不要创建巨大的接口。如果某个实现并不自然需要接口中的所有方法，就拆分接口。

## MVP 主流程

第一版稳定 MVP 应该先实现以下闭环：

```text
TaskSpec
  -> 搜索 learned tools
  -> 如果找到匹配工具：
       执行工具
       记录执行结果
  -> 如果没有找到匹配工具：
       生成工具
       在沙箱中验证工具
       保存工具和元数据
       执行工具
       记录执行结果
```

不要一开始就做自动修复、分布式执行、多 Agent 协作。那些属于后续阶段。

## MCP 要求

MCP 很重要，但 MCP 必须是集成层，而不是核心架构本身。

EvoTool 应该从两个方向支持 MCP：

1. 把 EvoTool 暴露成 MCP Server，让外部 Agent 调用。
2. 把外部 MCP 工具作为可能的工具来源或执行后端。

推荐 MCP Adapter 结构：

```text
internal/adapter/mcp/
  server/       # 将 EvoTool 能力暴露成 MCP Tools
  client/       # 在需要时调用外部 MCP Server
```

第一版 MCP Server 可以提供这些工具：

- `evotool.search_tools`
- `evotool.get_tool`
- `evotool.save_tool`
- `evotool.execute_tool`
- `evotool.record_execution`

MCP 实现规则：

- 不要把 MCP 请求/响应结构体放进 `domain`。
- 在 Adapter 边界把 MCP Payload 转换成 EvoTool 领域对象。
- MCP Tool 名称必须稳定，并在文档中说明。
- MCP Adapter 应该依赖 `api` 或 `service` 接口，不能重新写业务逻辑。
- 即使没有安装 MCP 相关依赖，EvoTool 核心也应该可以正常使用。

MCP 的作用是让 EvoTool 更容易接入 Agent 生态，而不是让 EvoTool 只能通过 MCP 使用。

## Agent 框架接入要求

EvoTool 后续可以为 Eino、OpenHands、LangChain、自定义 ReAct Agent 等框架提供 Adapter。

规则：

- 不要把框架特定类型放进核心包。
- 框架集成代码放在 `internal/adapter/<framework>` 中。
- 如果未来需要公开 Adapter，再考虑移动到 `pkg/adapter/<framework>`。
- 框架 Adapter 负责把框架概念转换成 EvoTool 概念。
- EvoTool 核心必须能被普通 Go 程序直接使用，不依赖任何 Agent 框架。

## Go 代码规范

使用惯用 Go 风格。

- 所有 Go 文件必须经过 `gofmt`。
- 包名要短、清晰、有语义。
- 导出的类型、函数、接口必须有注释。
- 涉及 I/O 或长耗时操作的方法，第一个参数使用 `context.Context`。
- 返回错误，不要随意 `panic`。
- 用 `fmt.Errorf("...: %w", err)` 包装错误上下文。
- 避免全局可变状态。
- 避免隐藏的后台 goroutine，除非生命周期管理非常明确。
- 函数要足够小，方便单独测试。
- 核心逻辑优先使用表驱动测试。

## 测试要求

每个核心 Service 都应该有单元测试。

重点测试：

- 工具搜索决策逻辑
- 保存和读取行为
- 验证成功/失败行为
- 执行结果记录
- 错误传播
- Adapter 边界转换

Adapter 测试可以使用 fake 或临时目录。普通单元测试不能强依赖 Docker、网络、付费 API 或外部服务。

如果集成测试需要外部服务，必须明确标记，并保持可选。

## 安全要求

生成出来的工具在验证之前都应该被视为不可信。

规则：

- 生成代码不能绕过 Validator 和 Executor 直接执行。
- 默认情况下，生成工具不能访问任意宿主机路径。
- 除非配置显式允许，否则不能把密钥传给生成工具。
- 沙箱策略必须显式表达。
- 必须记录足够的执行元数据，方便排查失败。
- 生成工具声明的依赖必须视为不可信输入。

第一版 MVP 可以先使用本地 Executor，但架构上必须允许之后替换成 Docker 或其他沙箱实现。

## 持久化要求

第一版可以从简单文件系统存储开始，但存储必须可替换。

可接受的第一版本地存储结构：

```text
.evotool/
  tools/
    <tool-id>/
      manifest.json
      versions/
        v1/
          tool.py
          README.md
          tests/
      metadata.json
      history.jsonl
```

规则：

- 不要把文件系统布局硬编码进 Service。
- 文件系统细节放在 Store Adapter 中。
- 必须保存足够的元数据，以支持未来语义检索和版本管理。
- 执行历史应使用便于追加写入的格式。

## 文档要求

新增重要概念时，需要同步更新文档。

重要文档：

- `README.md`：项目定位和快速开始。
- `docs/architecture.md`：架构和设计决策。
- `docs/mcp.md`：MCP Tool 名称、Payload 和示例。
- `docs/roadmap.md`：阶段性实现计划。

文档要务实，优先使用具体例子，少写空泛口号。

## 禁止事项

不要做以下事情：

- 把核心逻辑绑定到某一个 Agent 框架。
- 把所有逻辑都写进 CLI 命令里。
- 把所有逻辑都写进 MCP Server Handler 里。
- 让 Adapter 承担业务规则。
- 创建一个无所不包的 `Manager` 对象。
- 在简单 Retriever 稳定之前引入向量数据库。
- 在生成、验证、保存、搜索、执行稳定之前做自动修复。
- 从 Generator 直接执行生成代码。
- 吞掉失败信息而不记录。

## 实现优先级

按以下顺序推进：

1. 领域模型。
2. 核心接口。
3. 文件系统 ToolStore Adapter。
4. 简单关键词 Retriever。
5. ToolMemoryService。
6. 一个非常小的本地 Executor，用于可信 Demo Tool。
7. 验证流程。
8. 执行历史。
9. 简单 CLI 或示例 Agent。
10. MCP Server Adapter。
11. Eino 或其他框架 Adapter。
12. 语义/向量检索。
13. 工具版本管理。
14. 失败分析和自动修复。

如果用户要求的改动不符合这个顺序，也要保持改动足够小，并在 PR 或提交信息中说明取舍。

## 给 Coding Agent 的最终规则

写代码前，必须先判断当前改动属于哪一层：

- 领域模型
- Port/接口
- 应用服务
- 基础设施 Adapter
- API 门面
- CLI/示例
- 文档

然后把改动限制在对应层内。只有当边界确实需要配套调整时，才允许小范围修改其他层。
