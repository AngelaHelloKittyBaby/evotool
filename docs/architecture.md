# EvoTool 架构设计

EvoTool 的核心定位是 **Agent Capability Memory**：一个让 Agent 学会积累、复用和演进能力的能力记忆层。

它不是简单的 Tool Cache，也不是普通 Skill Registry。EvoTool 要解决的是：Agent 在任务中生成出来的工具，如何被命名、验证、安全审查、持久化、检索、复用、依赖管理、版本演进，并最终成为长期可维护的能力资产。

## 1. 总体架构

```text
External Agent Runtime
  -> EvoTool API / Facade
      -> Application Services
          -> Ports
              -> Adapters
```

核心分层：

- External Agent Runtime：Custom Agent、ReAct、Eino、OpenHands、MCP Client、CLI。
- EvoTool API / Facade：`LearnTool`、`SearchTools`、`ExecuteTool`、`RecordExecution`。
- Application Services：检索、学习、验证、执行、安全、生命周期、Benchmark。
- Ports：存储、检索、生成、执行、沙箱、安全、审计等接口。
- Adapters：文件系统、SQLite、向量数据库、Docker、MCP、Eino、OpenHands、LLM SDK。

## 2. 能力记忆框架

```text
Agent Capability Memory

  tools/        # Agent 可直接调用的顶层能力
  libraries/    # 多个 Tool 共享的底层能力
  registry/     # 元数据、embedding、依赖、版本、历史、安全策略
```

Tool 是可调用能力，例如 `pdf_to_excel`。Library 是共享底层能力，例如 `pdf_parser`、`excel_writer`。Registry 是检索和治理中心，保存 metadata、embedding、dependency graph、execution history、policy、benchmark records。

## 3. 分阶段工具检索

工具越来越多以后，检索必须分阶段，不能把全部工具交给 LLM。

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

规模示例：

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

Embedding 不应只基于工具名，而应基于：

```text
name + description + tags + input/output schema + examples + capabilities
```

最终排序不只看语义相似度，还要综合：

```text
Score = semantic_similarity
      + success_rate
      + reliability
      + latency
      + usage_history
      + environment_match
```

目标是让 Agent 不仅知道“哪个工具像”，还知道“哪个工具更可靠”。

## 4. 生成前先复用 Library

第一版不要做自动代码重构。更安全、更容易实现的路径是：生成 Tool 前先搜索已有 Library。

```text
Generate Tool
  -> Dependency Analysis
  -> Search Library
  -> If found: reuse
  -> If missing: create candidate
  -> Validate Tool with resolved dependencies
  -> Record Tool -> Library dependency
```

第二阶段再做：

```text
Existing Tools
  -> Code Similarity Detection
  -> Library Extraction candidate
  -> Agent-assisted refactor
```

## 4.1 当前 MVP 的 Library 检索实现

当前 MVP 已经把“生成前搜索 Library”落到本地文件系统适配器：

```text
registry/metadata/libraries/*.json
              |
              v
internal/adapter/retriever/local/LibraryRetriever
              |
              v
CapabilityMemoryService.SearchLibraries
              |
              v
Agent 选择兼容的 Library 作为新 Tool 的依赖
```

当前实现会根据 Library 的名称、描述、分类、Runtime、Tags、Exports 和依赖信息进行本地关键词召回，并按照 Runtime、网络权限、生命周期和信任等级进行兼容性过滤。

它暂时不做自动代码抽取或自动重构。后续可以在不改变 `ports.LibraryRetriever` 契约的前提下，替换成 embedding、向量数据库或混合检索实现。

## 4.2 Tool 到 Library 的依赖图

当 Agent 生成 Tool 前检索并选择了可复用 Library，EvoTool 必须把这个选择写成显式依赖，而不是只保存在临时上下文里。当前 MVP 的落点是：

```text
Tool Manifest.LibraryRefs
Tool Dependencies
registry/dependency_graph/<tool-id>.json
```

示例关系：

```text
pdf_to_excel --uses--> pdf_parser
```

这条依赖图后续会用于：

- 执行前检查依赖是否存在和健康。
- Library 升级后找出受影响的 Tool。
- 重新验证依赖该 Library 的 Tool。
- Benchmark 统计 Library 复用次数。
- 生命周期管理中判断是否需要 quarantine 或 deprecate。

当前 CLI 可以通过 `evotool demo link-tool-library` 写入示例依赖图，通过 `evotool deps graph --id pdf_to_excel` 查询图结构，并通过 `evotool deps check --id pdf_to_excel` 执行依赖健康检查。

依赖健康检查是执行前的最小 gate：它会读取依赖图，确认必需的 Library 是否存在，并检查版本约束是否匹配。当前状态会返回 `healthy`、`degraded` 或 `broken`，后续 ExecutionService 应在真正执行 Tool 之前调用这一步。

## 5. 完整闭环

```text
                    User Task
                        |
                        v
                 Agent Planner
                        |
                        v
              Capability Retriever
                        |
          +-------------+-------------+
          |                           |
    Existing Tool              Capability Gap
          |                           |
          |                           v
          |                    Tool Generator
          |                           |
          |                    Dependency Search
          |                           |
          |                    +------+------+
          |                    |             |
          |                 Reuse        Generate
          |                    |             |
          |                    +------+------+
          |                           v
          |                      Sandbox
          |                           v
          |                       Validate
          |                           v
          +-----------------------> Tool
                                      |
                                      v
                                  Execute
                                      |
                              +-------+-------+
                              |               |
                           Success         Failure
                              |               |
                              v               v
                         Tool Memory     Evolution later
                              |               |
                              +-------+-------+
                                      v
                               Capability Registry
```

第一版先把 Tool Registry、Library Registry、Semantic Retrieval、Dependency Graph、Eino Adapter 做扎实。自动代码重构和自动修复放到第二阶段。

## 6. 安全边界

生成工具默认不可信。执行前必须经过：

```text
Manifest Check
  -> Dependency Check
  -> Static Risk Check
  -> Policy Check
  -> Review if high risk
  -> Sandbox Execution
  -> Audit Log
```

高风险行为默认需要 Review，例如删除文件、覆盖文件、网络请求、数据库写入、系统命令、读取敏感路径、使用未声明依赖。
