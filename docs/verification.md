# 验证记录

## 最新验证：2026-11-19

环境：Windows、Go 1.25、Docker Linux 引擎与 `python:3.13-alpine`。已实际执行：

| 检查 | 结果与范围 |
| --- | --- |
| `go test ./... -count=1` | 全部通过；默认不启用 Docker 集成测试 |
| `go vet ./...`、`go build ./...` | 通过 |
| `TestLibrarySharedByTwoTools`（启用 `EVOTOOL_DOCKER_TEST=1`） | 真实 Docker 运行两个共享库 Tool，库变更后两者均被拦截 |
| `TestLiveBenchmarkDocker`（同上） | 模拟模型 HTTP API + 真实 Docker，验证生成、学习、执行、复用、输出核验和 usage 持久化 |

Live 集成测试中，两模式的输出核验均通过：Baseline 生成两次，EvoTool 生成一次并复用一次；模拟 API 返回每次 100 token，因此汇总为 200 与 100。该数字仅是测试断言，不是供应商账单或成本收益结论。

尚未验证：真实 Provider 的接口兼容性、模型生成质量和费用；完整 MCP 客户端接入；Library 自动学习与重验证；传递依赖与第三方包安装。未在本轮重新测量覆盖率，下面的覆盖率仅为历史数据。

可复制的命令和成功标志见 [文档入口](README.md#如何确认验证成功)、[共享库验证](library-execution.md#验证) 和 [生成与 live Benchmark](generation.md)。

## 历史基线：2026-11-04

以下保留初始基线与当时的缺口，不能作为当前实现状态。2026-11-05 后的外部提交、沙箱执行验收另见 [learning.md](learning.md)。

验证日期：2026-11-04。环境：Windows amd64，Go 1.25.0。

## 已实际执行

| 检查 | 结果 |
| --- | --- |
| `go test ./... -count=1` | 全部通过，无跳过包测试的额外配置 |
| `go vet ./...` | 通过 |
| `go build ./...` | 通过 |
| CLI 独立二进制测试 | 保存 Tool、保存 Library、两种搜索、关联依赖、查询依赖图、检查依赖、生命周期只读扫描均通过 |
| CLI 错误输入 | 缺少搜索 query 返回非零退出码 |
| `go test -race ./...` | 未运行成功，当前环境未启用 CGO |

单包语句覆盖率：CLI 69.5%，本地检索 70.0%，文件存储 58.2%，Service 76.1%；总计 69.8%。domain 和 ports 没有独立测试文件。此覆盖率按默认单包统计，跨包调用不计入被调用包覆盖率。

## 发现并修复

文件存储此前按 Name 保存，却按 ID 读取。ID 与 Name 不同会出现保存成功、读取失败。新增 `TestStoresRoundTripDistinctIDAndName` 先复现失败，再统一 Tool、Library 及 Registry 的存储键为 ID 优先，修复后全量测试通过。未提供 ID 时仍使用 Name。

已有 ID 与 Name 不同、按旧 Name 目录保存的数据不会自动迁移；需要从原始 metadata 重新保存。通常 ID 与 Name 相同的 Demo 数据不受影响。

## 初始基线的验证边界

- ExecutionService 的执行测试使用 fake Executor，不能证明真实生成代码的执行与隔离。
- PDF-to-Excel Demo 只返回推导出的 xlsx 路径，不解析 PDF，也不生成 Excel 文件。
- 当前检索是本地关键词召回，不是 embedding/vector 语义检索。
- 尚无具体 Generator、Validator、Sandbox、Benchmark Runner、Eino 或 MCP Adapter，无法验证完整的生成、沙箱验证、执行闭环及 token/耗时收益。
- 生命周期验证涵盖扫描与候选判断，不代表已支持归档、恢复或删除。
- 本次没有执行生成的 Python 文件，也没有安装外部依赖或修改已有 `.evotool` 数据。

## 本地复查

在仓库根目录使用 PowerShell：

```powershell
go test ./... -count=1 '-coverprofile=coverage.out'
go tool cover '-func=coverage.out'
go vet ./...
go build ./...
```

CLI 可按 README 的 Quick Start 顺序复查；请使用单独的 `--root tmp/manual-check`，避免覆盖原有能力记忆。测试与构建产物放在忽略的 `tmp/verification/`，覆盖率数据位于 `tmp/verification/coverage.out`。
