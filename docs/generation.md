# 模型生成与执行 Benchmark

`ports.ToolGenerator` 现在有 Chat Completions HTTP Adapter，核心 Service 不依赖模型 SDK。请求和 usage 契约参考 [Chat Completions API](https://developers.openai.com/api/reference/resources/chat)。兼容端点需支持 `response_format=json_object` 和 `max_completion_tokens`，不同 Provider 的兼容性需要实际验证。

## 配置与生成

在仓库根目录的 PowerShell 配置宿主环境变量，不要将密钥写进 task 或源码：

```powershell
$env:EVOTOOL_MODEL_ENDPOINT='https://你的服务/v1/chat/completions'
$env:EVOTOOL_MODEL='你的模型名称'
# 通过本机密钥管理方式设置 EVOTOOL_MODEL_API_KEY；本地免认证模型可不设置。
go run ./cmd/evotool generate --task examples/csv-analysis/task.json --root tmp/model-memory
```

端点是完整请求地址，允许 HTTPS 或 loopback HTTP。不隐式选择 Provider，不读取 `.env`，不自动重试。默认请求超时 120 秒，最大 completion token 8192；Go 装配方可通过 Config 修改。模型密钥不会进入提示词或 Docker。

流程：TaskSpec → 搜索最多五个 Library 契约 → 调模型生成 JSON → 解析候选包 → 解析库固定版本 → 命名/Schema/权限检查 → Docker 测试和样例验证 → 保存 Tool、Registry 和生成信息。无效 JSON、拒绝、截断、未知字段、依赖错误及测试失败均不会发布工具；已有工具不覆盖。

`generate` 是显式生成命令，不自动执行用户输入，也不代替 Agent 先搜索 Tool。成功输出 `tool_id` 和 `validation.Valid=true`。示例要求模型命名为 `csv_numeric_summary`，如实际返回不同名称，以 `tool_id` 为准：

```powershell
go run ./cmd/evotool execute --id csv_numeric_summary --input examples/csv-analysis/input.json --root tmp/model-memory
```

应出现 `Success=true`，输出 count=3、total=60、average=20。模型可能生成错误代码，不能保证每次成功；不能手动修改 approved 标签来绕过失败验证。

## 实际运行 Benchmark

```powershell
go run ./cmd/evotool benchmark live --suite examples/benchmark/csv-live-suite.json --root tmp/live-model-memory --out tmp/live-model-report.json
```

使用独立空目录作为 root，复跑请换新目录；live 会学习工具、写执行历史及统计。两任务空库理想情况：Baseline 调模型两次，EvoTool 第一次生成、第二次复用（总计三次请求）。该命令实际消耗配置的 API 用量。

每项任务必须有 Name、Spec.Intent、Input、Expected。Expected 是人提供的独立核验结果，不发给模型。live 忽略 suite 的 Package 和 TokenUsage。Baseline 每项重新生成、验证和执行，不发布工具；EvoTool 在检索未命中时生成、验证、保存、执行，后续任务可复用本轮学到的能力。

- `SuccessRate` / `FailureRate`：全任务为分母，生成、验证、执行、输出核验或持久化失败都计失败。
- `TotalTokenUsage`：仅 Provider envelope 返回的 usage，不接受模型正文伪造用量或 suite 估算。
- `MissingUsageCount`：未获得完整合法 usage 的生成请求数；大于 0 时汇总只代表已知部分，不能计算完整费用或节省。
- 每项 Generation 保留模型、响应 ID、UsageKnown；无效模型输出仍保留已收到的用量。
- `AverageLatency` 是生成/检索、验证、执行和输出核验的平均总时长（纳秒）。
- `ToolGeneratedCount` 是生成尝试，`ToolReusedCount` 是命中后执行尝试；实际成功与否见 Tasks。LibraryReusedCount 是无错误执行使用的库引用数量。

命中但执行失败不会静默重新生成，模型失败不重试。Baseline 始终先运行，冷启动及网络状态也影响时长。当前成功率只衡量固定测试输入，不代表模型任意任务表现；部分通用计数字段（如 DependencyFailureCount）尚未在 live 路径独立分类，具体错误请查看 Tasks。

## 不花模型费用的验证

```powershell
go test ./... -count=1
go vet ./...
go build ./...
```

普通测试覆盖 HTTP 和 usage、取消、生成失败、验证失败、输出错误和缺失 usage，不依赖模型 API。

准备 Docker Desktop Linux 引擎及 `python:3.13-alpine` 镜像后：

```powershell
$env:EVOTOOL_DOCKER_TEST='1'
go test ./cmd/evotool -run TestLiveBenchmarkDocker -count=1 -v
Remove-Item Env:EVOTOOL_DOCKER_TEST
```

此测试已实际通过：本地模拟模型 HTTP API + 真实 Docker，两个模式得到正确 CSV 结果，Baseline 请求两次、EvoTool 请求一次并复用一次，usage 汇总并写入 Tool。模拟用量为 100 token/请求，不是供应商模型账单。供应商模型质量和实际费用仍需配置自己的端点后现场验证。
