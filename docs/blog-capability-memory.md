# 让 Agent 记住自己造过的轮子：EvoTool 的设计与当前进度

> 写于 2026 年 11 月，对应提交 `7bf8d5c`。这是一篇阶段性记录，不是完成品宣言。

## 一、问题从哪里来

如果你让一个 Agent 连续做十次数据报表任务，它会十次都重新写一遍 `csv` 求和脚本。每次生成的代码大同小异，每次都要花掉一遍生成 token，每次的错误也大同小异——忘记处理空表、负数、带引号的字段。

这不是模型的失忆，是架构的失忆。模型本身无状态，上下文窗口关掉就什么都没了。业界目前的解法大致两类：

- **Skill**：开发者预先写好能力模块，Agent 按需加载。这是人工维护的能力库。
- **Tool Registry**：把工具元数据登记起来，让 Agent 能查找。这通常只解决"知道有哪些工具"，不解决"这些工具怎么来的、可信不可信、还能不能用"。

EvoTool 想走的是一条不同的路：**能力由 Agent 在任务中产生，经过验证和审查后进入记忆，未来被检索和复用，并且会随时间和依赖变化而演进。**

一句话概括定位：

> A capability memory layer that allows AI agents to discover, reuse, and evolve generated tools.

中文：一个让 Agent 学会积累、复用和演进能力的能力记忆层。

## 二、为什么不是 Skill

这是这个项目必须回答的第一个问题，因为很容易被归类成"又一个 Skill Registry"。

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

差别不在存储格式，在**责任链**。开发者写的 Skill，出了问题有人负责；Agent 现场生成的工具，没人负责。所以 EvoTool 的重点不在"存下来"，而在存下来之前和之后的那一整套：

```text
生成之前：能力缺口判断、先搜 Library 再生成
生成之后：命名检查、Schema 检查、沙箱跑测试、安全策略、审查
存下来后：多级检索、依赖健康检查、执行统计反馈排序
长期运行：失败累积、隔离、降级、版本演进、归档
```

如果只做第一半，它就是个 Tool Cache。EvoTool 的价值主张在后半段。

## 三、架构：五层，依赖只能向内

```text
External Agent Runtime
  -> EvoTool API / Facade
      -> Application Services
          -> Ports
              -> Adapters
```

- **External Agent Runtime**：自定义 ReAct Agent、Eino、OpenHands、MCP Client、CLI。
- **API / Facade**：`LearnTool`、`SearchTools`、`ExecuteTool`、`RecordExecution`，对外稳定，不暴露实现。
- **Application Services**：检索、学习、验证、执行、策略、生命周期、Benchmark。
- **Ports**：接口层。存储、检索、生成、执行、沙箱、embedder、审计……
- **Adapters**：文件系统、Docker、关键词检索、MCP、Eino、OpenHands。

依赖方向单向向内。核心代码不允许 import Eino、OpenHands、LangChain、OpenAI SDK、MCP SDK、SQLite driver、向量数据库 client、Docker client。这些东西只能出现在 `internal/adapter/` 下。

目前这个约束是守住的：`go list -deps` 里核心包只有标准库和自身。

能力记忆内部再分三层：

```text
tools/        # Agent 可直接调用的顶层能力
libraries/    # 多个 Tool 共享的底层实现
registry/     # 元数据、索引、依赖图、版本、历史、策略
```

Tool 和 Library 的区分是个关键设计。把两个工具里重复的 `extract_table()` 抽成共享 Library，比把两个工具整体复用好一个数量级——但如果一开始就不建模 Library，后面想补就得重构全部存储。

## 四、检索：不能把一万个工具塞给 LLM

```text
User Task
  -> Intent Extraction
  -> Semantic Retrieval      20,000 -> 50
  -> Metadata Filter             50 -> 10
  -> Capability Rank             10 -> 3
  -> LLM / rule Selection         3 -> 1
```

每一级都有明确的收敛职责。当前实现（`internal/adapter/retriever/local/`）：

- **Semantic Retrieval** 是本地关键词打分，对 `name`、`description`、`tags`、`capabilities` 做匹配。**这不是 embedding 检索**，是个占位实现，但阶段划分是真的，替换掉它不影响上下游。
- **Metadata Filter** 按 category、runtime、input/output 类型、生命周期状态、trust level、网络权限过滤。
- **Capability Rank** 用加权公式。

