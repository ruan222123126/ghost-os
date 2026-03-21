# GraphQL Mutation 工具化模式说明（对齐当前实现）

本文用于把“Query 查数据、Mutation 执行动作”的架构思路，收口为与 Ghost-OS 当前实现一致的表述，避免实现方误解“统一接口”发生在哪一层。

## 1. 一句话结论

可以把业务动作统一建模为 GraphQL Mutation，但当前系统里，Agent 仍通过工具调用协议进入执行链路。  
统一的是业务接口范式，不是底层调度协议。

## 2. 调用链路（当前实现）

当 GraphQL 工具入口启用时，执行路径是：

1. Agent 输出结构化工具调用（tool-call）。
2. Bridge 路由到 `graphql_query`、`graphql_schema_lookup` 或 `graphql_mutation`。
3. 工具内部执行 GraphQL 请求，并返回结构化结果给 Agent。

这意味着：

- 不存在“Agent 直接把 GraphQL 字符串原封不动扔给网关并自动执行所有动作”的运行模式。
- Go 网关不是零改动，而是保留一个稳定的 GraphQL 工具适配层；新增业务能力主要通过 schema 与策略配置扩展。
- 当前分支默认运行时未注册 GraphQL 工具入口；本文描述的是该入口启用后的统一接口语义。

## 3. 与传统 Tool Calling 的关系

两者不是互斥关系，而是分层关系：

- 调度层：仍是 Tool Calling（结构化参数、可追踪、可审计）。
- 业务层：通过 GraphQL Query/Mutation 统一读写能力。

这样做的收益：

- AI 层与业务层解耦，Agent 不需要为每个业务动作新增独立工具实现。
- 参数校验和类型约束前移到 GraphQL schema。
- 可复用 source/domain/policy 的统一治理能力。

## 4. 写操作不是“裸 Mutation 直发”

GraphQL 写链路不是“一条 mutation 直接执行完成”，而是显式两阶段：

1. `action=prepare`：校验文档、绑定策略、冻结请求、挂起审批。
2. `action=commit` / `retry_commit`：按 `intent_id` 提交冻结请求。

并配套：

- `action=status`：查询 intent 状态。
- `action=discard`：废弃挂起 intent。
- `action=list_pending`：列出可处理 intent。

## 5. 需要明确的三类挑战

### 5.1 先后依赖

当第二步依赖第一步返回值（例如先创建订单再支付订单），通常需要多轮：

1. 第一轮执行创建并拿到 ID。
2. 第二轮携带 ID 执行后续 mutation。

若业务允许，也可在服务端提供复合 mutation 统一封装。

### 5.2 审批与策略

写入必须命中 allowlist mutation policy，并按流程审批，不建议描述为“模型写一句话就能完成所有执行”。

### 5.3 交付幂等与不确定性

类型系统负责结构正确性，但不能替代网络不确定场景下的交付保障。  
需要通过 `delivery_key`、`request_hash`、`commit_state`、receipt 追踪确保可恢复与可审计。

## 6. 建议对外表述模板

推荐对外统一描述为：

> Ghost-OS 使用“Tool Calling + GraphQL”分层架构：  
> Agent 通过单一 GraphQL 工具入口进行读写；  
> Query 负责只读检索，Mutation 在策略与审批约束下执行写操作。

不推荐对外描述为：

> 网关无需任何适配，Agent 直接提交 GraphQL 语句即可替代工具调用协议。

## 7. 当前契约速览

- 只读：`graphql_query`（仅允许 query）。
- 结构检索：`graphql_schema_lookup`（本地 schema snapshot）。
- 写入：`graphql_mutation`（`prepare/commit/retry_commit/status/discard/list_pending`）。
- 可见性：GraphQL 工具属于按需加载工具面，非静态默认常驻。
- 运行时开关：当前分支默认未向 Agent 注册 GraphQL 工具入口；若重新启用，仍应遵循本文分层与审批语义。
