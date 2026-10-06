# EvoTool 架构设计

EvoTool 的核心定位是 **Agent Capability Memory**：一个让 Agent 学会积累、复用和演进能力的能力记忆层。

它不是一个简单的 Tool Cache，也不是一个普通 Skill Registry。EvoTool 要解决的是：Agent 在任务中生成出来的工具，如何被命名、验证、安全审查、持久化、检索、复用、依赖管理、版本演进，并最终成为长期可维护的能力资产。

## 1. 总体架构

EvoTool 的整体框架分为五层：

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

依赖方向必须始终向内：

```text
cmd/examples/adapters -> api -> service -> ports/domain
adapter implementations -> ports/domain
```

核心规则：

- `domain` 定义稳定领域模型。
- `ports` 定义核心需要的接口。
- `service` 编排业务流程。
- `adapter` 接入文件系统、数据库、LLM、MCP、Eino、OpenHands、Docker 等外部技术。
- `api` 对外提供简洁门面。
- `cmd` 和 `examples` 只负责装配，不写核心业务。

## 2. 能力记忆框架

EvoTool 的能力记忆由三部分组成：

```text
Agent Capability Memory

  tools/        # 用户或 Agent 可以直接调用的顶层能力
  libraries/    # 多个 Tool 共享的底层能力和公共代码
  registry/     # 元数据、索引、依赖、版本、历史、安全策略
```

### 2.1 Tool 层

Tool 是 Agent 可以直接选择和执行的能力。

示例：

```text
pdf_to_excel
csv_analyzer
excel_formatter
sentiment_report_generator
webpage_to_markdown
```

每个 Tool 必须拥有：

- 语义化名称
- 独立目录
- `manifest.json`
- 入口文件
- 测试
- README
- 版本
- 权限策略
- 执行历史
- 生命周期状态

### 2.2 Library 层

Library 是多个 Tool 共享的底层能力或公共代码。

示例：

```text
pdf_parser
excel_writer
text_processor
data_cleaner
```

Library 默认不直接暴露给用户任务调用。它主要用于减少重复代码，并让 Agent 生成出来的能力像软件生态一样管理依赖、版本和复用。

Tool 可以依赖 Library：

```text
pdf_to_excel
  -> pdf_parser
  -> excel_writer

csv_report_generator
  -> data_cleaner
  -> excel_writer
  -> report_renderer
```

### 2.3 Registry 层

Registry 是能力目录和检索中心，不是普通文件列表。

Registry 负责保存：

- Tool metadata
- Library metadata
- Category index
- Embedding index
- Version records
- Execution history
- Dependency graph
- Validation status
- Security policy
- Review status
- Benchmark records
- Similarity reports

Registry 是未来检索、复用、排名、审计和版本演进的核心。

## 3. 推荐目录结构

仓库代码结构：

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

本地能力存储结构：

```text
.evotool/
  tools/
    pdf_to_excel/
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
    pdf_parser/
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
    similarity_reports/
```

注意：文件系统布局属于 Store Adapter 细节，不能硬编码进 Service。

## 4. 核心运行流程

EvoTool 有四条核心流程。

### 4.1 工具复用流程

当 Agent 收到任务时，优先搜索已有能力。

```text
TaskSpec
  -> Normalize intent
  -> Build RetrievalQuery
  -> Search Registry
  -> Retrieve Tool candidates
  -> Filter by metadata
  -> Filter by policy
  -> Check dependency health
  -> Rank by score + success history
  -> Select Tool
  -> Execute Tool
  -> Record ExecutionResult
```

如果找到合适工具，就不再重新生成工具。

### 4.2 工具学习流程

当没有合适工具时，Agent 才生成新工具。

```text
TaskSpec
  -> Capability gap detected
  -> Generate tool source
  -> Generate manifest
  -> Validate naming and manifest
  -> Detect reusable libraries
  -> Validate generated tool
  -> Run policy check
  -> Review if high risk
  -> Save Tool
  -> Update Registry
  -> Execute Tool
  -> Record ExecutionResult
```

这条链路是 EvoTool 的核心价值：Agent 不只是完成任务，还把本次任务中产生的新能力沉淀下来。

### 4.3 Library 复用流程

Tool 生成后，不应该把所有公共代码复制进自己目录。

```text
GeneratedTool
  -> Analyze imports and helper functions
  -> Search existing Libraries
  -> Match by category, description, exports, embedding
  -> Reuse Library if compatible
  -> Record Tool -> Library dependency
  -> Validate Tool with resolved dependencies
```

MVP 阶段可以只做“发现并记录可复用 Library”。自动抽取共享库属于第二阶段。