排序公式值得单独说，因为它是 EvoTool 和"相似度排序器"的分界线：

```go
final := 0.55*semantic + 0.15*success + 0.10*reliability +
         0.10*latency + 0.05*usage + 0.05*environment
```

语义相似度只占 55%，剩下 45% 是**它过去干得怎么样**。考虑这个场景：

```text
Tool A: similarity 0.91, success_rate 99%, latency 1.2s, usage 1200
Tool B: similarity 0.95, success_rate 72%, latency 4.8s, usage 15
```

纯语义排序会选 B。EvoTool 应该选 A。这要求系统不仅知道"哪个工具像"，还要知道"哪个工具更可靠"。

### 这里曾经有个真实的洞

写完 ranker 之后有段时间，`ToolUsageStats` 这个结构体**从来没有被写入过**。执行完一个工具，历史记了，审计记了，但统计没动。于是那个公式里 45% 的权重全部塌缩成常量：

```text
实际生效的排序 = 0.55 * semantic + 0.125
```

也就是说，"我们知道哪个工具更可靠"这个核心卖点在代码层面是死代码。

修复方式是在 `ports` 加一个小接口：

```go
// UsageRecorder 把一次真实执行折叠进 Tool 的累计使用统计。
type UsageRecorder interface {
    RecordUsage(ctx context.Context, result domain.ExecutionResult) error
}
```

`ExecutionService` 只在**执行器真的跑过**之后调用它。被策略拦下的、被依赖检查拦下的，记历史、记审计，但不计入使用统计。这一点是刻意的：一个需要人工审查的工具，如果每次因为"需要审查"就被记一次失败，它会被永久压到排名底部，永远竞争不过从没被调用过的工具。区分"工具坏了"和"没让它跑"很重要。

对应的测试是端到端的：造两个 tool，一个 5 连胜，一个 5 连败，语义相似度相同，断言前者排前面。

### 还有一个更安静的 bug

`RetrievalFilters.AllowedStatuses` 为空时，原来的行为是**不过滤任何状态**。这意味着 `archived`、`quarantined`、`deleted` 的工具照样会出现在默认搜索结果里——直接违反 AGENTS.md 里"默认检索过滤 archived、quarantined 和 deleted"那一条。

修复是加 `domain.DefaultRetrievableStatuses()`，返回 `approved`、`active`、`stale`，所有调用点显式传入。这类 bug 不会报错，只会让垃圾数据慢慢浮上来，属于最难在代码审查里看见的那种。

## 五、安全：默认不可信

Agent 生成的工具默认是敌对的。这不是偏执——考虑这个链条：

```text
Agent 生成了 delete_database.py
  -> 工具被保存
  -> 下次用户说"清理数据"
  -> Agent 匹配到这个工具
  -> 数据没了
```

所以 `PolicyService` 是 fail-closed 的。允许执行的基础条件全部要满足：

- 生命周期是 `approved` / `active` / `stale`
- trust level 是 `validated` / `reviewed` / `trusted`
- 显式指定了当前版本
- 三项资源限制都是正数，且不超过宿主上限

未声明网络权限时**按禁止网络处理**。声明网络、宿主路径、系统命令、环境变量访问 → `Allowed=false`、`RequiresReview=true`。注意这两个字段的分工：需要审查不代表其他拒绝理由可以被忽略，两者分别记录。

一个刻意的保守决定：即使 trust label 已经是 `reviewed` 或 `trusted`，特权声明仍然要求审查。原因是当前模型还没有把"审查证明"绑定到具体版本和具体权限集合上。光靠改一个字段就能放行，那就等于没有审查。这个接口留着，等真正的审查工作流接进来。

策略判断只看元数据，**不执行代码，也不证明代码安全**。真正的边界在沙箱。

### 沙箱

Docker，默认配置：

```text
禁网 / 无宿主目录挂载 / 非 root / 只读根文件系统
去除 Linux capabilities / 限制 CPU、进程数、内存、输出、运行时长
源文件通过 stdin 传进容器内临时目录 / 超时后移除容器
```

镜像是宿主可信配置，执行不会隐式拉取。

## 六、验证：跑起来才算数

`learn` 的流程不是"存下来"，是"跑通了才存下来"：

