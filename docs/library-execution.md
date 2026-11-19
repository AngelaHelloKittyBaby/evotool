# 可执行共享库

Tool 的 `Manifest.LibraryRefs` 现在可以引用已经由宿主管理的、批准且验证过的 Python Library。普通外部包依赖和传递 Library 依赖仍明确拒绝，不会隐式安装依赖。

```json
{"LibraryRefs":[{"ID":"numeric_core","Kind":"library","Version":"v1"}]}
```

学习流程先解析库并绑定 `Digest`，验证流程将库和 Tool 一起放进沙箱运行测试。保存 Tool 时记录版本、摘要及依赖图；执行器再次解析同一版本并检查摘要。库的版本、内容、Manifest、信任或批准状态变化后，依赖 Tool 会被拒绝执行，需重新验证；当前还没有自动重验证和版本提升入口。

库源码位于沙箱内的 `shared/<Library.Name>/`。LibraryVersion.Files 中 `source/` 前缀会被去掉，其他相对路径保留。例如：

```text
numeric_core v1: source/stats.py
    -> shared/numeric_core/stats.py
```

Tool 可以使用：

```python
from shared.numeric_core.stats import total
```

Tool 不允许在自己的源码中覆盖 `shared/`。重复模块、路径穿越、未批准库、缺少版本、特权库、可选引用以及传递依赖都会拒绝。库不会被复制进保存的 Tool 源码；只在每次验证和执行时注入临时沙箱。

文件系统 LibraryStore 新增 `List`；通过可选 `ports.LibraryLister` 使用，避免强迫只支持按 ID 读取的 Store 实现额外接口。

## 验证

普通回归（不需要 Docker）：

```powershell
go test ./... -count=1
```

运行 Docker Desktop Linux 引擎、准备 `python:3.13-alpine` 镜像后：

```powershell
$env:EVOTOOL_DOCKER_TEST='1'
go test ./internal/service -run TestLibrarySharedByTwoTools -count=1 -v
Remove-Item Env:EVOTOOL_DOCKER_TEST
```

该测试在临时目录注册一个开发者可信库，用两个独立 Tool 真实导入它，经过学习、测试、样例验证、存储和执行，并检查修改库内容后两个 Tool 均被拦截。成功标志为 `--- PASS: TestLibrarySharedByTwoTools`。

Library 本身的自动学习/独立审查入口尚未提供；本轮支持预置的批准库，不能把未验证的 Agent 库直接标为可信。[ToolGenerator 与 live Benchmark](generation.md) 已实现，并会在生成前搜索库契约；真实供应商的模型质量和 token 用量收益尚未实测。
