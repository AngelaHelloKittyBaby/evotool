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

执行后的记录使用独立的最长五秒上下文，避免任务取消直接丢弃失败历史。历史、审计和使用统计不是原子事务；记录失败必须由调用方处理，不能盲目重跑具有副作用的工具。配置 `UsageRecorder` 后，实际执行结果会聚合更新 `ToolUsageStats` 和两份元数据；CLI 已装配该依赖。执行前被策略或依赖拦截的尝试不计入使用经验。

现已提供 Python Executor、Docker Sandbox 和 CLI `learn` / `generate` / `execute` 命令，详见 [提交与验证流程](learning.md)。允许 Python 标准库和预置批准的固定版本 Library；第三方包、传递库依赖及可选引用仍被拒绝。学习时绑定库版本及摘要，每次执行重新检查版本、内容、Manifest 和信任状态；Tool 不允许覆盖库使用的 `shared/` 命名空间。详见 [可执行共享库](library-execution.md)。配置方必须注入能落实资源和权限边界的 Executor，不能将普通宿主进程启动器当成安全沙箱。Library 独立学习、审查及自动重验证入口尚未实现。

生成代码中的删除、覆盖和数据库写入目前无法从权限模型中单独表达；宿主路径与命令访问因此统一要求审查。禁止通过缺少权限声明就直接在宿主环境运行代码。完整执行链仍需验证、依赖隔离、审计和沙箱配合实现。

## 模型生成边界

模型调用位于宿主的 Generator Adapter，密钥不传给生成代码或 Docker。模型输出始终是不可信候选包，必须通过验证和策略流程后才能发布；无自动重试或验证失败后强制保存。API 响应中的 usage 仅用于计量，不代表代码已验证。真实 Provider 兼容性与生成质量需要单独验收，见 [模型生成](generation.md)。