```text
提交 GeneratedTool JSON
  -> Schema 检查
  -> 命名检查（必须是语义化 snake_case）
  -> 策略预检
  -> 文件路径安全检查（拒绝绝对路径、..、反斜杠、重复路径）
  -> 包大小限制（512 KiB / 100 文件）
  -> 沙箱里跑每一个 test
  -> 沙箱里跑每一个 example，比对输出
  -> 全绿 -> 存为 approved / validated / v1
```

这里有个容易忽略的顺序问题：**依赖图先注册，工具后保存**。如果反过来，中途失败会留下一个能被检索到、但依赖图不存在的工具，后续执行时的依赖检查会直接报错。宁可留下一个空图（重试时会被覆盖），也不要留下一个可执行但依赖不清的工具。

## 七、Benchmark：用数字说话，但只说测得到的

EvoTool 的价值必须能被度量，否则就是营销。当前实现重放同一套任务两次：

```text
baseline：每个任务都生成工具
evotool ：先查能力记忆，命中且依赖健康就复用，否则生成
```

`examples/benchmark/csv-reporting-suite.json` 是三个月度/季度/小时的报表任务，输入列不同、需要的能力相同，每个任务假定生成成本 1200 token。

**空能力记忆：**

```text
metric                         baseline    evotool      delta
tools generated                       3          3         +0
tools reused                          0          0         +0
duplicate generations                 2          2         +0
total tokens                       3600       3600         +0
```

**已存有 `csv_numeric_summary` 且依赖健康：**

```text
metric                         baseline    evotool      delta
tools generated                       3          0         -3
tools reused                          0          3         +3
duplicate generations                 2          0         -2
total tokens                       3600          0      -3600
```

第一组数字比第二组重要。空记忆时必须报告 0——如果有哪个环节在硬编码收益，这一条会立刻暴露。`TestBenchmarkWithEmptyMemoryShowsNoSavings` 就是钉这个的。

依赖不健康时同样不复用：检索命中但 Library 缺失，计入 `dependency failures` 并退回生成，`GenerationSaved()` 返回 0。检索到不等于能跑，benchmark 不会替一个坏依赖记功。

### 这个数字的边界

EvoTool 不内置大模型。任务的 `Package`（本该生成的产物）和 `TokenUsage`（本该花掉的钱）都由 suite 提供。

所以这 3600 token 是**本该花掉的生成成本**，不是实测账单。它测的是"复用替代了多少次生成"，不是模型生成质量。

同样诚实地说，`SuccessRate` 和 `FailureRate` 现在保持为 0。这个 runner 重放的是复用决策，它没有真的执行生成的工具，填 1.0 就是一个没人测量过的断言。等真正的 `ToolGenerator` 接进来，这两个字段才有意义。

这条边界写在三个地方：代码注释、CLI 输出、[docs/benchmark.md](benchmark.md)。一个能被任意解读的数字比没有数字更糟。

## 八、删掉一个假的 demo

项目里曾经有一个 `demo save-tool` 命令，往用户的 `.evotool` 目录写一个假的 `pdf_to_excel` 工具。它没有解析 PDF 的能力，只能推导出一个 xlsx 路径。

它存在了很久，因为"让新用户三秒钟看到一个能跑的东西"看起来很有价值。

问题在于：这些落盘的文件，和一个 Agent 真正学习并验证过的能力，**在磁盘上完全无法区分**。跑完 demo 的人会以为自己有一个可用的 PDF 工具；如果在 demo 之后继续跑 `search`、`deps check`、`benchmark`，这些命令会认真地分析一个虚构的能力，然后给出看起来很专业的结论。

把夹具塞进二进制里当产品卖，这是能骗过测试、骗过演示、但骗不过第一个真实用户的错误。

删掉之后，测试改成通过 store API 播种数据，另外加了一条 `TestRunRemovedDemoCommand`，防止哪天又有人觉得 demo 很有用。

## 九、当前真实状态

截至 `7bf8d5c`，19 个提交，2026-10-05 至 2026-11-14。约 7000 行 Go（其中非测试代码 4656 行），69 个测试，总覆盖率 65.8%。

**能用的：**

