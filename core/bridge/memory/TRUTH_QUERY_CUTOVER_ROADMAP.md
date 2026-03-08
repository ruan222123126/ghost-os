# TRUTH_QUERY_CUTOVER_ROADMAP

本路线图将 `core/bridge/memory` 的 recall 主路径从“多层并行召回 + legacy 字段硬归并”切到“truth-first 主干 + projection hydration”。

## 当前基线

基于 `2026-03-08` 工作树，主干现状如下：

- `QueryService.QueryResultWithScope()` 只有在 `hybridEnabled` 时才走 truth-enhanced recall，truth 仍是候选层之一，不是唯一主源。
- `queryMarkdownWithPlan()` 仍直接把 `MarkdownNode.SourceIDs` / `RelatedTo` / `Anchors` 投影成 `MemoryEntry`，展示 DTO 仍携带事实归并痕迹。
- `entrySourceRefsForHydration()`、`objectIDFromEntry()`、`recallCandidateKey()` 仍依赖 `source_ids` / `memo_id` / `node_id` 做 hydration 与 dedupe。
- `TruthReader.Query()` 仍主要按 `intent_key` / `entity_id` / `anchor_key` / `constraint_type` 等 legacy 索引命中，再辅以全文打分。
- `DecisionService` 已开始支持 claim overlap 打分，但候选集选择仍依赖 `memoIDsByIntentKey` / `memosByAnchorKey` / `memosByGraphNode`。
- `GraphService` 仍返回 `GraphHit.SourceIDs`，排序仍以 `AnchorKeys` overlap 为主。
- 规划中的 cutover 开关当前尚未存在，需要在本阶段补齐配置与 debug 面。

## Cutover 目标

- 所有 recall 先以 `object_id / claim_ids / evidence_ids` 组织，再决定需要哪些 projection。
- `MemoryEntry` 退化为展示 DTO，不再承担事实归并、反查对象与 lineage 追溯职责。
- `graph` / `decision` / `markdown` 都变为 truth-derived projection，命中必须能解释回哪组 claim/evidence。
- `RelatedTo` / `SourceIDs` / `AnchorKeys` / `GraphRefs` 从“主查询输入与归并依据”降级为兼容输出与历史回填辅助。
- 关闭 legacy fallback 后，同一事实不会因 `source_ids` 差异被多层重复召回。

## 成功标准

- 相同 query 在 cutover 前后，`Entries` 主体语义不回退，top-k 核心对象保持一致或更强。
- `DecisionHits` / `GraphHits` 数量允许变化，但 explain 必须更完整，能给出 claim/evidence 级来源。
- 同一事实在 markdown / decision / graph 同时命中时，只保留一个 primary candidate，其余都作为 projection 附着。
- 主归并键从 `object_id -> memo_id -> node_id -> source_ids -> session+entry` 过渡为 `object_id -> claim-set -> evidence-set -> projection:<type>:<id>`。
- 关闭 legacy fallback 后，核心回归测试仍通过，兼容输出仍可服务现有 API 消费端。

## 目标架构

统一查询链路改为：

1. `TruthReader.QueryPrimary()` 产出 primary truth candidates。
2. `ProjectionHydrator` 基于 object/claim/evidence 补 `markdown` / `decision` / `graph` 视图。
3. `HybridReranker` 在 truth-centered candidates 上 rerank，而不是先分层召回后硬并。
4. `MemoryEntry` 仅承载可读摘要、兼容字段与 explain，不再负责 object hydration。

推荐新增或重构的内部结构：

- `TruthQueryOptions`
  - `Subject`
  - `Predicate`
  - `Object`
  - `Value`
  - `IncludeHistorical`
  - `ActiveOnly`
  - `ExposeConflicts`
  - `Explain`
- `TruthQueryExplain`
  - `matched_claim_ids`
  - `matched_evidence_ids`
  - `conflicted_claim_ids`
  - `projection_sources`
  - `why_matched`
- `ProjectionHit`
  - `projection_type`
  - `projection_id`
  - `object_id`
  - `source_claim_ids`
  - `source_evidence_ids`
  - `summary`
  - `explain`

## 实施顺序

### Phase 1：truth-first candidate pipeline

目标：把 `truth` 从 hybrid 候选层升级为 recall 主干。

