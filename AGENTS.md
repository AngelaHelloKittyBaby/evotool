# EvoTool Agent 开发指南

本文档用于约束后续 Coding Agent 在 EvoTool 仓库中的设计和编码行为。任何 Agent 在编写或修改代码前，都应该先阅读并遵守本文档。

EvoTool 的定位是：

> A capability memory layer that allows AI agents to discover, reuse, and evolve generated tools.

中文表达：

> 一个让 Agent 学会积累、复用和演进能力的能力记忆层。

EvoTool 不是简单的 Tool Cache。它的目标是围绕 Agent 生成工具这件事，形成完整闭环：Tool Memory + Library Reuse + Registry + Benchmark + Security + Lifecycle + Adapter + Docs + Real Cases。

## 总体框架

EvoTool 应该被设计成 Agent Capability Memory，而不是单纯的工具文件夹。

整个系统分为五层：

```text
+------------------------------------------------------------------+
| External Agent Runtime                                           |
|                                                                  |
|  Custom Agent / ReAct / Eino / OpenHands / MCP Client / CLI      |
+-----------------------------------+------------------------------+
                                    |
                                    v
+------------------------------------------------------------------+
| EvoTool API / Facade                                             |
|                                                                  |
|  LearnTool()  SearchTools()  ExecuteTool()  RecordExecution()    |
+-----------------------------------+------------------------------+
                                    |
                                    v
+------------------------------------------------------------------+
| Application Services                                             |
|                                                                  |
|  CapabilityMemoryService                                         |
|  RetrievalService                                                |
|  LearningService                                                 |
|  ValidationService                                               |
|  ExecutionService                                                |
|  PolicyService                                                   |
|  LifecycleService                                                |
|  BenchmarkService                                                |
+-----------------------------------+------------------------------+
                                    |
                                    v
+------------------------------------------------------------------+
| Ports                                                            |
|                                                                  |
|  ToolStore             LibraryStore          RegistryStore       |
|  ToolRetriever         Embedder              SimilarityDetector  |
|  ToolGenerator         ToolValidator         ToolExecutor        |
|  Sandbox               PolicyEngine          Reviewer            |
|  DependencyResolver    BenchmarkRunner       AuditLogger         |
+-----------------------------------+------------------------------+
                                    |
                                    v
+------------------------------------------------------------------+
| Adapters                                                         |
|                                                                  |
|  filesystem / sqlite / vector db / docker / local sandbox        |
|  openai generator / python executor / mcp server / eino adapter  |
+------------------------------------------------------------------+
```

五层职责：

- External Agent Runtime：外部 Agent 运行时，例如自定义 ReAct Agent、Eino、OpenHands、MCP Client、CLI。
- EvoTool API / Facade：对外暴露简单稳定的能力接口，例如学习工具、检索工具、执行工具、记录结果。
- Application Services：业务用例编排层，负责检索、学习、验证、执行、安全、生命周期和 Benchmark。
- Ports：核心接口层，定义 EvoTool 需要的存储、检索、生成、执行、沙箱、安全、审计等能力。
- Adapters：外部技术适配层，接入文件系统、SQLite、向量数据库、Docker、MCP、Eino、OpenHands、LLM SDK 等。

能力记忆内部再分为三层：

```text
Agent Capability Memory

  tools/        # 用户或 Agent 可以直接调用的顶层能力
  libraries/    # 多个 Tool 共享的底层能力和公共代码
  registry/     # 元数据、向量索引、依赖关系、版本、历史和安全策略
```

三层职责：

- Tool 层：表示 Agent 可以直接调用的能力，例如 `pdf_to_excel`、`csv_analyzer`、`report_generator`。
- Library 层：表示多个 Tool 共享的底层能力，例如 `pdf_parser`、`excel_writer`、`text_processor`。
- Registry 层：表示能力目录和检索中心，保存 metadata、embedding、version、execution_history、dependency_graph、policy、benchmark 等信息。

关系示意：

```text
                 Tool
                  |
        +---------+---------+
        |                   |
      uses                uses
        |                   |
    Library A          Library B
```

