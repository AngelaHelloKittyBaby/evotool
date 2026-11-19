# 文档导航与实现状态

更新日期：2026-11-19。架构与生命周期文档包含设计目标；当前可用范围及验证证据以本页和 [验证记录](verification.md) 为准。

| 主题 | 当前状态 | 使用与验证 |
| --- | --- | --- |
| 外部 Agent 提交 Tool | Python 包经过策略、测试及 Docker 样例验证后保存 | [学习流程](learning.md) |
| 可执行 LibraryRefs | 复用预置、批准的 Python 库，固定版本及摘要，执行前重新检查 | [共享库](library-execution.md) |
| ToolGenerator | 可配置 Chat Completions HTTP Adapter，生成前检索最多五个库契约 | [模型生成](generation.md) |
| 静态 Benchmark | 重放复用决策，token 来自 suite，不执行工具 | [静态 Benchmark](benchmark.md) |
| Live Benchmark | 实际生成、沙箱验证、执行和独立输出核验，记录响应 usage | [Live Benchmark](generation.md#实际运行-benchmark) |
| 检索与运行经验 | 关键词召回、元数据过滤、排序、执行反馈；未实现向量检索 | [架构设计](architecture.md) |
| 生命周期 | 只读扫描 stale / archive 候选；归档、恢复、清理尚未实现 | [生命周期设计](lifecycle.md) |
| MCP | 只读 Adapter 原型，可嵌入；不是完整可部署 MCP Server | [MCP 边界](mcp.md) |

## 如何确认验证成功

在仓库根目录运行：

```powershell
go test ./... -count=1
go vet ./...
go build ./...
```

准备 Docker Desktop Linux 引擎和 `python:3.13-alpine` 镜像后，执行两条集成测试：

```powershell
$env:EVOTOOL_DOCKER_TEST='1'
go test ./internal/service -run TestLibrarySharedByTwoTools -count=1 -v
go test ./cmd/evotool -run TestLiveBenchmarkDocker -count=1 -v
Remove-Item Env:EVOTOOL_DOCKER_TEST
```

两条测试均须出现对应测试名的 `--- PASS`，不能只有 `SKIP`。前者证明两个 Tool 实际共享一个库、库变更后均被拦截；后者使用模拟模型 HTTP 响应和真实 Docker，证明生成、学习、复用、输出核验与 usage 记录能走通。它们不证明真实供应商模型的生成质量或费用收益。

真实模型验收需自行配置端点，再按 [模型生成指南](generation.md) 运行 `generate` 和 `benchmark live`。当前没有供应商实测结论；请勿将模拟 token 或静态估算当成真实账单。

## 设计与约束

- [整体架构](architecture.md)
- [安全模型](security.md)
- [共享库依赖管理](library-execution.md)
- [阶段性设计长文](blog-capability-memory.md)

尚未完成的主要能力：Library 独立学习与审查入口、传递依赖及第三方包解析、自动依赖重验证、高风险审查批准、向量检索、完整 MCP/Eino/OpenHands 接入、自动修复和公共库抽取。