- Tool / Library Manifest 与命名校验
- 文件系统存储（ToolStore / LibraryStore / RegistryStore）
- 四级检索管道：关键词召回 → 元数据过滤 → 能力排序 → 选择
- 执行结果反馈进排序（`UsageRecorder`）
- 依赖图与依赖健康检查
- Docker 沙箱验证与执行历史
- fail-closed 安全策略
- 只读生命周期扫描（stale / archive 候选）
- baseline vs evotool 的 benchmark

**不能用的：**

- **`ToolGenerator` 没有实现。** 工具由外部 Agent 生成后提交，EvoTool 不调用模型。这是当前最大的缺口——它让整个学习流程少了一半。
- **可执行的 Library 依赖。** `LibraryRefs` 会被记录、会被依赖检查，但 `validation_service.go` 直接拒绝带 Library 引用的提交。Library 能搜到，但用不上。
- **审查工作流。** 缺口在"审查证明绑定到版本和权限"，不是 UI。
- **生命周期只扫不写。** `archive` / `restore` / `purge` 三项都还是设计目标。
- **MCP、Eino、OpenHands Adapter** 都没开始。
- **语义检索是关键词。** 没有 embedder 实现。

## 十、下一步的顺序

按依赖关系排，不是按有趣程度排：

1. **可执行的 Library 依赖** —— `LibraryStore` 缺 `List`，`validation_service` 的拒绝分支要拆开。做完它，Tool 才真的能共享底层实现，"Library 复用"才不是一句空话。
2. **真正的 `ToolGenerator`** —— 补上它，benchmark 才能报告真实执行成功率和真实 token 账单，而不是靠 suite 声明。
3. **审查工作流** —— 让特权工具有一条能真正走通的路径，而不是永远卡在 `RequiresReview`。
4. **embedder + 向量检索** —— 替换关键词打分。在有稳定检索管道的前提下，这是纯粹的质量提升，风险可控。

再往后才是 MCP / Eino Adapter、代码相似度检测、共享 Library 自动抽取、故障分析与自动修复。这些都是 AGENTS.md 明确划入第二阶段的能力，现在做等于在没有地基的上盖楼。

## 十一、几个还在的取舍

写到这里，有几件事我仍然不确定，记下来免得以后假装当时想清楚了：

**检索排序的权重是拍的。** `0.55 / 0.15 / 0.10 / 0.10 / 0.05 / 0.05` 没有数据支撑，是凭直觉分配。它需要一个能标注"正确工具是什么"的数据集来调参，而这个数据集目前没有。

**`stale` 状态同时参与检索又不参与信任提升。** 一个 90 天没用的工具仍然会被检索到、仍然可以被执行，只是排序会被 usage 项拉低。这可能过于宽松，但也可能是对的——季节性任务本来就会长期闲置。没有真实用量数据，现在改只是换一种猜测。

**"重复生成"的判定是按包名。** 两个 Agent 用不同名字写了功能完全相同的工具，benchmark 不会算作重复。这低估了重复率，但在没有相似度检测之前，按包名是唯一不会误报的办法。

**`AverageLatency` 只统计决策耗时，不包含执行。** 它适合做回归对比，不适合对外说"任务快了多少"。

这些不是待办事项，是已知的、被有意保留的不确定性。写下来是为了让下一个接手的人知道哪里是猜的。

---

## 附：快速跑一遍

```bash
# 学习一个工具（需要 Docker Linux 引擎 + python:3.13-alpine 镜像）
go run ./cmd/evotool learn --package examples/csv-analysis/generated.json --root .evotool

# 检索、执行、记录
go run ./cmd/evotool search tools --query "csv numeric summary" --root .evotool
go run ./cmd/evotool execute --id csv_numeric_summary --input examples/csv-analysis/input.json --root .evotool

# 依赖与生命周期
go run ./cmd/evotool deps check --id csv_numeric_summary --root .evotool
go run ./cmd/evotool lifecycle scan --root .evotool

# 度量复用到底省了多少
go run ./cmd/evotool benchmark run --suite examples/benchmark/csv-reporting-suite.json --root .evotool
```

完整验证脚本（含反向用例：失败测试、网络权限申请、重复提交）见 [learning.md](learning.md)。

架构细节见 [architecture.md](architecture.md)，安全模型见 [security.md](security.md)，benchmark 方法见 [benchmark.md](benchmark.md)，生命周期见 [lifecycle.md](lifecycle.md)。