设计目标不是“把代码保存下来”，而是像软件生态一样管理 Agent 生成出来的能力、依赖、版本和复用关系。

## 项目定位

EvoTool 是一个面向 AI Agent 的能力记忆层。它的核心职责是：让 Agent 能够在任务执行过程中保存、检索、验证、执行并复用自己生成过的工具，而不是每次遇到相似任务都重新生成代码。

必须始终围绕这条主线设计：

```text
Agent 发现能力缺口
  -> 生成 Tool
  -> 识别是否有可复用 Library
  -> 验证 Tool 和 Library
  -> 安全审查
  -> 保存到 Capability Memory
  -> 更新 Registry
  -> 未来多级检索
  -> 复用 Tool 或 Library
  -> 记录执行效果
  -> 版本演进、重构或废弃
```

项目要证明的价值不是“能保存文件”，而是：

- Agent 可以积累能力。
- 相似任务可以复用已有工具。
- 多个工具可以共享底层库，减少重复代码。
- 重复工具生成次数下降。
- 任务耗时和 token 成本下降。
- 生成工具经过验证、安全控制和生命周期管理。
- 能以 Adapter 方式接入 Eino、OpenHands、MCP 等 Agent 生态。

## 为什么不直接用 Skill

后续实现和文档必须明确回答这个问题：为什么不用已有 Skill 机制？

区别如下：

```yaml
Skill:
  creator: 开发者
  flow: 开发者创建能力 -> Agent 使用能力
  nature: 人工定义的可复用能力

Learned Tool:
  creator: Agent
  flow: Agent 在任务中生成能力 -> 验证后保存 -> 未来 Agent 使用能力
  nature: Agent 自动产生并演进的可复用能力
```

Skill 更像人工维护的能力模块。EvoTool 的 Learned Tool 更像 Agent 在任务执行中获得的能力记忆。

因此，EvoTool 不能只做 Skill Registry，也不能只是 Tool Registry。它必须覆盖以下问题：

- 工具为什么被生成？
- 工具如何命名？
- 工具如何验证？
- 工具是否允许保存？
- 工具是否允许执行？
- 工具依赖哪些 Library？
- 是否存在重复代码可以抽取为 Library？
- 未来任务如何检索它？
- 失败后如何记录、降级、修复或升级？

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
    evotool/                       # CLI 入口

  internal/
    domain/                        # 领域模型
      tool.go
      library.go
      manifest.go
      task.go
      retrieval.go
      execution.go
      validation.go
      version.go
      lifecycle.go
      policy.go
      benchmark.go
      registry.go
      dependency.go
      similarity.go

    ports/                         # 核心接口
      store.go
      library_store.go
      registry.go
      retriever.go
      generator.go
      validator.go
      executor.go
      sandbox.go
      embedder.go
      policy.go
      reviewer.go
      benchmark.go
      dependency_resolver.go
      similarity_detector.go
      audit.go
      mcp.go

    service/                       # 业务用例编排
      capability_memory_service.go
      retrieval_service.go
      learning_service.go
      validation_service.go
      execution_service.go
      policy_service.go
      lifecycle_service.go
      benchmark_service.go
      registry_service.go
      library_service.go
      dependency_service.go

    adapter/                       # 外部技术实现
      store/
        filesystem/
        sqlite/
      retriever/
        keyword/
        vector/
        hybrid/
      generator/
        openai/
      executor/
        python/
        shell/
      sandbox/
        local/
        docker/
      similarity/
        ast/
        embedding/
      benchmark/
        local/
      mcp/
        server/
        client/
      eino/
      openhands/

    api/                           # 对外门面
      evotool.go

  examples/
    simple-agent/
    mcp-server/
    eino-adapter/
    openhands-adapter/
    pdf-to-excel/
    webpage-to-markdown/

  docs/
    architecture.md
    security.md
    benchmark.md
    mcp.md
    lifecycle.md
    adapters.md
    naming.md
    retrieval.md
    dependency-management.md
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

## 工具命名要求

工具命名非常重要。Agent 后续不是靠文件名调用工具，而是通过任务意图、embedding、metadata 和执行历史进行匹配。

工具名称必须满足：

