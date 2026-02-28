# Ghost-OS CLI

Minimal Rust terminal client for Ghost-OS bridge (`core/bridge`).

## Build

```bash
cd apps/cli
cargo build --release
```

## Install

Install from repository root:

```bash
cargo install --path apps/cli
```

Or via task runner:

```bash
python3 task.py install-cli
```

## Run (Interactive REPL)

```bash
cd core/bridge
go run . serve
```

In another terminal:

```bash
cd apps/cli
cargo run
```

## Run (One-shot)

```bash
cd apps/cli
cargo run -- --message "Hello, what can you do?"
```

## Arguments

- `--bridge-url <URL>`: Override bridge URL (default: `http://localhost:8080`)
- `--timeout <SECONDS>`: Override HTTP timeout (default: `30`)
- `--message <TEXT>`: Send one message and exit

## Environment Variables

- `GHOST_BRIDGE_URL`: Bridge base URL
- `GHOST_CLI_TIMEOUT`: HTTP timeout in seconds

Precedence: CLI arguments > environment variables > defaults.

## REPL Commands

- `/help`, `/h`
- `/config`
- `/model <name>`
- `/provider <name>`
- `/clear`
- `/literal`, `/l` (显示如何发送字面量 `/...`)
- `/exit`, `/quit`, `/q`

Send a literal slash message to the model with `//text` (example: `//help` sends `/help`).

## CI Check

Minimal verification command:

```bash
cargo check --manifest-path apps/cli/Cargo.toml
```

Or via task runner:

```bash
python3 task.py check-cli
```
