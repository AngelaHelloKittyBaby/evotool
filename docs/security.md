# 安全策略

`service.PolicyService` 实现 `ports.PolicyEngine`，负责执行前的声明式策略判断。装配方必须显式配置正数的运行时间、内存和输出大小上限。核心服务不依赖具体执行器或 Agent 框架。

允许执行的基础条件：

- 生命周期为 `approved`、`active` 或 `stale`。
- 信任等级为 `validated`、`reviewed` 或 `trusted`。
- 显式指定当前版本。
- Tool 的三个资源限制均为正数且不超过宿主策略上限。

未声明网络权限时按禁止网络处理。未知网络策略和运行环境与权限之间的网络冲突直接拒绝。声明网络、宿主文件路径、系统命令或环境变量访问时，返回 `Allowed=false`、`RequiresReview=true`，并提供原因。拒绝项与审查项分别记录；需要审查不表示其他拒绝原因可以被忽略。

当前模型尚未保存与具体版本及权限绑定的审查证明，因此即使信任标签为 `reviewed` 或 `trusted`，也不能绕过特权审查。后续审查接入必须绑定版本、代码和权限，不能只靠修改标签放行。

策略判断只检查元数据，不会执行代码或证明代码安全，也不实施文件、网络、内存和进程隔离。

## 执行编排

`service.ExecutionService` 接收 ToolStore、DependencyResolver、PolicyEngine、ToolExecutor、ExecutionRecorder 和 AuditLogger；全部依赖都必须显式提供。可通过 `CapabilityMemoryServiceConfig.ExecutionService` 接入门面的 `ExecuteTool` 方法。

流程是：读取存储的 Tool → 确认当前版本存在 → 解析依赖且健康状态为 `healthy` → 策略允许且无需审查 → 写入开始审计 → 调用 Executor → 记录结果及完成/失败审计。依赖错误、策略拒绝、审查要求和开始审计失败均阻止执行，并记录失败尝试。依赖检查使用现有 Resolver；缺少 Registry 依赖图时默认报错，不将其视为无依赖。

Executor 返回的 ToolID、版本及时间戳由服务覆盖为当前调用的真实信息。执行器返回错误时强制标记失败；执行器仅返回 `Success=false` 时也向调用方返回错误。记录和审计失败通过错误链返回，不会丢弃执行结果或原始执行错误。

执行后的记录使用独立的最长五秒上下文，避免任务取消直接丢弃失败历史。历史和审计不是原子事务；记录失败必须由调用方处理，不能盲目重跑具有副作用的工具。当前只追加执行历史，尚未聚合更新 ToolUsageStats。

现已提供 Python Executor、Docker Sandbox 和 CLI `learn` / `execute` 命令，详见 [提交与验证流程](learning.md)。这条执行路径只允许无外部依赖的 Python 工具，声明第三方包或 Library 引用会被拒绝。配置方只能注入能落实资源和权限边界的 Executor；不能将普通宿主进程启动器当成安全沙箱。现有通用依赖检查主要覆盖 Library 存在性和版本，完整依赖声明覆盖、传递依赖、Library 审批及隔离解析仍需进一步实现。

生成代码中的删除、覆盖和数据库写入目前无法从权限模型中单独表达；宿主路径与命令访问因此统一要求审查。禁止通过缺少权限声明就直接在宿主环境运行代码。完整执行链仍需验证、依赖隔离、审计和沙箱配合实现。