- 使用稳定的 `snake_case`。
- 名字能表达核心能力。
- 优先使用 `领域_动作_对象` 或 `输入_to_输出` 风格。
- 不允许使用 `tool1`、`test`、`helper`、`abc_tool`、`pdf` 这种无语义名称。
- 名称必须和 description、category、input、output、capabilities 一致。

推荐示例：

```text
pdf_table_extractor
pdf_to_excel
csv_analyzer
excel_formatter
sentiment_report_generator
webpage_to_markdown
```

不推荐示例：

```text
tool1
test
helper
abc_tool
pdf
main_tool
```

Manifest 至少要表达这些信息：

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
  },
  "capabilities": ["pdf_table_extraction", "xlsx_generation"]
}
```

## 工具目录管理要求

一个 Tool 必须对应一个独立目录。不要把所有工具平铺成一堆脚本文件。

禁止：

```text
tools/
  tool1.py
  tool2.py
  tool3.py
```

推荐：

```text
tools/
  pdf_table_extractor/
    manifest.json
    main.py
    requirements.txt
    tests/
    README.md

  csv_analyzer/
    manifest.json
    main.py
    tests/

  image_compressor/
    manifest.json
    main.py
```

每个 Tool 都应该被当成一个独立能力单元，类似一个轻量软件包或 Docker 镜像。

## Tool 与 Library 分层

不要把所有可复用代码都建模成 Tool。

```text
Agent Capability System

        |
   +----+----+
   |         |
 Tools   Libraries
 工具     公共能力
```

Tool 层负责用户或 Agent 可直接调用的能力：

```text
pdf_to_excel
csv_analysis
excel_format
report_generator
```

Library 层负责多个 Tool 共享的底层代码：

```text
libraries/
  pdf_parser/
    manifest.json
    parser.py
    extractor.py
    tests/

  excel_writer/
    manifest.json
    writer.py
    formatter.py
    tests/

  text_processor/
    manifest.json
    cleaner.py
    tokenizer.py
    tests/
```

Tool 可以依赖 Library：

```python
from shared.pdf_parser import extractor
from shared.excel_writer import writer
```

规则：

- Tool 是可被 Agent 直接检索和调用的顶层能力。
- Library 默认不直接暴露给用户任务调用。
- Library 必须有版本、manifest、测试和依赖声明。
- Tool 对 Library 的依赖必须被记录进 Registry。
- Library 变更必须触发依赖它的 Tool 重新验证。

## Shared Library 来源

Shared Library 有两个来源。

第一类：开发者预置基础库。

类似 MCP Server 或内置 capability，例如：

```text
filesystem
database
http
excel
pdf
text
```

第二类：Agent 发现重复代码后抽取出来。

例如第一次生成：

```text
pdf_to_excel
  extract_table()
```

第二次又生成：

```text
pdf_summary
  extract_table()
```

系统通过 Code Similarity Detection 发现两个 `extract_table()` 功能相似，然后进入候选重构流程：

```text
pdf_to_excel.extract_table
pdf_summary.extract_table
  -> similarity score 0.95
  -> propose shared library pdf_table_extractor_core
  -> validate dependent tools
  -> save library
  -> update dependency graph
```

注意：自动抽取 Shared Library 是第二阶段能力。MVP 可以先记录重复代码和相似度，不要直接自动重构生产工具。

## 多阶段检索要求

工具检索必须分阶段完成，不能把全部工具都交给 LLM。

禁止设计：

```text
用户任务
  -> 把 10000 个工具全部塞给 LLM
  -> 让 LLM 自己挑
```

推荐设计：

```text
                 User Task
                     |
                     v
              Intent Extraction
                     |
                     v
             Semantic Retrieval
                     |
                  Top-K
                     |
                     v
             Metadata Filter
                     |
                     v
              Capability Rank
                     |
                     v
               LLM Selection
                     |
                     v
                   Tool
```

示例：

```text
20,000 tools
  -> semantic retrieval
  -> 50 candidates
  -> input/output/type/permission/environment filter
  -> 10 candidates
  -> success rate + usage count + latency + version ranking
  -> 3 candidates
  -> LLM or rule selection
  -> 1 tool
