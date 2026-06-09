# AGENTS.md

Repository-wide rules for coding agents in Ghost-OS.


## Mission

- Ghost-OS is an AI-driven digital twin execution layer, not a traditional remote desktop.
- Optimize for stability, observability, and contract consistency over feature sprawl.

## Communication

- Default to Chinese in user-facing replies unless the user asks otherwise.
- Understand intent with minimal prompting, state low-risk assumptions briefly, and finish the task end-to-end.
- For multi-step work, send a short preamble before tool calls.
- Do not add unrelated features or post-answer enhancement suggestions.

## Debug First

- Surface failures explicitly with errors, logs, and tests.
- Do not add silent fallbacks, fake-success paths, hidden caps, or defensive branches just to keep code running.
- If a safeguard is truly required, make it explicit, documented, easy to disable, and user-approved.

## Architecture Boundaries

- `drivers/native`: stateless atomic execution only; no business decisions.
- `core/bridge`: state, protocol routing, orchestration, and safety; no concrete OS syscalls.
- `apps/web`, `apps/cli`, `apps/android`: perception and interaction only.
- No cross-layer direct coupling.

## Contracts

- Components communicate through the standardized message bus.
- Every operation must remain traceable with `trace_id`.
- Cross-process payloads must follow `core/shared/schema.json`:
  - Request: `{ "action": "string", "params": "object", "trace_id": "string" }`
  - Response: `{ "status": "success|error", "payload": "object", "error": "string" }`
- Assistant tool-calling baseline remains `<t:ID>JSON</t>` with `[TOOL_TAG_RESULT]`; do not reintroduce legacy plain `mutation/query` calls.

## Change Strategy

- Prefer root-cause fixes over symptom patches.
- Remove duplicate logic, dead code, and obsolete compatibility paths unless compatibility is explicitly required.
- Keep changes explicit, minimal, and structural when contracts or shared invariants are involved.
- Do not revert or overwrite user changes outside your task scope.

## Engineering Baseline

- Follow SOLID, DRY, separation of concerns, and YAGNI.
- Prefer short functions, shallow nesting, clear names, and immutable data flow.
- Inject dependencies instead of hard-wiring concrete implementations.
- Add concise comments only for non-obvious logic boundaries or contracts.
- Web uses `pnpm` only; keep `pnpm-lock.yaml`, never add `package-lock.json` or `yarn.lock`.

## Validation

- Run targeted verification whenever feasible: changed tests, then type/lint, then build, then minimal smoke.
- Backend unit tests must use a hard timeout of `60s`.
- If validation cannot run, say why and state the next best check.
- Before finalizing, scan the diff for hidden fallbacks, duplicated logic, dead code, contract drift, and security regressions.

## Collaboration

- Prefer parallel agents only for independent, bounded subtasks with clear ownership.
- Treat unrelated worktree changes as intentional; adapt around them instead of reverting them.