### 4.4 生命周期演进流程

工具不能只保存不维护。

```text
Tool v1
  -> Execute
  -> Record success/failure
  -> Detect repeated failures
  -> Quarantine if unhealthy
  -> Generate or repair v2
  -> Validate v2
  -> Policy check
  -> Promote v2 as current
  -> Deprecate v1
```

Library 也必须有生命周期。Library 升级后，依赖它的 Tool 必须重新验证。

## 5. 多级检索框架

工具检索不能只搜索 Tool 名字。

推荐检索链路：

```text
用户任务
  -> 能力分类召回
  -> Keyword / BM25 召回
  -> Embedding 语义召回
  -> Metadata 过滤
  -> 安全策略过滤
  -> 依赖状态过滤
  -> 执行历史排序
  -> LLM 或规则最终选择
```

示例任务：

```text
分析用户评论情绪，然后生成报告
```

分类召回：

```text
NLP
Data Analysis
Report Generation
```

Metadata 过滤：

```json
{
  "input": "csv",
  "output": "markdown",
  "domain": "customer_feedback"
}
```

排序规则可以综合：

- 语义相似度
- category 是否匹配
- input/output 是否匹配
- success rate
- usage count
- last validated time
- dependency health
- risk level

在其他条件相近时，优先选择成功率高、使用次数多、最近验证通过、依赖健康、风险更低的工具。

## 6. Tool 命名和 Manifest

命名直接影响未来检索质量。

推荐命名：

```text
pdf_table_extractor
pdf_to_excel
csv_analyzer
excel_formatter
sentiment_report_generator
webpage_to_markdown
```

禁止命名：

```text
tool1
test
helper
abc_tool
pdf
main_tool
```

Manifest 示例：

```json
{
  "name": "pdf_table_extractor",
  "description": "Extract tables from PDF documents and export to Excel format",
  "category": "document_processing",
  "runtime": "python",
  "entry_point": "main.py",
  "input": {
    "file": "pdf"
  },
  "output": {
    "file": "xlsx"
  },
  "capabilities": [
    "pdf_table_extraction",
    "xlsx_generation"
  ],
  "dependencies": [
    "pdf_parser:v1",
    "excel_writer:v1"
  ],
  "permissions": {
    "filesystem": "workspace_read_write",
    "network": "none"
  }
}
```

## 7. 领域模型关系

核心领域关系如下：

```text
TaskSpec
  -> RetrievalQuery
  -> ToolCandidate
  -> Tool
       -> ToolManifest
       -> ToolVersion
       -> ToolPolicy
       -> ExecutionHistory
       -> Dependencies
             -> Library
                  -> LibraryManifest
                  -> LibraryVersion

ExecutionResult
  -> ToolID
  -> Version
  -> Duration
  -> TokenUsage
  -> Success/Failure

Registry
  -> Metadata
  -> Embeddings
  -> Versions
  -> DependencyGraph
  -> Policies
  -> Benchmarks
```

第一阶段必须先把这些核心名词稳定下来，再逐步扩展能力。

## 8. 核心服务职责

### CapabilityMemoryService

统一对外提供能力记忆操作。

职责：

- 搜索能力
- 保存能力
- 查询能力详情
- 记录执行结果
- 协调 Tool 和 Library

### RetrievalService

负责多级检索。

职责：

- 构造检索请求
- 执行 category / keyword / embedding 召回
- 做 metadata、policy、dependency 过滤
- 根据历史表现排序

### LearningService

负责从任务中学习新工具。

职责：

- 判断能力缺口
- 调用 ToolGenerator
- 生成 Manifest
- 查找可复用 Library
- 触发验证和策略检查
- 保存 Tool 并更新 Registry

### ValidationService

负责验证 Tool 或 Library 是否可用。

职责：

- Schema 检查
- 测试执行
- 依赖检查
- 沙箱报告汇总

### ExecutionService

负责执行工具。

职责：

- 检查工具状态
- 检查依赖健康
- 调用 PolicyService
- 调用 Executor / Sandbox
- 记录 ExecutionResult

### PolicyService

负责安全策略。

职责：

- 权限检查
- 风险评级
- 判断是否需要 Review
- 拒绝危险执行

### LifecycleService

负责生命周期和版本演进。

职责：

- Draft / Validated / Approved / Quarantined / Deprecated 状态流转
- 版本提升
- 失败次数监控
- 依赖变更影响分析

### BenchmarkService

负责证明 EvoTool 是否真的有效。

职责：