```

检索阶段职责：

- Intent Extraction：从任务中提取 intent、input、output、domain、environment、constraints。
- Semantic Retrieval：基于 manifest embedding 做大规模召回。
- Metadata Filter：按 input/output schema、runtime、权限、环境、依赖状态过滤。
- Capability Rank：根据使用经验和可靠性进行排序。
- LLM Selection：只把少量高质量候选交给 LLM 或规则选择器。

Embedding 不能只基于工具名字。Embedding 文本应该主要由以下内容组合：

```text
name
+ description
+ tags
+ input/output schema
+ examples
+ capabilities
```

排序不能只看语义相似度。最终分数应该综合：

```text
Score =
  semantic_similarity
  + success_rate
  + reliability
  + latency
  + usage_history
  + environment_match
```

示例：

```text
Tool A: similarity 0.91, success_rate 99%, average_latency 1.2s, usage_count 1200
Tool B: similarity 0.95, success_rate 72%, average_latency 4.8s, usage_count 15
```

不能因为 B 的相似度更高就直接选 B。EvoTool 要做到：不仅知道“哪个工具像”，还知道“哪个工具更可靠”。

在其他条件接近时，优先选择成功率高、使用次数多、最近验证通过、依赖健康、环境匹配、风险更低的工具。
## Registry 要求

Registry 是 EvoTool 的能力目录，不只是一个文件列表。

Registry 至少应该保存：

- Tool metadata
- Library metadata
- Embedding index
- Category index
- Version records
- Execution history
- Dependency graph
- Validation status
- Security policy
- Review status
- Benchmark records

推荐结构：

```text
registry/
  metadata/
  embeddings/
  versions/
  execution_history/
  dependency_graph/
  policies/
  benchmarks/
