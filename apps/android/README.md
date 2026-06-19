# Ghost-OS Mobile

Tauri Mobile 版 Android 客户端。React/TypeScript 负责移动端界面，Rust Tauri 宿主负责本地命令与 Bridge HTTP 访问。

## 结构

```text
src/                 # React mobile UI
src-tauri/           # Rust Tauri host
src-tauri/gen/android # Tauri 生成的 Android 工程
```

## 开发

默认使用电脑端 Tauri 桌面壳调试移动端界面和连接能力，不需要把 App 安装到手机上：

```bash
pnpm install
pnpm tauri dev
```

这个模式会在电脑上启动一个 Tauri 桌面壳，并注入 `window.__TAURI_INTERNALS__`。因此当前的“导入配对”、`invoke`、凭据保存、HTTP fallback 都能跑；把窗口尺寸调成手机大小后，可以覆盖 UI、导入配置、连接 Bridge、WebRTC 通道等主要调试路径。

需要验证 Android WebView、手机网络、权限或真机交互时，再使用 Android 真机开发模式：

```bash
pnpm tauri android init
pnpm tauri android dev
```

## 构建

```bash
pnpm build
pnpm tauri android build
```

Bridge 通信统一走 `/api/bus`，请求体保持 `{ "action": "...", "params": {}, "trace_id": "..." }`。