- 重构 `core/bridge/memory/memory_query.go` 中的 `QueryResultWithScope()`：默认先跑 truth，再按 projection 需求补 hydration，最后 rerank。
- 将当前 `hybridEnabled && truth.Enabled()` 分支改为 “truth 是 hybrid 的 primary source”，非“额外召回层”。
- 新增 `truth primary` debug 输出，区分：primary candidates、projection hydration、legacy fallback 命中数。
- 为结果增加 `legacy_fallback=true/false`、`projection_only=true/false` 等 explain 标记，便于灰度对比。

交付后要求：

- `TruthHits` 成为 `Entries` 主体的来源，而不是旁路 debug 数据。
- hybrid merge 不再需要先把 markdown/decision/graph 独立 flatten 成同级候选。

### Phase 2：truth query 输入语义升级

目标：把 truth query 从 legacy claim label 命中改为标准 claim schema 命中。

- 在 `core/bridge/memory/truth_query.go` 增加 predicate-first 查询入口，优先按 `subject/predicate/object/value` 匹配。
- 新增 active-only / include-historical 模式，默认对主 recall 使用 active-only。
- 把冲突 claim 从“隐藏实现细节”升级为 explain 与 debug 的一等输出。
- 保留 `intent_key` / `anchor_key` / `entity_id` / `constraint_type` 一版周期，但只作为 legacy query adapter 输入。

建议新增索引：

- `claims_by_predicate`
- `claims_by_subject_predicate`
- `claims_by_value_lookup`
- `projections_by_object_id`
- `projections_by_claim_id`

### Phase 3：truth-centered hybrid merge

目标：主归并按 `object / claim / evidence / projection`，彻底降级 `source_ids`。

- 改造 `core/bridge/memory/memory_query_hybrid_helpers.go`：
  - `entrySourceRefsForHydration()` 仅保留历史对象缺失时的 fallback。
  - `objectIDFromEntry()` 优先从显式 `object_id` 与 claim/evidence lineage 解析。
  - `recallCandidateKey()` 改为 `object:<id>` 或 `projection:<type>:<id>`，不再用 `source_ids` 作为主键。
- 所有候选必须优先携带：
  - `object_id`
  - `source_claim_ids`
  - `source_evidence_ids`
- `MemoryEntry.Metadata["source_ids"]` 只保留兼容输出，不再参与 dedupe/hydration 主链。

交付后要求：

- markdown / decision / graph 不会因不同 legacy source id 被合并失败。
- rerank report 能明确显示 primary candidate 与附属 projections。

### Phase 4：markdown projection cutover

目标：markdown 只做 truth-derived 人类可读摘要视图。

- 调整 `queryMarkdownWithPlan()`：不再“枚举所有 node 后 entryMatchesQuery”，而是根据 truth matches 按 `object_id` / `claim_ids` hydration markdown projection。
- `markdownNodeSourceIDs()` 保留兼容 metadata 输出，但不再作为召回依据。
- `MarkdownNode` 历史节点没有 `source_claim_ids` / `source_evidence_ids` 时，只允许通过 legacy fallback 进入结果，并在 explain 标记 fallback 来源。

建议新增索引：

- `nodes_by_source_claim_id`
- `nodes_by_source_evidence_id`

### Phase 5：decision claim-query cutover

目标：decision 以 claim lineage 为候选集主入口，anchor/graph 仅兼容加权。

- 在 `core/bridge/memory/internal/decision/decision_query.go` 中：
  - 候选 memo 选择改为先查 `memos_by_source_claim_id`。
  - recipe 选择改为先查 `recipes_by_source_claim_id`。
  - runs 选择接入 `runs_by_selected_claim_id`。
- `AnchorKeys` / `GraphNodeRefs` 保留一版周期，但只做兼容打分或 explain。
- 最终 `DecisionHit` explain 必须能回答：命中了哪些 claims、缺哪些 claims、冲突了哪些 claims。

建议新增索引：

- `memos_by_source_claim_id`
- `recipes_by_source_claim_id`
- `runs_by_selected_claim_id`

### Phase 6：graph claim-projection cutover

目标：graph 继续作为 explain 视角，但必须附着在 truth claim 或 truth-derived edge 上。

- `GraphHit` 补齐 `object_id` / `source_claim_ids` / `source_evidence_ids`，`SourceIDs` 降级为兼容字段。
- `scoreGraphAnchor()` 改为 claim overlap / claim alias 投影得分；anchor key 只做历史兼容。
- graph seed 与 traversal 命中优先基于 claim/object lineage，而不是 node anchor lookup。

