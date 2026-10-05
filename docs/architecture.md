# EvoTool 架构设计

EvoTool 的核心定位是 Agent Capability Memory：一个让 Agent 学会积累、复用和演进能力的能力记忆层。

## 三层能力框架

```text
Agent Capability Memory

  tools/        # 用户或 Agent 可以直接调用的顶层能力
  libraries/    # 多个 Tool 共享的底层能力和公共代码
  registry/     # 元数据、向量索引、依赖关系、版本、历史和安全策略
```

## Tool 层

Tool 是 Agent 可以直接选择和执行的能力，例如：

```text
pdf_to_excel
csv_analyzer
excel_formatter
sentiment_report_generator
```

每个 Tool 必须拥有独立目录、manifest、测试、README、版本和执行历史。

## Library 层

Library 是多个 Tool 共享的底层能力，例如：

```text
pdf_parser
excel_writer
text_processor
data_cleaner
```

Library 默认不直接暴露给用户任务调用。它主要用于减少重复代码，并让能力系统像软件生态一样管理依赖和复用。

## Registry 层

Registry 是能力目录和检索中心，负责保存：

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

## 多级检索

EvoTool 不能只按名字检索工具。推荐检索链路：

```text
用户任务
  -> 能力分类召回
  -> Embedding 语义召回
  -> Metadata 过滤
  -> 安全策略过滤
  -> 依赖状态过滤
  -> 执行历史排序
  -> LLM 或规则最终选择
```

## 依赖和复用

Tool 可以依赖 Library。Library 变更后，依赖它的 Tool 必须重新验证。

未来可以加入 Code Similarity Detection，发现多个工具中重复的实现，并提出共享 Library 抽取建议。

自动抽取共享库属于第二阶段能力。MVP 阶段只需要记录重复和相似度，不要直接自动重构生产工具。

## 安全边界

生成工具默认不可信。执行前必须经过：

```text
Validation
  -> Policy Check
  -> Review if high risk
  -> Sandbox Execution
  -> Audit Log
```

任何涉及删除、覆盖、网络请求、数据库写入、系统命令的工具，默认需要审查。
