# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Ghost-OS is an AI-driven "digital twin execution layer" — not a remote desktop tool. Users command AI to complete tasks on remote machines via Web/CLI. The project follows a strict three-tier architecture with clear separation of concerns.

## Build Commands

```bash
python task.py build      # Compile Rust drivers + Go bridge
python task.py init-web   # Initialize Next.js web console (first-time setup)
```

Individual builds:
```bash
cd drivers/native && cargo build --release   # Rust execution layer
cd core/bridge && go build -o ../../bin/ghost-bridge  # Go bridge
```

## Architecture: The Trinity

Three layers with strict boundaries — never cross-reference between them:

1. **Execution Layer** (`drivers/native/`, Rust) — Stateless atomic operations: screenshots (`xcap`), input simulation (`enigo`), window queries. No business logic allowed here.

2. **Central Bridge** (`core/bridge/`, Go) — State management, LLM orchestration, protocol dispatch, security review. Never makes direct system calls.

3. **Perception Layer** (`apps/web/` Next.js, `apps/cli/`) — Web UI (dark mode), browser integration, CLI control. Interaction and feedback only.

## IPC Contract

All cross-process communication must follow `core/shared/schema.json`:
- Request: `{ action, params, trace_id }`
- Response: `{ status, payload, error }`
- Supported actions: `BASH_EXEC`, `SCREEN_SHOT`, `MOUSE_CLICK`, `BROWSER_QUERY`

## Agent Constitution (from AGENTS.md)

1. Rust layer must NOT contain business logic — only atomic interfaces
2. Go layer must NOT handle system calls — only forwarding and orchestration
3. All IPC must strictly follow `core/shared/schema.json`
4. Code must stay minimal and functional in style

## AI Decision Priority

When implementing a feature, follow this order:
1. **Scripting first** — write a local script (Bash/Python) to solve it
2. **API/Native** — use system or browser native interfaces
3. **Vision last resort** — screenshot + OCR + simulated clicks only when nothing else works

## Code Style

- Write only necessary code. If 10 lines solve it, don't write 11.
- Explicit over implicit — no over-abstraction.
- All operations must be traceable (logged with trace IDs).
- Dark mode default for all UI and terminal output.

## Tech Stack

| Layer | Language | Key Deps |
|-------|----------|----------|
| Execution | Rust 2024 edition | `xcap`, `enigo`, `tokio`, `serde` |
| Bridge | Go 1.25 | — |
| Web | Next.js + TypeScript + Tailwind | — |
