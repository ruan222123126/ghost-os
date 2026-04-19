# AGENTS.md

This file defines repository-wide execution rules for coding agents.

## Current Project Status

See `PROJECT_PROGRESS.md` before implementation. Current baseline is MVP stabilization:
- Bridge main flow is usable and remains the primary development center (`core/bridge`).
- Web Console and CLI are both usable; Web streaming and user image input are connected (`apps/web`, `apps/cli`).
- Android has initial session/display integration but is still behind Web/CLI maturity (`apps/android`).
- Native atomic capabilities cover screenshot/input/script/window-query, but are still not production-complete (`drivers/native`).
- Assistant text tool-calling protocol baseline is `<t:ID>JSON</t>` with `[TOOL_TAG_RESULT]`; legacy plain `mutation/query` text calls are no longer active.

## Mission

Ghost-OS is not a traditional remote desktop tool. It is an AI-driven digital twin execution layer. Users should be able to command AI via Web/CLI as if operating their own hands on a remote machine.

## Core Philosophy

1. **Minimalist**: prefer native, lightweight, high-performance Rust/Go implementations.
2. **Bash-First**: default to scriptable operations (Bash/PowerShell/Python); use visual fallback only when GUI cannot be scripted.
3. **Seamless Ecosystem**: browser, CLI, and backend are one coordinated system.
4. **Clean & Aesthetic**: code stays concise and explicit; UX defaults to high-contrast dark style.

## Package Management

1. Web app package manager is **pnpm** (`apps/web`).
2. Use `pnpm install` and `pnpm run <script>` for Web dependency and script operations.
3. Commit `pnpm-lock.yaml`; do not introduce `package-lock.json` or `yarn.lock`.

## The Trinity (Strict Boundaries)

1. **Execution Layer** (`drivers/native`, Rust)
   - Stateless and atomic.
   - Handles screenshot, input simulation, window tree/query.
   - Must not contain business decisions.

2. **Central Layer** (`core/bridge`, Go)
   - Manages state, protocol routing, AI orchestration, safety checks.
   - Must not implement concrete OS system calls.

3. **Perception Layer** (`apps/web`, `apps/cli`, `apps/android`)
   - Interaction and feedback.
   - Web rendering, browser structure access, immersive CLI control.

## Decision Priority

Always choose implementation path in this order:
1. **Level 1 (Scripting)**: solve with local scripts first.
2. **Level 2 (API/Native)**: use native system/browser APIs.
3. **Level 3 (Vision)**: screenshot-recognize-click as last resort only.

## Agent Collaboration Strategy

1. If a task can be split into independent and bounded subtasks, prefer multi-agent parallel execution.
2. Assign clear ownership per agent (module/file scope) and keep traceability for each subtask output.

## Communication and Contract Rules

1. Components communicate through a standardized message bus.
2. No cross-layer direct coupling.
3. Every operation must be traceable (trace log with `trace_id`).
4. Cross-process payloads must strictly follow `core/shared/schema.json`:
   - Request: `{ "action": "string", "params": "object", "trace_id": "string" }`
   - Response: `{ "status": "success|error", "payload": "object", "error": "string" }`
5. Assistant text tool-calls must use the Tool-Tag protocol:
   - Tool call text: `<t:ID>JSON</t>`
   - Internal feedback tag: `[TOOL_TAG_RESULT]`

## Engineering Aesthetics

1. Write only necessary code.
2. Explicit over implicit; avoid over-abstraction.
3. Keep a minimal, functional style.
4. Default dark mode for Web/terminal output.

## Commenting Guidelines

1. Add concise comments at key logic boundaries, non-obvious decisions, and cross-module contracts.
2. Do not write line-by-line comments for obvious code.
3. Keep comments short and readable so they improve clarity without adding noise.

## Progress Tracking

1. `PROJECT_PROGRESS.md` at repository root is the canonical progress log for active work.
2. Before starting implementation, read `PROJECT_PROGRESS.md` to align with the latest status and pending tasks.
3. After completing meaningful changes, update `PROJECT_PROGRESS.md` with concise progress notes.
