# CLAUDE.md

This file provides guidance to Claude Code when working in this repository.

## Mission

Ghost-OS is not a traditional remote desktop tool. It is an AI-driven digital twin execution layer. Users should be able to command AI via Web/CLI as if operating their own hands on a remote machine.

## Core Philosophy

1. **Minimalist**: prefer native, lightweight, high-performance Rust/Go implementations.
2. **Bash-First**: default to scriptable operations (Bash/PowerShell/Python); use visual fallback only when GUI cannot be scripted.
3. **Seamless Ecosystem**: browser, CLI, and backend are one coordinated system.
4. **Clean & Aesthetic**: code stays concise and explicit; UX defaults to high-contrast dark style.

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
4. Cross-process payloads must strictly follow [`core/shared/schema.json`](core/shared/schema.json):
   - Request: `{ "action": "string", "params": "object", "trace_id": "string" }`
   - Response: `{ "status": "success|error", "payload": "object", "error": "string" }`

## Build and Dev Commands

```bash
python3 task.py build
python3 task.py ping
python3 task.py init-web
```

Direct commands:

```bash
cd drivers/native && cargo build --release
cd drivers/native && cargo build
cd core/bridge && go run .
```

## Engineering Aesthetics

1. Write only necessary code.
2. Explicit over implicit; avoid over-abstraction.
3. Keep a minimal, functional style.
4. Default dark mode for Web/terminal output.
