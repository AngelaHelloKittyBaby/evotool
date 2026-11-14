# EvoTool

[English](README.md) | **简体中文**

一个让 Agent 学会积累、复用和演进能力的能力记忆层。

EvoTool 帮助 AI Agent 把一次性的生成脚本变成经过验证、可检索、可复用、可演进的能力。面对相似任务时，Agent 不必反复重新生成同样的辅助代码，而是可以积累一份能力记忆，在未来工作中复用已验证的 Tool 和共享 Library。

关于设计思路、当前进度和仍未解决的取舍，请阅读[项目设计长文](docs/blog-capability-memory.md)。

## 快速开始

外部 Agent 提交 Python Tool 包，经沙箱验证后进入能力记忆。详见[学习流程与一键验收测试](docs/learning.md)。这条路径需要可用的 Docker Linux 引擎和准备好的 Python 镜像，本身不内嵌 LLM。

从一个提交的包学习一个 Tool：

```bash
go run ./cmd/evotool learn --package examples/csv-analysis/generated.json --root .evotool
go run ./cmd/evotool search tools --query "csv numeric summary" --root .evotool
go run ./cmd/evotool execute --id csv_numeric_summary --input examples/csv-analysis/input.json --root .evotool
```

这会生成如下本地结构：

```text
.evotool/
  tools/csv_numeric_summary/versions/v1/main.py
  registry/metadata/tools/csv_numeric_summary.json
  registry/execution_history/csv_numeric_summary.jsonl
  registry/audit/events.jsonl
```

`search` 命令执行本地检索流水线：关键词召回、元数据过滤、能力排序，最后完成选择。

Library 检索与 Tool 检索刻意分离：Tool 是顶层可调用能力，而 Library 是可复用的实现依赖。

```bash
go run ./cmd/evotool search libraries --query "csv statistics" --runtime python --root .evotool
```

查看 Tool -> Library 依赖图：

```bash
go run ./cmd/evotool deps graph --id csv_numeric_summary --root .evotool
go run ./cmd/evotool deps check --id csv_numeric_summary --root .evotool
```

用 Benchmark 度量复用效果：

```bash
go run ./cmd/evotool benchmark run --suite examples/benchmark/csv-reporting-suite.json --root .evotool
```

预览长期未使用的 Tool：

```bash
go run ./cmd/evotool lifecycle scan --root .evotool
```

默认策略会把 90 天未使用的 Tool 标记为 `stale` 候选，把 180 天未使用的 Tool 标记为归档候选。该命令是只读的：只报告候选，不修改也不删除任何文件。

## 当前状态

EvoTool 处于早期设计与 MVP 阶段。

执行策略服务会检查生命周期、验证可信度、版本是否存在，以及显式声明的宿主机资源上限。`ExecutionService` 负责编排依赖检查、策略检查、审计事件、执行器调用和执行历史。外部提交流程目前可以验证无依赖的 Python Tool，并通过 Docker 沙箱执行它们。Benchmark Runner 会以 baseline 和 EvoTool 两种模式重放任务套件，报告复用实际节省了什么。涉及特权的提交会被拒绝；模型生成、可执行的共享库依赖、以及审查批准界面尚未实现。详见[学习流程](docs/learning.md)、[Benchmark](docs/benchmark.md) 和[安全模型](docs/security.md)。

## 核心思路

```text
任务
  -> 检索已学习的 Tool
  -> 匹配则复用
  -> 缺失则生成
  -> 识别可复用 Library
  -> 在沙箱中验证
  -> 应用安全策略与审查
  -> 保存 Tool 元数据与代码
  -> 更新 Registry 与依赖图
  -> 记录执行结果
  -> 随时间演进、重构或废弃能力
```

## 能力记忆架构

EvoTool 不只是一个 Tool Cache。它被设计成三层结构的 Agent Capability Memory：

```text
Agent Capability Memory

  tools/        # 用户可调用的能力
  libraries/    # 共享的实现构建块
  registry/     # 元数据、embedding、版本、依赖、历史与策略
```

### Tools

Tool 是 Agent 可以直接选择和执行的能力：

```text
pdf_to_excel
csv_analyzer
sentiment_report_generator
webpage_to_markdown
```

每个 Tool 都应该有语义化名称、manifest、测试、文档、执行历史和生命周期状态。

## Tool 生命周期与保留策略

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

Library 是多个 Tool 共享的底层能力：

```text
pdf_parser
excel_writer
text_processor
data_cleaner
```

Tool 可以依赖 Library，而不是一遍又一遍复制同样的辅助代码。

### Registry

Registry 是能力目录。它保存元数据、embedding、分类、版本、执行历史、安全策略、Benchmark 记录，以及 Tool 与 Library 之间的依赖图。

## 为什么不直接用 Skill？

Skill 通常是开发者定义的能力：

```yaml
Skill:
  creator: 开发者
  flow: 开发者创建能力 -> Agent 使用能力
```

EvoTool 关注的是 Agent 在任务中学习生成出来的 Tool：

```yaml
Learned Tool:
  creator: Agent
  flow: Agent 生成能力 -> EvoTool 验证并保存 -> 未来的 Agent 复用它
```

EvoTool 不只是一个 Skill Registry 或 Tool Registry。它管理生成能力的整个生命周期：命名、验证、安全审查、检索、依赖复用、执行历史、Benchmark、版本管理，以及未来的演进。

