# Orchestration Map

更新时间：2026-05-31

## 范围

本盘点覆盖 `core/bridge/orchestration` 当前工作区状态，并把原 M1-M4
计划更新为从现状继续下沉的执行口径。当前工作区存在未提交改动，以下指标以
本地文件扫描与 `go test ./orchestration/... -timeout 60s` 结果为准。

- 总 Go 文件：272。
- 顶层 Go 文件：92（生产 74，测试 18）。
- 非测试文件超过 300 行：3。
- 超过 15 个 Go 文件的目录：3。

当前结构守卫位于 `core/bridge/orchestration/internal/structure_guard_test.go`。
守卫仍按 M2 预算运行，并且当前失败点为：顶层 Go 文件 92，超过 M2 门槛 70。
生产文件行数也已回退，需要在下一步一起修复。

## 当前超限项

### 生产文件超过 300 行

| File | Lines | Target |
| --- | ---: | --- |
| `task_run_cards.go` | 311 | 下沉到 `internal/trace/runcards` 或同职责 trace 子包。 |
| `task_orchestration_owner_runtime.go` | 311 | 下沉到 owner turn loop / orchestration owner 子域。 |
| `internal/trace/sessiondraft/projector.go` | 322 | 同包拆为 lifecycle / completion / tool projector。 |

### 超过 15 个 Go 文件的目录

| Directory | Go Files | Target |
| --- | ---: | --- |
| `core/bridge/orchestration` | 92 | M2.1 回到 <=70，M3 压到 <=30，M4 压到 <=15。 |
| `core/bridge/orchestration/internal/app/workflows` | 21 | M3 拆为 `runner` / `nodes` / `screen` / `path` 等子域。 |
| `core/bridge/orchestration/internal/app/orchestrations` | 19 | M3 拆为 `runner` / `owner` / `member` / `dispatch` 等子域。 |

## 目标边界

顶层 `orchestration` 的终态职责只保留：

- 对外 façade：`Service` 构造、生命周期、公开方法兼容入口。
- 合约类型：外部调用方仍需要引用的 request/response/type alias。
- transport/app usecase 入口：`core/bridge/app` 与 `core/bridge/transport` 只能依赖顶层
  façade，不能直接 import 新 internal 子包。
- 生成入口：必须留在顶层的 generated contract 文件，除非生成链路同步迁移。

具体实现继续下沉到以下子域：

| Target | Responsibility |
| --- | --- |
| `internal/dispatch` | action router、参数解码、service action handler 注册、task scope dispatch。 |
| `internal/app/agentturn` / `internal/loop/*` | agent turn prepare/run/resume/history/relay/plan mode。 |
| `internal/app/workflows` | workflow 图校验、节点执行、screen step、find_icon 协调。 |
| `internal/app/orchestrations` | group orchestration runner、owner/member runner、dispatch executor。 |
| `internal/app/{config,prompts,skills,tasks,tools}` | service usecase 的领域入口，顶层只委托。 |
| `internal/policy` 或 `internal/domain/runtimeopts` | runtime config、runtime overrides、validation、tool policy、session guards。 |
| `internal/trace` | `trace_id`、stream event、session push、checkpoint、run/session projection、run cards。 |

## 更新后的执行计划

### M2.1：现实重校准与 M2 守卫恢复

目标是不改运行语义，先让当前工作区重新满足 M2 门槛。

- 顶层从 92 压回 <=70；优先迁走新近回流的 task/session/workflow/orchestration
  辅助文件，而不是放宽守卫。
- 拆分 3 个超过 300 行的生产文件，生产文件全部回到 <=300 行。
- 保持 `internal/app/workflows` 与 `internal/app/orchestrations` 暂不扩容；若迁入新文件，
  必须同步拆子目录，避免把超限从顶层转移到 app 目录。
- `service_router.go` 暂时只允许承担 service 装配与 façade 委托；RSS、skills、runner、
  runtime state、task scheduler 等具体构建逻辑不得继续扩写在顶层。
- 必跑验收：`cd core/bridge && go test ./orchestration/... -timeout 60s`。

### M3a：dispatch 与 policy 二次收口

