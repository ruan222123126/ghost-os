# Orchestration Map

更新时间：2026-06-06

## 范围

本盘点覆盖当前工作区的 `core/bridge/orchestration` 及阶段 0 明确要求的相关目录。
阶段 0 只冻结边界，不搬迁业务代码，不修改公开契约。

## 当前基线

| 项 | 当前值 | 阶段 0 门槛 |
| --- | ---: | --- |
| `core/bridge/orchestration` Go 文件总数 | 287 | 记录基线 |
| 顶层 `core/bridge/orchestration` Go 文件 | 29 | M3 <=30，最终 <=15 |
| 顶层生产 Go 文件 | 11 | 记录基线 |
| `internal/app/workflows` 顶层 Go 文件 | 15 | 目标 <=15 |
| `internal/app/orchestrations` 顶层 Go 文件 | 15 | 目标 <=15 |
| `core/bridge/tasks/types.go` 行数 | 3 | 记录基线 |
| 非测试生产文件超过 300 行 | 0 | 保持 0 |
| 超过 15 个 Go 文件的目录 | 1 | 只允许顶层 `orchestration` 暂时超过 |

## 当前超过目录目标的目录

| Directory | Go Files | Target |
| --- | ---: | --- |
| `core/bridge/orchestration` | 29 | M3 已满足 <=30；最终收口到 <=15。 |

`internal/app/workflows` 与 `internal/app/orchestrations` 当前均为 15 个 Go 文件，已达到相关目录目标上限；
后续迁入文件必须先拆子目录或继续下沉，不能让这两个目录回流超限。

## 冻结守卫

守卫位于 `core/bridge/orchestration/internal/structure_guard_test.go`，并由
`scripts/check-layers.sh` 调用。

- `TestOrchestrationM2StructureBudget`：冻结当前结构预算，顶层 Go 文件 <=29，生产文件 <=300 行，超过 15 个 Go 文件的目录 <=1。
- `TestOrchestrationTopLevelFileAllowlist`：顶层 Go 文件白名单冻结，禁止新增业务文件继续放顶层。
- `TestOrchestrationTopLevelFileAllowlistBaselineIsExact`：白名单必须与当前文件精确匹配，迁走文件后要删除旧白名单项。
- `TestDomainConcreteImportFreeze`：`orchestration/internal/domain/*` 禁止依赖 `config`、`session`、`agent`、`tools`、`tasks` concrete 包。
- `TestDomainConcreteImportFreezeBaselineIsExact`：domain concrete import 债务白名单必须与现状精确匹配；当前白名单为空。

顶层允许范围仅限 façade、构造入口、兼容 type alias、生成合约和顶层黑盒契约测试。普通实现应进入
`internal/app`、`internal/domain`、`internal/ports` 或 `internal/adapters`。

## 目标边界

顶层 `orchestration` 的终态职责只保留：

- 对外 façade：`Service` 构造、生命周期、公开方法兼容入口。
- 合约类型：外部调用方仍需要引用的 request/response/type alias。
- transport/app usecase 入口：`core/bridge/app` 与 `core/bridge/transport` 只能依赖顶层 façade。
- 生成入口：必须留在顶层的 generated contract 文件，除非生成链路同步迁移。

具体实现继续保持在以下子域：

| Target | Responsibility |
| --- | --- |
| `internal/dispatch` | action router、参数解码、service action handler 注册、task scope dispatch。 |
| `internal/app/agentturn` | agent turn prepare/run/resume/history/relay/plan mode。 |
| `internal/app/workflows` | workflow 图校验、节点执行、screen step、find_icon 协调。 |
| `internal/app/orchestrations` | group orchestration runner、owner/member runner、dispatch executor。 |
| `internal/app/{config,prompts,skills,tasks,tools}` | service usecase 的领域入口，顶层只委托。 |
| `internal/domain/*` | 纯领域规则、归一化、投影；禁止新增 concrete bridge package import。 |
| `internal/trace` | `trace_id`、stream event、session push、checkpoint、run/session projection、run cards。 |

## 验收规则

- 必跑：`bash scripts/check-layers.sh`。
- 必跑：`cd core/bridge && go test ./orchestration/... -timeout 60s`。
- 时间允许：`cd core/bridge && go test ./... -timeout 60s`。
- 结构守卫不得通过放宽预算来恢复；只有达到更严格里程碑后才更新常量。

## 公共 API / 合约约束

- `ghost-os/bridge/orchestration` 公开方法与核心类型签名保持兼容。
- `core/shared/schema/*`、`orchestration.Service`、bus action 与 `trace_id` 传播语义不在阶段 0 修改。
- 跨进程 payload 继续遵守 `core/shared/schema.json` 的 request/response 结构。
- 不新增 fallback、模拟成功、静默降级或隐藏错误的边界规则。
