# TRUTH_SCHEMA_V2

`truth v2` 将 truth shadow 从 “object 附属 claim 列表” 升级为独立的 `claim/evidence/event` 中心层。

## 目标

- event log 继续是 source of truth
- claim 拥有独立 schema、状态与生命周期
- object snapshot 退化为兼容 projection，不再是事实本体
- 旧读路径保持可用，写路径开始产出标准化 `evidence + claim + status` events

## Event Types

- `evidence.message.observed`
- `evidence.markdown.note.observed`
- `evidence.tool.result.observed`
- `claim.asserted`
- `claim.status.changed`
- `claim.validity.changed`
- `claim.retracted`
- `object.projected`
- `recipe.distilled`（预留）

## Claim Shape

`MemoryClaim` v2 关键字段：

- `claim_id`
- `subject`
- `predicate`
- `object`
- `value`
- `datatype`
- `evidence_refs`
- `confidence`
- `valid_from` / `valid_to`
- `status`
- `asserted_at`
- `supersedes`
- `metadata`

兼容字段 `type / intent_key / anchor_key / entity_id / constraint_type / risk_type` 仍保留一版周期，用作 legacy mapper input / frozen query path。

## Arbitration

按 `subject + predicate` 形成竞争域。

- `multi_value`：默认并存
- `single_value`：
  - 时间窗口不重叠时，较新/较强 claim 成为 `active`，旧 claim 记为 `superseded`
  - 时间窗口重叠且强度接近时，双方进入 `conflicted`
  - 新 claim 更弱时，新 claim 记为 `superseded`
- 证据不足时，claim 进入 `unverified`
- 不覆盖删除旧 claim，只变更状态

默认 policy 示例：

- `owner_of` → `single_value`
- `preference.language` → `single_value`
- `uses_tool` → `multi_value`
- `avoids_pattern` → `multi_value`

## Storage

继续复用：

- `events/*.jsonl`
- `objects.snapshot.json`
- `claims.snapshot.json`
- `replay.checkpoint.json`

其中 `claims.snapshot.json` 在 v2 中是 claim 主索引快照，支持 replay 全量重建。

## Compatibility

- `MemoryObject` 仍保留，作为兼容 projection
- `MemoryEntry` 语义不变，仅增加 `explain` 承载能力
- graph 仍是 projector，不是 truth source
- markdown / decision / query 旧读路径保持冻结

## Mapper Coverage

- `Archive` → message evidence object
- `SaveMarkdownNode` → markdown evidence + anchor claims
- `CaptureTurn` → decision evidence + structured claims

## Verification

`TruthVerifier` 现同时比较：

- object snapshot
- claim snapshot
- active claim set
- conflict claim set
- superseded chain 集合