- 将 `service_config_*`、`service_prompts.go`、`service_skills.go`、
  `service_tools*.go`、`service_sessions*.go`、`service_tasks.go` 的业务分支迁入
  `internal/app/*` 与 `internal/dispatch`。
- 顶层 `Service` 方法保留签名，只做 decode、调用 internal usecase、返回
  `ServiceResult` 的薄桥接。
- runtime config、runtime overrides、tool allowlist、session guard、validation
  收到 `internal/policy` 或既有 `internal/domain/runtimeopts`，不要让 task/session
  runner 直接散落读取配置。
- 顶层 action router 只保留兼容注册入口；默认 action 表在 `internal/dispatch` 维护。

### M3b：agent turn / relay / plan loop 下沉

- 迁移 `session_turn_preparer*`、`session_runner*`、`session_resume_runner.go`、
  `service_usecase_agent.go`、`service_usecase_human.go`。
- 迁移 `relay_mode_*` 与 `plan_mode.go`，让 relay/plan 成为 loop usecase，而不是顶层
  mode 分支。
- session history、human resume、stream checkpoint 与 turn draft 只通过 loop + trace
  接口协作，避免 session 代码直接操作推送、运行注册和 runtime 构造细节。
- 保留顶层黑盒契约测试；新增/迁移 loop 白盒测试覆盖 resume、awaiting human、plan mode、
  relay handoff。

### M3c：workflow 与 group orchestration 下沉

- 将 `task_workflow_*` 迁到 `internal/app/workflows/{runner,nodes,screen,path}`，
  文件数按子域控制在每目录 <=15。
- 将 `task_orchestration_*` 迁到
  `internal/app/orchestrations/{runner,owner,member,dispatch}`，owner runtime 与 repair
  逻辑跟 owner 子域放在一起。
- `task_executor_adapter.go`、`task_usecase_runner*` 只保留任务 façade 所需桥接；
  具体 workflow/orchestration runner 通过 ports 注入。
- workflow/orchestration run cards 统一走 `internal/trace` 的 run-card recorder，避免
  task、workflow、orchestration 各自拼投影。

### M3d：trace 投影终态收口

- 将 `task_run_cards*`、`task_run_transcript.go`、`session_push_adapter.go`、
  `trace_compat.go` 迁入 `internal/trace/*`，顶层只保留必要 type alias 或兼容函数。
- `internal/trace/sessiondraft` 拆分 projector，保持 turn timeline 行为不变。
- 所有 stream / session push / run log 事件必须继续透传 `trace_id`，失败直接返回错误，
  不新增静默 fallback。

### M4：兼容层退场与终态压缩

- 外部调用方完成迁移并经过两个里程碑观察期后，删除已无调用的 shim。
- 顶层最终压到 <=15（含测试）；所有目录 <=15；生产文件全部 <=300 行。
- 结构守卫升级到 M4 预算，并在 CI 中阻断顶层回流、目录膨胀、生产文件超长。
- 同步更新生成入口与文档，确保路径迁移后可持续维护。

## 量化门槛

| 里程碑 | 顶层 Go 文件数（含测试） | 非测试 >300 行文件数 | >15 文件目录数 |
| --- | ---: | ---: | ---: |
| 当前工作区 | 92 | 3 | 3 |
| M2.1 | <=70 | 0 | <=3 |
| M3 | <=30 | 0 | <=1 |
| M4 | <=15 | 0 | 0 |

## 验收规则

- 每期必跑：`cd core/bridge && go test ./orchestration/... -timeout 60s`。
- 顶层契约黑盒：action 路由、agent/human resume stream、task scope、session push、
  tool schema、workflow/orchestration run log。
- 域内白盒：dispatch 分发、loop 回合状态机、workflow runner、group orchestration
  runner、policy normalize/validate、trace projection。
- 结构守卫不得通过放宽预算来恢复；只有达到对应里程碑后才更新常量。

## 公共 API / 合约约束

- `ghost-os/bridge/orchestration` 公开方法与核心类型签名保持软兼容。
- `core/bridge/app`、`core/bridge/transport` 不直接 import 新 internal 子包。
- 跨进程 payload 继续遵守 `core/shared/schema.json` 的 request/response 结构。
- `trace_id` 保持全链路可追踪。
- 不新增 fallback、模拟成功、静默降级或隐藏错误的边界规则。