```

## 核心领域概念

后续代码中应统一使用以下概念。

### Tool

表示一个 EvoTool 学到或注册的可复用顶层能力。

建议字段：

- ID
- Name
- Description
- Category
- Manifest
- CurrentVersion
- Versions
- Dependencies
- LifecycleStatus
- TrustLevel
- CreatedAt
- UpdatedAt

### Library

表示多个 Tool 共享的底层能力或公共代码。

建议字段：

- ID
- Name
- Description
- Category
- Manifest
- CurrentVersion
- Versions
- UsedByTools
- LifecycleStatus
- TrustLevel
- CreatedAt
- UpdatedAt

### ToolManifest

表示 Tool 的机器可读契约。

建议字段：

- Name
- Description
- Category
- Runtime
- EntryPoint
- Inputs
- Outputs
- Capabilities
- Dependencies
- LibraryRefs
- Tags
- Permissions
- ResourceLimits

### LibraryManifest

表示 Library 的机器可读契约。

建议字段：

- Name
- Description
- Category
- Runtime
- Exports
- Dependencies
- Tags
- Permissions
- ResourceLimits

### TaskSpec

表示用户任务或 Agent 任务的标准化描述。

建议字段：

- Intent
- InputSummary
- Constraints
- ExpectedOutput
- Domain
- Metadata

### ToolCandidate

表示一次检索得到的候选工具及其匹配信息。

建议字段：

- Tool
- Score
- MatchReason
- Source
- RiskSummary
- DependencyHealth
- HistoricalSuccessRate

### GeneratedTool

表示刚生成、还没有进入可信工具记忆的新工具。

建议字段：

- Manifest
- SourceFiles
- Tests
- Readme
- Dependencies
- LibraryCandidates
- GenerationReason

### ValidationResult

表示工具验证结果。

建议字段：

- Valid
- Errors
- Warnings
- TestOutput
- SandboxReport
- PolicyReport
- DependencyReport

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
- TokenUsage
- Metadata

### DependencyGraph

表示 Tool 和 Library 之间的依赖关系。

建议字段：

- Nodes
- Edges
- VersionConstraints
- HealthStatus

### SimilarityReport

表示代码或能力相似度检测结果。

建议字段：

- SourceA
- SourceB
- SimilarityScore
- Reason
- SuggestedAction

### ToolLifecycle

表示工具从生成到废弃的状态。

建议状态：

- Draft：刚生成，尚未验证。
- Validated：验证通过，但尚未人工或策略批准。
- Approved：允许被自动检索和执行。
- Quarantined：发现风险或多次失败，暂停使用。
- Deprecated：已有更好版本，不再推荐。
- Deleted：被移除。

### ToolPolicy

表示工具的权限、安全和资源限制。

建议字段：

- AllowedPaths
- DeniedPaths
- NetworkPolicy
- MaxRuntime
- MaxMemory
- MaxOutputSize
- AllowedCommands
- DeniedCommands
- RequiresReview

### BenchmarkResult

表示 EvoTool 对任务成本和复用效果的评估结果。

建议指标：

- TotalTasks
- ToolGeneratedCount
- ToolReusedCount
- LibraryReusedCount
- DuplicateGenerationCount
- AverageLatency
- AverageTokenUsage
- SuccessRate
- FailureRate

## 核心服务职责

### CapabilityMemoryService

统一对外提供能力记忆操作。

职责：

- 搜索能力。
- 保存能力。
- 查询能力详情。
- 记录执行结果。
- 协调 Tool 和 Library。

### RetrievalService

负责多级检索。

职责：

- 构造检索请求。
- 执行 category / keyword / embedding 召回。
- 做 metadata、policy、dependency 过滤。
- 根据历史表现排序。

### LearningService

负责从任务中学习新工具。

职责：

- 判断能力缺口。
- 调用 ToolGenerator。
- 生成 Manifest。
- 查找可复用 Library。
- 触发验证和策略检查。
- 保存 Tool 并更新 Registry。

### ValidationService

负责验证 Tool 或 Library 是否可用。

职责：

- Schema 检查。
- 测试执行。
- 依赖检查。
- 沙箱报告汇总。

### ExecutionService

负责执行工具。

职责：

- 检查工具状态。
- 检查依赖健康。
- 调用 PolicyService。
- 调用 Executor / Sandbox。
- 记录 ExecutionResult。

### PolicyService

负责安全策略。

职责：

- 权限检查。
- 风险评级。
- 判断是否需要 Review。
- 拒绝危险执行。

### LifecycleService

负责生命周期和版本演进。

职责：

- Draft / Validated / Approved / Quarantined / Deprecated 状态流转。
- 版本提升。
- 失败次数监控。
- 依赖变更影响分析。

### BenchmarkService

负责证明 EvoTool 是否真的有效。

职责：

- 运行 benchmark suite。
- 对比 baseline 和 EvoTool 模式。
- 统计 token、耗时、复用次数、失败率、安全拦截次数。

## 核心接口

核心接口统一放在 `internal/ports` 中。接口要小而专注，避免大而全。

接口设计方向示例：

```go
type IntentExtractor interface {
    Extract(ctx context.Context, task domain.TaskSpec) (domain.RetrievalQuery, error)
}

type ToolRetriever interface {
    Retrieve(ctx context.Context, query domain.RetrievalQuery) (domain.RetrievalResult, error)
}

type CapabilityRanker interface {
    Rank(ctx context.Context, query domain.RetrievalQuery, candidates []domain.ToolCandidate) ([]domain.ToolCandidate, error)
}

type ToolSelector interface {
    Select(ctx context.Context, task domain.TaskSpec, candidates []domain.ToolCandidate) (domain.ToolCandidate, error)
}

type LibraryRetriever interface {
    SearchLibraries(ctx context.Context, query domain.RetrievalQuery) ([]domain.LibraryCandidate, error)
}

type ToolStore interface {
    Save(ctx context.Context, tool domain.Tool) error
    Get(ctx context.Context, id string) (domain.Tool, error)
    Update(ctx context.Context, tool domain.Tool) error
}

type LibraryStore interface {
    Save(ctx context.Context, library domain.Library) error
    Get(ctx context.Context, id string) (domain.Library, error)
    Update(ctx context.Context, library domain.Library) error
}

type ToolGenerator interface {
    Generate(ctx context.Context, task domain.TaskSpec) (domain.GeneratedTool, error)
}

