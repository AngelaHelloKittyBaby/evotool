# 验证记录

这是 2026-11-04 的初始基线记录。2026-11-05 已新增实际 Python 执行、Docker 沙箱和外部提交验证流程；最新验收方法及结果见 [learning.md](learning.md)。下方“当前验证边界”描述的是初始基线，不是最新状态。

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

## 当前验证边界

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
