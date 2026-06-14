# Ghost-OS Mobile

Tauri Mobile 版 Android 客户端。React/TypeScript 负责移动端界面，Rust Tauri 宿主负责本地命令与 Bridge HTTP 访问。

## 结构

```text
src/                 # React mobile UI
src-tauri/           # Rust Tauri host
src-tauri/gen/android # Tauri 生成的 Android 工程
```

## 开发

```bash
pnpm install
pnpm tauri android init
pnpm tauri android dev
```

## 构建

```bash
pnpm build
pnpm tauri android build
```

Bridge 通信统一走 `/api/bus`，请求体保持 `{ "action": "...", "params": {}, "trace_id": "..." }`。