type ToolExecutor interface {
    Execute(ctx context.Context, tool domain.Tool, input map[string]any) (domain.ExecutionResult, error)
}
```

不要创建巨大的接口。如果某个实现并不自然需要接口中的所有方法，就拆分接口。
## MVP 主流程

第一版稳定 MVP 应该先实现以下闭环：

```text
TaskSpec
  -> 多级检索 learned tools
  -> 如果找到匹配工具：
       检查依赖状态
       执行安全策略检查
       执行工具
       记录执行结果
  -> 如果没有找到匹配工具：
       生成工具
       检查命名和 Manifest
       识别可复用 Library
       在沙箱中验证工具
       执行安全策略检查
       保存工具和元数据
       更新 Registry
       执行工具
       记录执行结果
```

不要一开始就做自动修复、自动抽取公共库、分布式执行、多 Agent 协作。那些属于后续阶段。

## 核心运行流程

EvoTool 至少有四条核心流程。后续代码必须围绕这些流程拆分职责，不要把所有逻辑写进一个对象或一个函数。

### 工具复用流程

```text
TaskSpec
  -> Intent Extraction
  -> Build RetrievalQuery
  -> Semantic Retrieval, large-scale recall
  -> Metadata Filter
  -> Capability Rank
  -> LLM or rule Selection
  -> Check dependency health
  -> Policy Check
  -> Execute Tool
  -> Record ExecutionResult and usage stats
```

如果找到合适工具，就不再重新生成工具。

### 工具学习流程

```text
TaskSpec
  -> Capability gap detected
  -> Tool Generator
  -> Dependency Analysis
  -> Search Library first
  -> If library found: reuse
  -> If library missing: create candidate library only when needed
  -> Generate tool source and manifest
  -> Validate naming, schema, dependencies, and tests
  -> Policy Check
  -> Review if high risk
  -> Save Tool / Library
  -> Update Registry and Dependency Graph
  -> Execute Tool
  -> Record ExecutionResult and usage stats
```

第一版重点是生成 Tool 前先搜索已有 Library，而不是生成之后立刻自动重构已有代码。

### Library 复用流程

```text
Generate Tool
  -> Dependency Analysis
  -> Search Library
  -> If found: reuse Library
  -> If not found: create Library candidate
  -> Validate dependent Tool with resolved dependencies
  -> Record Tool -> Library dependency
```

MVP 阶段可以只做“生成前搜索并复用 Library”。已有 Tool 之间的自动 Library Extraction 属于第二阶段。

### 生命周期演进流程

```text
Tool v1
  -> Execute
  -> Record success/failure
  -> Detect repeated failures
  -> Quarantine if unhealthy
  -> Generate or repair v2 later
  -> Validate v2
  -> Policy check
  -> Promote v2 as current
  -> Deprecate v1
```

第一版不做自动修复。失败只需要被记录、统计、影响排序，并为后续 Evolution 留接口。
## Benchmark 要求

EvoTool 必须通过 Benchmark 证明它的价值。不能只说“更智能”或“更高效”。

Benchmark 需要比较至少两组：

```text
Baseline：没有 Tool Memory，每次由 Agent 重新生成工具。
EvoTool：先检索 Tool Memory，找不到才生成工具。
```

必须记录的指标：

- 工具生成次数
- 重复工具生成次数
- 工具复用次数
- Library 复用次数
- 平均任务耗时
- 平均 token 使用量
- 成功率
- 失败率
- 因安全策略被拦截的执行次数
- 因依赖失效导致的失败次数

README、docs 和示例中可以使用类似表达：

```text
EvoTool reduces redundant tool generation and enables agents to accumulate reusable capabilities across tasks.
```

但真正进入稳定版本前，必须用可复现实验数据支持这个结论。

## 安全要求

安全是 EvoTool 的核心问题之一，不是附加功能。

危险示例：

```text
Agent 自动生成 delete_database.py
  -> 工具被保存
  -> 下次用户说“清理数据”
  -> Agent 误调用该工具
  -> 造成破坏性后果