## Tool 命名很重要

Agent 不应该把工具保存成 `tool1.py`、`test.py` 或 `helper.py`。

一个学到的 Tool 应该有语义化的 manifest：

```json
{
  "name": "pdf_table_extractor",
  "description": "从 PDF 文档中提取表格并导出为 Excel 格式",
  "category": "document_processing",
  "input": {
    "file": "pdf"
  },
  "output": {
    "file": "xlsx"
  }
}
```

好的命名能提升检索效果，因为未来的 Agent 是通过意图、描述、元数据、embedding 和执行历史来查找工具的。

## 检索策略

EvoTool 不应只依赖工具名，也不应把所有工具都交给 LLM。检索是分阶段的：

```text
用户任务
  -> 意图提取
  -> 语义检索，例如 20,000 个工具召回 50 个候选
  -> 元数据过滤，例如 50 个候选筛到 10 个
  -> 能力排序，例如 10 个候选排序到 3 个
  -> LLM 或规则选择，最终 1 个工具
```

Tool 的 embedding 应该由以下内容组合而成：

```text
name + description + tags + input/output schema + examples + capabilities
```

排序应该综合考虑：

- 语义相似度
- 成功率
- 可靠性
- 平均延迟
- 使用历史
- 环境匹配
- 依赖健康度

这意味着 EvoTool 不仅要判断“哪个工具更像”，还要判断“哪个工具更可靠”。

执行结果如今会反馈到排序中。执行器真正运行之后，`ExecutionService` 会把结果合并进已保存的 `UsageStats` 和两份元数据，因此反复成功会抬高工具评分，反复失败会降低评分。被策略或依赖检查拦截的调用会被记入执行历史和审计日志，但不计入使用经验。

## EvoTool 想提供什么

- 面向生成工具的 Tool Memory
- 面向可复用实现构建块的 Shared Library Memory
- 保存元数据、embedding、版本、依赖图、策略和执行历史的 Registry
- 度量复用替代了多少生成的 Benchmark
- 安全机制，包括沙箱、权限、审查、执行限制和依赖隔离
- 面向 Agent 生态的 MCP 集成
- 先做 Eino Adapter，再做 OpenHands Adapter
- 完整文档和可复现示例
- 真实任务案例，例如 PDF 转 Excel、网页转 Markdown、CSV 分析和报告生成

## 安全优先

生成的工具默认不可信。EvoTool 围绕明确的安全边界设计：

- 沙箱执行
- 权限策略
- 高风险工具的人工或策略审查
- 执行时间、内存、输出、文件和网络限制
- 依赖隔离
- 覆盖生成、验证、执行、失败和版本变更的审计日志

一个生成的工具必须通过验证和策略检查之后，才能成为可自动复用的能力。

## Benchmark

EvoTool 用可复现的 Benchmark 证明自己的价值，对比两组：

```text
Baseline：Agent 对重复出现的任务每次都重新生成工具。
EvoTool：Agent 先检索已学习的 Tool 和 Library，只有找不到时才生成。
```

```bash
go run ./cmd/evotool benchmark run --suite examples/benchmark/csv-reporting-suite.json --root .evotool
```

在自带的三个重复 CSV 报表任务套件上，当 `csv_numeric_summary` 已保存且依赖健康时：

```text
metric                         baseline    evotool      delta
tools generated                       3          0         -3
tools reused                          0          3         +3
duplicate generations                 2          0         -2
total tokens                       3600          0      -3600
```

在空能力记忆下，同一套件所有 delta 都是 `+0`。这个零正是重点：Runner 不会硬编码一个“节省”的结论。

EvoTool 本身不内嵌模型，所以 token 数是套件中每个任务自己声明的生成成本。Runner 度量的是复用替代了多少次生成，而不是模型生成质量，也不会执行生成的工具。这条边界记录在 [`docs/benchmark.md`](docs/benchmark.md) 中。

关键指标：

- 工具生成次数
- 重复生成次数
- 工具复用次数
- Library 复用次数
- 平均决策延迟
- 平均 token 使用量
- 成功率与失败率（执行接入 harness 之后）
- 安全策略拦截次数
- 依赖相关失败次数

## Roadmap

已实现：

- 核心领域模型、Ports 和 Services
- 带命名校验的 Tool 与 Library manifest
- 文件系统 ToolStore、LibraryStore 和 RegistryStore
- 分阶段检索：关键词召回、元数据过滤、能力排序
- 执行结果通过 `UsageRecorder` 反馈进排序
- Registry、依赖图和依赖健康检查
- 通过 Docker 沙箱进行验证并记录执行历史
- 特权权限 fail-closed 的安全策略模型
- 只读的 stale 与归档候选生命周期扫描
- 对比 baseline 与 EvoTool 复用效果的 Benchmark Runner

尚未实现：

- 模型生成：`ToolGenerator` 没有实现，工具由外部 Agent 提交
- 可执行的共享库依赖：`LibraryRefs` 会被记录和健康检查，但在验证阶段被拒绝
- 高风险工具的审查批准界面
- 生命周期归档、恢复和清理命令
- MCP Server Adapter
- 先做 Eino Adapter，再做 OpenHands Adapter
- 语义/向量检索，替换当前的关键词检索器
- 代码相似度检测和共享 Library 抽取候选
- 失败分析和自动修复

## License

MIT License.
