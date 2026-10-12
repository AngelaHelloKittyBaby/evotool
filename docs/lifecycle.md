# EvoTool 工具生命周期与保留策略

## 1. 目标

EvoTool 会持续接收外部 Agent 生成的 Tool。如果所有 Tool 永久处于可检索状态，能力库会逐渐出现：

- 检索候选变多，噪声增加。
- 旧版本和新版本同时参与选择。
- 依赖失效或运行环境变化后仍被推荐。
- 低质量、重复或一次性 Tool 占据 Registry。
- 管理员无法判断哪些能力仍然值得维护。

生命周期管理的目标是：

1. 让活跃且可靠的 Tool 优先参与检索。
2. 让长期未使用的 Tool 降低检索噪声。
3. 在删除之前保留恢复机会。
4. 防止误删低频但重要的工具。
5. 让每一次状态变化都可审计、可解释。

## 2. 状态流转

```text
draft
  -> validated
  -> approved
  -> active
  -> stale
  -> archived
  -> deleted
```

风险状态可以从任意可执行状态进入：

```text
active / stale
  -> quarantined
```

状态说明：

| 状态 | 是否默认检索 | 是否允许执行 | 说明 |
| --- | --- | --- | --- |
| `draft` | 否 | 否 | 刚生成，尚未完成验证 |
| `validated` | 否或低优先级 | 按策略 | 验证通过，但还没有批准自动复用 |
| `approved` | 是 | 是 | 已通过验证和策略检查 |
| `active` | 是 | 是 | 当前正在使用和维护的能力 |
| `stale` | 默认降低优先级 | 按策略 | 长时间未使用，等待归档评估 |
| `archived` | 否 | 默认否 | 保留文件和元数据，可恢复 |
| `quarantined` | 否 | 否 | 发现风险、依赖问题或连续失败 |
| `deprecated` | 否或低优先级 | 按策略 | 有更好的替代版本 |
| `deleted` | 否 | 否 | 已经完成物理删除或逻辑删除 |

`active`、`stale` 和 `archived` 是保留策略需要新增或明确的治理状态。具体实现可以复用现有 `LifecycleStatus`，也可以在不破坏兼容性的前提下通过策略字段表达“是否进入默认检索”。

## 3. 使用统计

生命周期服务需要依赖执行历史和工具元数据，至少关注：

```yaml
usage_count: 120
success_count: 116
failure_count: 4
success_rate: 0.966
last_used_at: 2026-10-10T10:00:00Z
last_successful_at: 2026-10-10T10:00:00Z
last_validated_at: 2026-10-09T12:00:00Z
```

不能只按照 `last_used_at` 删除工具。以下情况需要阻止自动归档或删除：

- Tool 被其他 Tool 或 Library 依赖。
- Tool 被标记为 `pinned`。
- Tool 设置了 `retain_until`。
- Tool 属于关键业务、应急任务或季节性任务。
- Tool 没有可用替代版本。
- Tool 最近刚刚验证或恢复。

## 4. 推荐策略

策略应该由部署者配置，不应该写死在业务代码中。例如：

```yaml
retention:
  stale_after: 90d
  archive_after: 180d
  purge_after: 365d
  require_no_dependents: true
  require_explicit_delete_confirmation: true
  protect_pinned_tools: true
  allow_archived_search: true
```

推荐解释：

- `stale_after`：超过这个时间没有使用，标记为 `stale`。
- `archive_after`：持续未使用达到这个时间，归档。
- `purge_after`：归档达到这个时间后，才生成删除候选。
- `require_no_dependents`：仍被依赖时禁止删除。
- `require_explicit_delete_confirmation`：删除不允许由后台扫描静默完成。
- `protect_pinned_tools`：受保护 Tool 不参与自动清理。
- `allow_archived_search`：没有匹配能力时允许显式检索归档区。

对于 MVP，建议使用：

```text
扫描 -> stale -> archived -> 可恢复
```

物理删除应等到执行、依赖、审计和恢复机制稳定后再实现。

## 5. 检索行为

默认检索只读取可用能力：

```text
普通任务
  -> 检索 active / approved Tool
  -> 过滤 quarantined / archived / deleted
```

当普通检索没有合适结果时，可以由 Agent 或管理员显式开启归档检索：

```text
普通检索无结果
  -> 查询 archived Tool
  -> 检查版本、依赖和安全状态
  -> 恢复或重新验证
  -> 再决定是否执行
```

归档 Tool 不能因为被重新搜到就自动恢复为 `approved`。它必须重新通过依赖检查、验证和策略检查。

## 6. 计划中的服务和命令

后续 `LifecycleService` 应负责：

- 扫描 Tool 使用统计。
- 计算生命周期候选。
- 生成扫描报告。
- 支持 `dry-run` 预览。
- 标记 `stale`。
- 归档和恢复 Tool。
- 检查依赖关系和保护标记。
- 生成删除候选。
- 记录生命周期审计事件。

当前已经实现的只读扫描命令：

```powershell
go run ./cmd/evotool lifecycle scan --root .evotool
```

默认策略是 90 天进入 `stale` 候选，180 天进入归档候选。可以通过参数调整：

```powershell
go run ./cmd/evotool lifecycle scan --root .evotool --stale-days 30 --archive-days 90
```

该命令只读取并输出候选，不会修改 Tool 文件或生命周期状态。

后续计划中的 CLI 形式：

```powershell
go run ./cmd/evotool lifecycle archive --dry-run --root .evotool
go run ./cmd/evotool lifecycle restore --id pdf_to_excel --root .evotool
go run ./cmd/evotool lifecycle purge --dry-run --root .evotool
```

`archive`、`restore` 和 `purge` 目前是设计目标，尚未代表当前版本已经实现。

## 7. 安全要求

生命周期清理属于破坏性操作，必须满足：

1. 默认先归档，不直接删除。
2. 删除前检查 Tool 和 Library 依赖图。
3. 删除前检查 `pinned`、`retain_until` 和关键业务标记。
4. 删除前提供 `dry-run` 报告。
5. 删除操作必须显式确认。
6. 删除前保存审计事件和必要的摘要信息。
7. 删除后不能让 Registry 残留指向不存在文件的元数据。
8. 失败时保留原状态，不能出现“文件删了但 Registry 还显示可用”的不一致。

## 8. 验收标准

生命周期功能真正实现后，至少需要验证：

- 从未使用的 Tool 可以被识别为候选。
- 最近使用的 Tool 不会被误归档。
- `pinned` Tool 不会被自动归档或删除。
- 被其他能力依赖的 Tool 不会被删除。
- 归档 Tool 默认不会出现在普通检索结果中。
- 归档 Tool 可以恢复。
- 恢复后必须重新验证依赖和策略。
- `dry-run` 不会修改 Tool 文件或状态。
- 每次状态变化都有审计记录。