```

因此，生成出来的工具在验证和策略批准之前都必须被视为不可信。

必须支持或预留以下安全机制：

- Sandbox：生成工具必须通过沙箱执行边界。
- Permission：工具必须声明权限，运行前必须检查权限。
- Review：高风险工具必须进入人工或策略审查流程。
- Execution Limit：必须限制执行时间、内存、输出大小和文件访问范围。
- Dependency Isolation：工具依赖必须隔离安装或隔离解析，不能污染宿主环境。
- Audit Log：工具生成、验证、执行、失败、升级都必须可追踪。

安全规则：

- 生成代码不能绕过 Validator 和 Executor 直接执行。
- 默认情况下，生成工具不能访问任意宿主机路径。
- 除非配置显式允许，否则不能把密钥传给生成工具。
- 沙箱策略必须显式表达。
- 必须记录足够的执行元数据，方便排查失败。
- 生成工具声明的依赖必须视为不可信输入。
- 涉及删除、覆盖、网络请求、数据库写入、系统命令的工具默认需要 Review。
- Library 更新必须触发依赖 Tool 的重新验证。

第一版 MVP 可以先使用本地 Executor，但架构上必须允许之后替换成 Docker 或其他沙箱实现。

## Tool Lifecycle 要求

EvoTool 不能只保存工具代码，还必须管理工具生命周期。

需要回答的问题：谁负责维护 Agent 生成出来的工具？

例如：

```yaml
pdf_to_excel:v1
```

一年以后可能出现：

- Python 版本升级。
- 依赖包失效。
- 新 PDF 格式不兼容。
- 原工具安全策略不再满足。
- 依赖 Library 出现破坏性更新。
- 有更好的 v2 版本。

因此，工具必须有生命周期和版本管理。

建议流程：

```text
Generate v1
  -> Validate
  -> Approve
  -> Execute
  -> Record success/failure
  -> Detect repeated failures
  -> Quarantine or Repair
  -> Generate v2
  -> Validate v2
  -> Promote v2 as current
  -> Deprecate v1
```

实现要求：

- 每个工具必须有版本。
- 每个 Library 必须有版本。
- 当前版本必须显式标记。
- 执行历史必须记录版本号。
- 多次失败的工具不能继续被静默复用。
- 高风险工具不能自动升级为 Approved。
- 依赖变更必须进入验证流程。
- Library 版本升级必须记录影响范围。

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
- `evotool.review_tool`
- `evotool.run_benchmark`
- `evotool.search_libraries`
- `evotool.get_dependency_graph`

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

Eino/OpenHands Adapter 的价值在于证明 EvoTool 能进入真实 Agent 生态，而不是只停留在 demo。

## 实际案例要求

EvoTool 必须有实际案例。案例要能体现“先生成、再复用、再记录效果”。

优先案例：

- PDF 表格提取为 Excel。
- 网页内容转换为 Markdown。
- CSV 数据统计分析。
- 项目日志分析。
- 多 API 数据聚合生成报告。

每个案例至少包含：

- 第一次任务：没有工具，生成并验证工具。
- 第二次任务：检索并复用工具。
- Library 复用：展示多个工具如何共享底层库。
- Benchmark：对比是否减少生成次数、耗时和 token。
- 安全策略：说明工具获得了哪些权限、被限制了哪些行为。

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
- Library 搜索和依赖解析
- 保存和读取行为
- 验证成功/失败行为
- 安全策略允许/拒绝行为
- 生命周期状态流转
- 执行结果记录
- Benchmark 指标计算
- 相似度检测结果转换
- 错误传播
- Adapter 边界转换

Adapter 测试可以使用 fake 或临时目录。普通单元测试不能强依赖 Docker、网络、付费 API 或外部服务。

如果集成测试需要外部服务，必须明确标记，并保持可选。

## 持久化要求

第一版可以从简单文件系统存储开始，但存储必须可替换。

可接受的本地存储结构：

```text
.evotool/
  tools/
    <tool-name>/
      manifest.json
      versions/
        v1/
          main.py
          README.md
          tests/
      metadata.json
      policy.json
      history.jsonl
      benchmark.jsonl

  libraries/
    <library-name>/
      manifest.json
      versions/
        v1/
          source/
          tests/
      metadata.json
      policy.json

  registry/
    metadata/
    embeddings/
    versions/
    execution_history/
    dependency_graph/
    policies/
    benchmarks/
