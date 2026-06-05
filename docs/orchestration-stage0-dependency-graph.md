# Orchestration Stage 0 Dependency Graph

更新时间：2026-06-06

## 冻结范围

阶段 0 只冻结边界，不迁移业务代码。以下公开契约保持不变：

- `core/shared/schema/defs/*` 的跨进程 payload 契约。
- `orchestration.Service` 公开 facade、公开类型 alias 与兼容入口。
- `orchestration/internal/contracts/bus` 的 action/status 常量。
- HTTP/SSE 与 bus 链路中的 `trace_id` 传播语义。

## 当前量化基线

| 项 | 当前值 | 阶段 0 处理 |
| --- | ---: | --- |
| `core/bridge/orchestration` Go 文件总数 | 258 | 记录基线 |
| 顶层 `orchestration` Go 文件 | 70 | 白名单冻结，禁止新增顶层 Go 文件 |
| `internal/app/workflows` Go 文件 | 21 | 记录为后续拆分压力点 |
| `internal/app/orchestrations` Go 文件 | 19 | 记录为后续拆分压力点 |
| `core/bridge/tasks/types.go` | 387 行 | 记录为后续类型拆分压力点 |

## 直接依赖图

```mermaid
flowchart TD
  Transport["transport / server HTTP-SSE"] --> Facade["orchestration public facade"]
  Facade --> App["internal/app/* use cases"]
  Facade --> Domain["internal/domain/*"]
  Facade --> Ports["internal/ports"]
  Facade --> Adapters["internal/adapters/*"]
  Facade --> Dispatch["internal/dispatch"]
  Facade --> Trace["internal/trace"]
  Facade --> Contracts["internal/contracts/*"]

  Dispatch --> AppTasks["internal/app/tasks"]
  Dispatch --> AppTools["internal/app/tools"]
  Dispatch --> Contracts

  App --> Domain
  App --> Ports
  App --> Trace
  App --> Contracts

  Adapters --> Ports
  Adapters --> App
  Adapters --> Domain

  Trace --> Contracts
  Trace --> SessionDraft["internal/trace/sessiondraft"]
  Trace --> RunCards["internal/trace/runcards"]

  Domain -. frozen legacy concrete imports .-> Agent["bridge/agent"]
  Domain -. frozen legacy concrete imports .-> Config["bridge/config"]
  Domain -. frozen legacy concrete imports .-> Session["bridge/session"]
  Domain -. frozen legacy concrete imports .-> Tasks["bridge/tasks"]
```

## First-party Import Matrix

| Package | Direct first-party imports |
| --- | --- |
| `orchestration` | `agent`, `config`, `llm`, `mode`, `runtime`, `session`, `skills`, `streaming`, `tasks`, `tools`, `internal/adapters/*`, `internal/app/*`, `internal/contracts/*`, `internal/dispatch`, `internal/domain/*`, `internal/ports`, `internal/shared/text`, `internal/trace*` |
| `internal/dispatch` | `internal/app/tasks`, `internal/app/tools`, `internal/contracts/api`, `internal/contracts/bus` |
| `internal/app/agentturn` | `agent`, `config`, `llm`, `streaming`, `tasks`, `internal/contracts/*`, `internal/trace` |
| `internal/app/workflows` | `llm`, `tasks`, `tools`, `internal/domain/workflow`, `internal/shared/*` |
| `internal/app/orchestrations` | `tasks`, `internal/domain/group`, `internal/ports`, `internal/shared/*` |
| `internal/app/{config,prompts,skills,tasks,tools}` | `config`, `llm`, `skills`, `tasks`, `tools`, `internal/contracts/*` |
| `internal/adapters/agent` | `tasks`, `tools`, `internal/adapters/toolregistry`, `internal/app/*`, `internal/domain/group`, `internal/ports` |
| `internal/adapters/toolregistry` | `tools`, `internal/domain/group` |
| `internal/ports` | `agent`, `config`, `llm`, `session`, `streaming`, `tasks`, `tools`, `internal/domain/group` |
| `internal/trace` | `agent`, `session`, `streaming`, `tasks`, `internal/contracts/api`, `internal/trace/sessiondraft` |
| `internal/trace/runcards` | `streaming`, `tasks`, `internal/contracts/api`, `internal/trace` |
| `internal/trace/sessiondraft` | `session`, `streaming` |
| `internal/contracts/api` | `tasks` |
| `internal/contracts/toolschema` | `llm` |
| `internal/domain/sessionturn` | `agent`, `config`, `llm`, `session`, `internal/contracts/api` |
| `internal/domain/task` | `agent`, `llm`, `tasks` |
| `internal/domain/workflow` | `tasks` |
| `internal/domain/group` | `tasks` |

## 冻结守卫

- `core/bridge/orchestration/internal/structure_guard_test.go`
  - `TestOrchestrationTopLevelFileAllowlist`：冻结当前顶层 70 个 Go 文件，新增顶层文件必须迁入目标层或显式修改白名单。
  - `TestDomainConcreteImportFreeze`：冻结 domain 层当前 concrete import 债务，禁止新增 `agent/config/session/tasks/tools` 具体包依赖。
- `scripts/check-layers.sh`
  - 保留现有 apps/driver 与 transport/orchestration 粗粒度禁令。
  - 追加运行上述两个阶段 0 守卫，使用 `-timeout 60s`。

## 当前已知债务

- `internal/domain/*` 仍存在既有 concrete imports：`agent`、`config`、`session`、`tasks`。阶段 0 已冻结，后续迁移需要通过 domain contract 或 ports 消除。
- 顶层 `orchestration` 仍直接编排大量 app/domain/trace/adapters 包，目标态应继续收敛为 facade、alias 与兼容入口。
- `internal/ports` 当前接口仍暴露多个 concrete bridge 类型，后续 ports/contracts 需要继续瘦身。
