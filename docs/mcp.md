# MCP Adapter 原型

当前 `internal/adapter/mcp/server` 是可嵌入的只读 Adapter 原型，不是完整可部署的 MCP Server，也未完成外部 MCP Client 兼容验收。

## 已有接口

JSON-RPC Handler 支持 `initialize`、`tools/list` 和 `tools/call`，提供两个只读操作：

| 操作 | 参数 | 行为 |
| --- | --- | --- |
| `evotool.search_tools` | `{"intent":"..."}` | 调用 CapabilityMemoryService 检索 Tool |
| `evotool.get_tool` | `{"id":"..."}` | 读取已保存的 Tool |

装配方需注入 `Server.Memory`。`Server.ServeStdio(ctx, reader, writer)` 使用逐行 JSON 请求；`Server.ServeHTTP` 可挂载到 HTTP POST 路由。传输层限制请求为 2 MiB，HTTP 拒绝非 POST 请求。Adapter 委托已有 Service，不重写检索或存储业务。

## 尚未完成

- 没有 `evotool mcp` CLI 启动命令或开箱即用的服务部署入口。
- 未完整实现通知和初始化生命周期、版本协商、会话以及 Streamable HTTP 协议。
- 未提供 HTTP 认证和授权配置。
- 未暴露学习、执行、审查和 Benchmark 等写入或执行操作。
- 未在真实 MCP Client 中完成端到端验证。

生产接入与 Eino/OpenHands Adapter 仍属于后续工作；现阶段请使用 Go API 或 [CLI 学习与执行流程](learning.md)。