```

规则：

- 不要把文件系统布局硬编码进 Service。
- 文件系统细节放在 Store Adapter 中。
- 必须保存足够的元数据，以支持未来语义检索和版本管理。
- 执行历史应使用便于追加写入的格式。
- 策略、权限和审计记录必须可追踪。
- Tool 和 Library 的依赖关系必须可查询。

## 文档要求

新增重要概念时，需要同步更新文档。

重要文档：

- `README.md`：项目定位、核心价值和快速开始。
- `docs/architecture.md`：架构和设计决策。
- `docs/security.md`：安全模型、权限、沙箱和审查流程。
- `docs/benchmark.md`：Benchmark 方法、指标和结果。
- `docs/mcp.md`：MCP Tool 名称、Payload 和示例。
- `docs/lifecycle.md`：工具生命周期和版本演进策略。
- `docs/adapters.md`：Eino、OpenHands 等 Adapter 的接入方式。
- `docs/naming.md`：Tool 和 Library 命名规范。
- `docs/retrieval.md`：多级检索策略。
- `docs/dependency-management.md`：Tool/Library 依赖和共享代码复用策略。
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
- 在相似度检测稳定之前自动重构共享库。
- 从 Generator 直接执行生成代码。
- 保存未验证、未审查的高风险工具为 Approved。
- 让生成工具默认拥有网络、数据库写入或任意文件删除权限。
- 使用 `tool1.py`、`helper.py` 这类无语义命名。
- 将多个 Tool 的共享代码复制粘贴而不记录重复。
- 吞掉失败信息而不记录。

## MVP 边界

第一版不要碰自动代码重构和自动修复。先把能力注册、检索、依赖、复用和真实接入做扎实。

第一版必须完成：

1. Tool Registry。
2. Library Registry。
3. Tool / Library Manifest。
4. Semantic Retrieval 基础能力。
5. Metadata Filter。
6. Capability Rank，至少包含 similarity、success_rate、latency、usage_count、environment_match。
7. Dependency Graph。
8. 生成 Tool 前先搜索已有 Library。
9. ExecutionResult 和 ToolUsageStats 记录。
10. 基础安全策略模型。
11. 一个本地可信 Demo Executor。
12. 一个实际案例。
13. Eino Adapter 的最小可运行版本。

第一版明确不做：

- 自动代码重构。
- 自动 Library Extraction。
- 自动修复。
- 分布式沙箱。
- 复杂多租户权限系统。
- 多 Agent 协作。

第二阶段再做：

- Code Similarity Detection。
- Shared Library 候选抽取。
- 自动重构建议。
- Docker Sandbox。
- MCP Server。
- OpenHands Adapter。
- 失败分析和自动修复。
## 实现优先级

按以下顺序推进：

1. 领域模型。
2. 核心接口。
3. Tool/Library 命名和 Manifest 规范。
4. Tool Registry。
5. Library Registry。
6. Semantic Retrieval 基础能力。
7. Metadata Filter。
8. Capability Rank。
9. Dependency Graph。
10. ToolMemoryService。
11. RegistryService。
12. 安全策略模型。
13. 一个非常小的本地 Executor，用于可信 Demo Tool。
14. 验证流程。
15. 执行历史和 ToolUsageStats。
16. 简单 CLI 或示例 Agent。
17. Benchmark Runner。
18. Eino Adapter。
19. MCP Server Adapter。
20. OpenHands Adapter。
21. 代码相似度检测。
22. 共享 Library 候选抽取。
23. 工具版本管理增强。
24. 失败分析和自动修复。

如果用户要求的改动不符合这个顺序，也要保持改动足够小，并在 PR 或提交信息中说明取舍。
## 给 Coding Agent 的最终规则

写代码前，必须先判断当前改动属于哪一层：

- 领域模型
- Port/接口
- 应用服务
- 基础设施 Adapter
- API 门面
- CLI/示例
- Benchmark
- 安全策略
- Tool/Library 依赖管理
- MCP 集成
- Agent 框架 Adapter
- 文档

然后把改动限制在对应层内。只有当边界确实需要配套调整时，才允许小范围修改其他层。



