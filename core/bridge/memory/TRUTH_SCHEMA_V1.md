# Truth Schema V1

## Scope

- Week 1 only lands unified schema v1 plus shadow dual-write.
- Read paths remain frozen; `core/bridge/memory/memory_query.go` does not read truth store.
- Graph stays a derived sidecar, not a truth source.
- Warm/Hot remain runtime caches and do not enter truth primary storage.

## Config

`MemoryConfig` adds four internal-only switches:

- `TruthEnabled`: enables truth shadow components.
- `TruthDualWrite`: enables live shadow writes from existing write entrances.
- `TruthBaseDir`: optional override for truth shadow directory.
- `TruthShadowFailOpen`: defaults to fail-open when truth shadow write fails.

When `TruthBaseDir` is empty and truth is enabled, the default path is a sibling of `ColdBaseDir`:

- `ColdBaseDir=/path/to/cold`
- `TruthBaseDir=/path/to/cold-truth-shadow`

## Schema V1

### `MemoryObject`

Unified truth object with four views:

- `raw_evidence`
- `summary`
- `claims`
- `embedding_refs`

Object metadata also carries:

- `schema_version = 1`
- `object_id`
- `object_type`
- `source_refs`
- `created_at`
- `updated_at`
- `confidence`

### `MemoryClaim`

Week 1 stable keys are intentionally narrow:

- `intent_key`
- `anchor_key`
- `entity_id`
- `constraint_type`
- `risk_type`

Claim IDs are deterministic from `object_id + claim type + stable key fields + value`.

### `SourceRef`

Every object/evidence/claim can carry `SourceRef`:

- `session_id`
- `turn_id`
- `trace_id`
- `source_kind`
- `source_id`

This keeps later replay, arbitration and debugging traceable without changing current external APIs.

### `EmbeddingRef`

Week 1 only stores placeholder embedding links:

- compatible with existing `MemoryEntry.EmbeddingID`
- compatible with existing `MarkdownNode.EmbeddingID`
- no vector generation or ANN read path yet

### `TruthWriteResult`

Single write / replay summary with:

- `event_id`
- `object_id`
- `object_type`
- `object_count`
- `claim_count`
- `source_ref_count`
- `occurred_at`

## Shadow Store Layout

Under `TruthBaseDir`:

- `events/*.jsonl`: append-only event log, partitioned by UTC day
- `objects.snapshot.json`: atomic object snapshot
- `claims.snapshot.json`: atomic claim snapshot
- `replay.checkpoint.json`: last replay checkpoint

The event log is the source of truth. Snapshots are rebuildable projections.

## Mapping Rules

### Archive → `evidence.message`

- one archived message becomes one truth object
- stable identity derives from `session_id + message_index`
- no new query path is introduced

### Markdown Node → `semantic.note`

- preserves summary, anchors and `source_ids`
- anchors become stable claims
- existing markdown file remains unchanged

### Decision Memo → `procedure.memo`

- `intent_key`, `constraints`, `validation_checks`, `avoid_patterns`, `tools_used` map into claims
- decision input is kept as supplemental evidence only
- memo/recipe sidecar remains the current live read source for decision recall

## Write Entrances

Week 1 keeps existing write entrances intact and adds truth as a side write only:

- `ColdMemory.Archive`
- `ColdMemory.SaveMarkdownNode`
- `DecisionService.CaptureTurn`

All three continue to persist their current storage first, then write truth shadow objects.

## Replay + Verification

- `TruthWriter.Replay()` rebuilds snapshots from the event log.
- `TruthVerifier.Verify()` compares live dual-write snapshots against replayed state.
- Expected equality target for Week 1 is counts + canonical object/claim content.

## Failure Handling

- Shadow write is fail-open by default.
- Failures increment truth metrics and emit logs.
- Decision path logs must include `trace_id`.
- Old archive/markdown/decision paths must stay successful when shadow write fails.

## Metrics

`MemoryMetrics` now exposes truth shadow counters:

- `truth_events_written`
- `truth_objects_upserted`
- `truth_claims_upserted`
- `truth_errors`
- `truth_replays`

## Week 2 Hook Points

Planner/vector integration intentionally starts from truth store instead of current recall path:

- planner reads should layer on top of replayed `MemoryObject` / `MemoryClaim`
- vector pipelines should attach through `EmbeddingRef` fill-in, not by mutating query contracts
- any future truth arbitration should consume event log + snapshots, not graph sidecar

Until that lands, query/rerank/context injection remain unchanged.
