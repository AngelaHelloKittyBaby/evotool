# 外部 Agent 提交、验证和复用工具

EvoTool 不捆绑模型。外部 Agent 可以提交 `domain.GeneratedTool` JSON，也可以配置模型 API 后使用 `generate`，见 [模型生成与实测 Benchmark](generation.md)。两条路径都经过验证、隔离运行、保存和检索。仓库中的 `examples/csv-analysis/generated.json` 是固定的示例提交包，不是模型现场生成的结果。

## 一键验收（PowerShell）

前提：Go 1.25+、运行中的 Docker Desktop Linux 引擎。准备官方 Python 镜像后，在仓库根目录执行：

```powershell
docker pull python:3.13-alpine
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-learning.ps1
```

成功标志：最后出现 `PASS: validation, persistence, execution, search, reuse, history, failure rejection, permission rejection, overwrite protection`，退出码为 0。

脚本创建独立的 `tmp/learning-<随机ID>/memory`，不改动已有 `.evotool`。它实际验证：

1. 学习 CSV 数值统计工具，运行三个 Python 断言和一个输入输出样例，验证通过后存为 `approved / validated / v1`。
2. 输入 10、20、30，得到 count=3、total=60、average=20。
3. 同一个工具输入 40、60，得到 count=2、total=100、average=50；无需再次提交代码。
4. 搜索选中 `csv_numeric_summary`，并检查两条执行历史、Registry 内的验证证据。
5. 故意提交失败测试、申请网络权限、重复工具，确认全部拒绝。

脚本中的 Python Traceback、`network access requires review` 和 `tool already exists` 是预期的反向验证输出；请以最终 PASS 和退出码判断结果。

## 手动验证

选择一个未使用的 root；已有工具禁止重复提交覆盖。

```powershell
go build -o tmp/evotool.exe ./cmd/evotool
./tmp/evotool.exe search libraries --query 'csv numeric summary' --root tmp/manual-learning
./tmp/evotool.exe learn --package examples/csv-analysis/generated.json --root tmp/manual-learning
./tmp/evotool.exe search tools --query 'csv numeric summary' --root tmp/manual-learning
./tmp/evotool.exe execute --id csv_numeric_summary --input examples/csv-analysis/input.json --root tmp/manual-learning
./tmp/evotool.exe execute --id csv_numeric_summary --input examples/csv-analysis/input-second.json --root tmp/manual-learning
```

learn 的 JSON 中应出现 `validation.Valid: true`。execute 的 JSON 中应出现 `Success: true`；`Output` 是 JSON 字符串，可再次解析。失败命令返回非零退出码。

保存的证据包括：

```text
tools/csv_numeric_summary/versions/v1/main.py
tools/csv_numeric_summary/versions/v1/tests/check_summary.py
registry/metadata/tools/csv_numeric_summary.json
registry/dependency_graph/csv_numeric_summary.json
registry/execution_history/csv_numeric_summary.jsonl
registry/audit/events.jsonl
```

## 提交契约和边界

外部 Agent 应先搜索 Tool，已有能力则直接复用；未命中时先搜索 Library，再生成提交包。以 `examples/csv-analysis/generated.json` 为完整字段模板：Manifest、SourceFiles、Tests、GenerationReason 必填；Readme 可选。测试文件是可以直接运行、用 assert 或异常返回失败的 Python 脚本，不会自动通过 pytest 发现测试函数。每个 Examples 的 Input/Output 都是 JSON 对象字符串，工具从 stdin 接收对象并向 stdout 输出一个 JSON 对象。

当前支持无第三方包依赖的 Python 工具，并允许执行引用已批准的固定版本 Python Library；库的加载、源码摘要绑定和验证方式见 [可执行共享库](library-execution.md)。暂不安装第三方包，不支持传递依赖。

Schema 支持 object、properties、required，以及 string、number、integer、boolean 属性；不支持的 schema 关键字明确拒绝，尚非完整 JSON Schema 验证器。

容器默认禁网、无宿主目录挂载、非 root、只读根文件系统、去除 Linux capabilities，限制 CPU、进程数、内存、输出和运行时长。源文件通过 stdin 传入容器内的临时目录。超时后移除容器。镜像必须预先安装，执行不会隐式拉取镜像；`--image` 属于宿主可信配置，生产环境建议固定镜像 digest。

主机上限为 30 秒、256 MiB 内存、1 MiB 输出；单包源文本上限 512 KiB / 100 个文件，提交 JSON 和执行输入各上限 1 MiB。申请网络、宿主路径、命令或宿主环境变量的包不会自动批准。本版拒绝这类包，尚未提供人工审批后放行的入口。

测试通过只代表提交的测试和样例通过，不证明工具适用于所有输入。当前文件存储没有跨文件事务或并发版本竞争保护，应按单写者使用。

## 使用统计与检索排序

`ExecutionService` 在 Executor 实际运行后，把结果折叠进 Tool 的 `UsageStats`：`UsageCount`、`SuccessCount`、`FailureCount`、`SuccessRate`、`AverageLatency`、`LastUsedAt`。统计同时写入 `tools/<id>/metadata.json` 和 `registry/metadata/tools/<id>.json`，检索侧读取后者。

只有 Executor 真正执行过的结果才计入统计。生命周期不合规、依赖不健康、策略拒绝或要求审查而被拦下的调用记入执行历史和审计，但不改变成功率——否则需要审查的工具会因为被拦截而看起来不可靠。

这一步在本地文件存储上是「先私有再公开」的两步写，崩溃可能让两份 metadata 短暂不一致；下一次执行会从同一份聚合结果修复。`UsageRecorder` 是可选的：不接入时只保留执行历史，排序器继续使用默认值。

排序公式见 `internal/adapter/retriever/local/ranker.go`，语义相似度占 0.55，成功率/可靠性/延迟/使用量/环境匹配合计 0.45。执行几次不同结果后，可以观察 `search tools` 输出的 `score` 变化。尚未实现连续失败自动隔离。

## 自动化测试

普通测试不依赖 Docker 或模型 API：

```powershell
go test ./... -count=1
go vet ./...
go build ./...
```

显式启用真实 Docker 隔离测试：

```powershell
$env:EVOTOOL_DOCKER_TEST='1'
$env:EVOTOOL_HOST_SECRET='must-not-reach-container'
go test ./internal/adapter/sandbox/docker -run TestDockerIsolation -v -count=1
Remove-Item Env:EVOTOOL_DOCKER_TEST
Remove-Item Env:EVOTOOL_HOST_SECRET
```

2026-11-06 在 Windows + Docker Desktop Linux 引擎实测：一键验收通过；禁网、只读根文件系统、非 root、宿主环境变量隔离、超时、内存限制、输出限制和失败测试的容器集成测试全部通过。示例声明 30 秒时限，包含容器启动时间。首次冷启动可能超过工具声明的运行时长并被拒绝，可预热镜像后重试；失败验证不会发布 Tool。