建议新增索引：

- `edges_by_claim_id`
- `nodes_by_claim_alias`

### Phase 7：关闭 fallback 与清理 legacy

目标：在灰度稳定后删主链上的旧 helper、旧索引、旧字段依赖。

第一批：降级为兼容只读

- `MemoryEntry.RelatedTo`
- `MarkdownNode.RelatedTo`
- `MarkdownNode.SourceIDs`
- `DecisionMemo.AnchorKeys`
- `DecisionMemo.GraphNodeRefs`
- `DecisionRecipe.SourceMemoIDs`
- `DecisionRecipe.GraphRefs`
- `DecisionRecipe.AnchorKeys`
- `GraphHit.SourceIDs`

第二批：从主逻辑移除

- `markdownNodeSourceIDs()` 作为召回依据的调用
- `entry.Metadata["source_ids"]` 作为 hydration 主依据的逻辑
- `memosByAnchorKey` / `memosByGraphNode` 主召回索引
- graph ranking 中的 `AnchorKeys` 主打分逻辑

第三批：物理删除

- legacy struct fields
- normalize/clone helper 中的 legacy 透传分支
- 测试夹具里的 legacy 样本

## 灰度方案

建议新增开关：

- `Truth.QueryPrimaryEnabled`
- `Truth.QueryProjectionHydrationEnabled`
- `Truth.LegacyFallbackEnabled`
- `Decision.ClaimQueryEnabled`
- `Graph.ClaimProjectionEnabled`
- `Markdown.ClaimProjectionEnabled`

灰度顺序：

1. 开 `Truth.QueryPrimaryEnabled`，保留 `Truth.LegacyFallbackEnabled`
2. 开 `Markdown.ClaimProjectionEnabled` 与 `Truth.QueryProjectionHydrationEnabled`
3. 开 `Decision.ClaimQueryEnabled`
4. 开 `Graph.ClaimProjectionEnabled`
5. 关闭 `Truth.LegacyFallbackEnabled`
6. 删除 legacy 索引、helper、字段

## 回归与观测

### 一致性

- 同一 query 在 legacy 与 truth-first 下，`Entries` 数量差异可控。
- top-k 的核心 `object_id` 一致，explain 至少不弱于 legacy。
- 同一事实对象不会因为 `source_ids` 差异重复出现。

### 去重

- 同一事实同时存在 markdown / decision / graph 时，只保留一个 primary candidate。
- 其余 projection 通过 explain 与 metadata 挂接，不另起主条目。

### 冲突

- conflicted claim query 能显式展示 `conflicted_claim_ids`。
- markdown / graph 的历史缓存不会掩盖 active/conflicted 状态。

### 兼容

- 历史 markdown 没有 `source_claim_ids` 时仍可通过 fallback 查到。
- 历史 memo 没有 lineage 时仍能降级召回。
- 所有 fallback 命中必须带 `legacy_fallback=true`。

### 观测

建议补指标：

- `memory_query_truth_primary_total`
- `memory_query_projection_hydration_total`
- `memory_query_legacy_fallback_total`
- `memory_query_duplicate_candidates_total`
- `memory_query_conflicted_claim_hits_total`

## PR 拆分

1. `PR1`：truth-first candidate pipeline
2. `PR2`：markdown query cutover
3. `PR3`：decision claim-query cutover
4. `PR4`：graph claim-projection cutover
5. `PR5`：关闭 legacy fallback，保留兼容输出
6. `PR6`：删除旧索引 / 旧 helper / 旧字段
7. `PR7`：清文档、清测试、更新 `PROJECT_PROGRESS.md`

## 首批落点文件

- `core/bridge/memory/memory_query.go`
- `core/bridge/memory/memory_query_hybrid_helpers.go`
- `core/bridge/memory/truth_query.go`
- `core/bridge/memory/memory_markdown.go`
- `core/bridge/memory/internal/decision/decision_query.go`
- `core/bridge/memory/internal/graph/graph_query.go`
- `core/bridge/memory/memory_types.go`

## 执行备注

- 本阶段不重写 UI 交互，不新增 memory layer，不调整外部 API 大结构。
- 优先完成内部主路径 cutover，再逐步收紧兼容字段。
- `TRUTH_SCHEMA_V2.md` 是 schema 基线；本路线图关注 query/read path 的切换顺序与灰度手段。
