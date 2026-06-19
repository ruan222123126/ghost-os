# Ghost-OS Mobile

Tauri Mobile 版 Android 客户端。React/TypeScript 负责移动端界面，Rust Tauri 宿主负责本地命令与 Bridge HTTP 访问。

## 结构

```text
src/                 # React mobile UI
src-tauri/           # Rust Tauri host
src-tauri/gen/android # Tauri 生成的 Android 工程
```

## 开发

首次准备 Android 工程时：

```bash
pnpm install
pnpm tauri android init
```

之后默认启动 Android 真机开发模式：

```bash
pnpm dev
```

`pnpm dev` 会调用 `tauri android dev`，并由 Tauri 自动启动 Vite 前端服务。

如果只需要启动前端 Vite 服务用于页面调试：

```bash
pnpm frontend:dev
```

## 构建

```bash
pnpm build
pnpm tauri android build
```

Bridge 通信统一走 `/api/bus`，请求体保持 `{ "action": "...", "params": {}, "trace_id": "..." }`。
