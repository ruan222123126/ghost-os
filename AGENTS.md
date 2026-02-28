# AGENTS.md

This file defines repository-wide execution rules for coding agents.

## Current Project Status

See `PROJECT_PROGRESS.md` before implementation. Current baseline is MVP skeleton:
- Bridge main flow is usable (`core/bridge`).
- Web Console MVP is usable (`apps/web`).
- CLI baseline is in place (`apps/cli`).
- Native atomic capabilities in `drivers/native` are still incomplete and should not be assumed production-ready.

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

3. **Perception Layer** (`apps/web`, `apps/cli`)
   - Interaction and feedback.
   - Web rendering, browser structure access, immersive CLI control.

## Decision Priority

Always choose implementation path in this order:
1. **Level 1 (Scripting)**: solve with local scripts first.
2. **Level 2 (API/Native)**: use native system/browser APIs.
3. **Level 3 (Vision)**: screenshot-recognize-click as last resort only.

## Communication and Contract Rules

1. Components communicate through a standardized message bus.
2. No cross-layer direct coupling.
3. Every operation must be traceable (trace log with `trace_id`).
4. Cross-process payloads must strictly follow `core/shared/schema.json`:
   - Request: `{ "action": "string", "params": "object", "trace_id": "string" }`
   - Response: `{ "status": "success|error", "payload": "object", "error": "string" }`

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
