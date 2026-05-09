# Ghost-OS Android Client

最小可用的 Android 客户端，直连 Bridge HTTP 接口。

## 功能

- 配置 Bridge URL 和 API Token
- 测试连接（调用 `/api/config`）
- 发送消息并接收回复（调用 `/api/agent`）
- 订阅同一 `session_id` 的实时推送（SSE）
- 本地持久化配置和会话 ID

## 技术栈

- Kotlin + Jetpack Compose
- OkHttp (网络请求)
- kotlinx.serialization (JSON 序列化)
- DataStore (本地存储)

## 构建

```bash
cd apps/android
./gradlew assembleDebug
```

APK 输出路径：`app/build/outputs/apk/debug/app-debug.apk`

## 安装

```bash
adb install app/build/outputs/apk/debug/app-debug.apk
```

## 使用

1. 打开 App，点击右上角"配置"
2. 填写 Bridge URL（如 `http://192.168.1.100:8080`）
3. 填写 API Token
4. 点击"测试连接"验证配置
5. 如需接收其他端同一会话的消息，可填入对应 `Session ID`
6. 返回聊天页面，发送消息或等待推送

## 限制

- 当前版本支持会话级 SSE 推送，但仅在 App 运行时保持连接
- 支持 `ask_human` 提示推送与回答续跑
- 不支持多会话管理与系统级后台通知
- 需要 Bridge 服务端配置 `GHOST_API_KEY` 才能正常调用 AI 模型

## 代码结构

```
app/src/main/java/dev/ghostos/android/
├── network/BridgeClient.kt       # HTTP 客户端
├── model/*Models.kt              # schema 生成的 API 契约模型
├── store/SettingsStore.kt        # 本地存储
├── viewmodel/ChatViewModel.kt    # 聊天逻辑
├── ui/ChatScreen.kt              # 聊天界面
├── ui/ConfigScreen.kt            # 配置界面
└── MainActivity.kt               # 入口
```
