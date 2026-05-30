# Orchestration Map

更新时间：2026-05-29

## 范围

本盘点覆盖 `core/bridge/orchestration` 结构整理 M2 后状态。

- 总 Go 文件：288。
- 顶层 Go 文件：70（生产 41，测试 29）。
- 非测试文件超过 300 行：0。
- 超过 15 个 Go 文件的目录：2。

当前 M2 守卫位于 `core/bridge/orchestration/internal/structure_guard_test.go`，随
`go test ./orchestration/... -timeout 60s` 执行。

## 目标子域

| Target | Responsibility |
| --- | --- |
| `dispatch` | HTTP/bus/service facade、任务入口、工具 action 分派。 |
| `loop` | Agent 回合、session turn、plan/relay、group orchestration runner。 |
| `workflow` | workflow 图校验、节点执行、screen step/find_icon 协调。 |
| `policy` | runtime config、runtime overrides、validation、tool policy、session guards。 |
| `trace` | `trace_id`、stream event、session push、checkpoint、run/session projection。 |

## M2 结构结论

| Metric | Current | M2 Gate |
| --- | ---: | ---: |
| 顶层 Go 文件数（含测试） | 70 | <=70 |
| 非测试 >300 行文件数 | 0 | <=1 |
| >15 文件目录数 | 2 | <=3 |

仍超过 15 个 Go 文件的目录：

| Directory | Go Files | Next Target |
| --- | ---: | --- |
| `core/bridge/orchestration/internal/app/workflows` | 21 | M3 继续按 node/screen/path 拆子目录。 |
| `core/bridge/orchestration/internal/app/orchestrations` | 19 | M3 继续按 runner/owner/dispatch 拆子目录。 |

## 已完成

### M1
- 删除空壳 skill 顶层文件，skill 行为继续由 `bridge/skills.ActionHandler` 承接。
- 合并顶层小型 shim/façade，公开 API 类型与 `Service` 方法签名保持软兼容。
- 拆分 `session_turn_preparer.go` 与 `internal/domain/sessionturn/microcompact.go`，生产文件均不超过 300 行。
- 合并少量同主题测试文件，顶层测试仍覆盖原有契约场景。
- 补 M1 结构守卫，阻止顶层数量、生产文件行数、目录文件数指标回退。

### M2
- 新增 `internal/dispatch/`，承载 action router、参数解码、默认 action 注册表、task scope dispatch、tools/sessions action 常量与 dispatch 白盒测试。
- 新增/增强 `internal/trace/`，收口 run registry、session push、stream checkpoint、terminal buffer、session/run projection，并将 turn draft projection 下沉到 `internal/trace/sessiondraft`。
- 顶层 `orchestration` 保留 facade：`Service`/router/trace compat 仍存在，只委托 internal 包。
- 将 trace draft projection 测试下沉到 `internal/trace/sessiondraft`，结构守卫升级为 M2 预算。
- 清理了迁移后未使用的顶层 `taskListParams` 和旧注册 helper。
- action 路由、service action handlers、tasks/presets/prompts/skills/tools/sessions 分发逻辑迁入 `internal/dispatch/*`。
- run registry、session push、stream checkpoint、terminal buffer 等迁入 `internal/trace/*`。

## M3 计划

- `loop`：session turn prepare/run/resume/history/relay/plan mode 全量下沉到 `internal/loop/*`。
- `policy`：runtime config、runtime overrides、validation、tool policy、session guards 下沉到 `internal/policy/*`。
- `workflow`：workflow/orchestration task runner、node 执行、screen step/find_icon 协调逻辑下沉到 `internal/workflow/*`。
- 顶层仅保留 façade、合约类型与必要兼容桥接。
- 量化目标：顶层 <=30，非测试 >300 行 =0，>15 文件目录 <=1。

## 边界

- `core/bridge/app` 与 `core/bridge/transport` 继续只 import 顶层 `orchestration` facade。
- 跨进程 payload 仍遵守 `core/shared/schema.json` 的 request/response 结构。
- `trace_id` 透传路径未改变；本轮未新增 fallback、模拟成功或静默降级。