- 运行 benchmark suite
- 对比 baseline 和 EvoTool 模式
- 统计 token、耗时、复用次数、失败率、安全拦截次数

## 9. 核心接口草案

接口放在 `internal/ports`，保持小而专注。

```go
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

type RegistryStore interface {
    UpsertToolMetadata(ctx context.Context, tool domain.Tool) error
    UpsertLibraryMetadata(ctx context.Context, library domain.Library) error
    SaveDependencyGraph(ctx context.Context, graph domain.DependencyGraph) error
}

type ToolRetriever interface {
    Search(ctx context.Context, query domain.RetrievalQuery) ([]domain.ToolCandidate, error)
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

type DependencyResolver interface {
    Resolve(ctx context.Context, tool domain.Tool) (domain.DependencyGraph, error)
}

type SimilarityDetector interface {
    Detect(ctx context.Context, files []domain.SourceFile) ([]domain.SimilarityReport, error)
}

type PolicyEngine interface {
    Evaluate(ctx context.Context, tool domain.Tool, task domain.TaskSpec) (domain.PolicyDecision, error)
}

type BenchmarkRunner interface {
    Run(ctx context.Context, suite domain.BenchmarkSuite) (domain.BenchmarkResult, error)
}
```

## 10. 安全框架

生成代码默认不可信。

执行前必须经过：

```text
Manifest Check
  -> Dependency Check
  -> Static Risk Check
  -> Policy Check
  -> Review if high risk
  -> Sandbox Execution
  -> Audit Log
```

必须支持或预留：

- Sandbox
- Permission
- Review
- Execution limit
- Dependency isolation
- Audit log

高风险行为默认需要 Review：

- 删除文件
- 覆盖文件
- 网络请求
- 数据库写入
- 系统命令
- 读取敏感路径
- 使用未声明依赖

## 11. MCP 和框架 Adapter

MCP 是集成层，不是核心架构。

推荐 MCP 能力：

```text
evotool.search_tools
evotool.get_tool
evotool.save_tool
evotool.execute_tool
evotool.record_execution
evotool.review_tool
evotool.run_benchmark
evotool.search_libraries
evotool.get_dependency_graph
```

Adapter 规则：

- MCP 请求/响应结构不能进入 `domain`。
- Eino/OpenHands 类型不能进入核心包。
- Adapter 只做转换和装配，不写业务规则。
- EvoTool 核心必须能在没有 MCP、Eino、OpenHands 的情况下运行。

## 12. Benchmark 框架

EvoTool 必须用 Benchmark 证明价值。

对比组：

```text
Baseline：Agent 每次重新生成工具。
EvoTool：Agent 先搜索 Tool/Library Memory，找不到才生成。
```

指标：

- Tool generation count
- Duplicate generation count
- Tool reuse count
- Library reuse count
- Average latency
- Average token usage
- Success rate
- Failure rate
- Safety policy blocks
- Dependency-related failures

Benchmark 输出要能被 README、文档和论文式报告引用。

## 13. MVP 边界

第一版 MVP 不要试图一次做完完整能力生态。

MVP 必须完成：

1. 领域模型。
2. Tool Manifest。
3. Library Manifest 的基础模型。
4. 文件系统 ToolStore。
5. 文件系统 LibraryStore。
6. 简单 Registry metadata。
7. 简单关键词检索。
8. ToolMemoryService。
9. ExecutionResult 记录。
10. 基础安全策略模型。
11. 一个本地可信 Demo Executor。
12. 一个实际案例。

MVP 暂时不做：

- 自动修复。
- 自动抽取公共库。
- 分布式沙箱。
- 复杂向量数据库。
- 完整 Eino/OpenHands 合并级 Adapter。
- 多 Agent 协作。

第二阶段再做：

- Embedding 检索。
- Hybrid Retriever。
- Dependency Graph。
- Code Similarity Detection。
- Shared Library 候选抽取。
- Docker Sandbox。
- MCP Server。
- Eino/OpenHands Adapter。

## 14. 最终形态

EvoTool 最终应该像一个 Agent 能力生态管理层：

```text
Agent
  -> EvoTool API
      -> RetrievalService
      -> LearningService
      -> ValidationService
      -> PolicyService
      -> ExecutionService
      -> LifecycleService
      -> BenchmarkService
          -> Tool Memory
          -> Library Memory
          -> Registry
              -> Metadata
              -> Embeddings
              -> Dependency Graph
              -> Execution History
              -> Security Policies
              -> Benchmark Records
```

它要解决的不是“工具放在哪里”，而是：

> Agent 生成出来的能力，如何成为可发现、可复用、可验证、可维护、可演进的长期资产。
