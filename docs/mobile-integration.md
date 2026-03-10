# 手机内网穿透接入指南（第一阶段）

本阶段目标是让手机端通过 Tailscale 直接访问 `core/bridge`，复用现有 `token` 鉴权、`/api/agent`、`/api/questions/answer` 与 `ask_human` 交互链路，不改动任何业务逻辑。

## 1. 适用范围

- iOS 快捷指令
- iOS 原生 App（Swift / SwiftUI）
- Android 原生 App（Kotlin）
- 同 Tailnet 内的其他移动端客户端

当前方案属于**部署与接入层**，不涉及 Bridge 业务代码改造。

## 2. 前置条件

1. Bridge 所在机器已安装并登录 Tailscale。
2. 手机已加入同一个 Tailnet，或 ACL 已允许访问该设备。
3. 本地已能启动 Bridge：

```bash
go -C core/bridge run . serve
```

4. 已生成足够强的 API Token：

```bash
openssl rand -hex 32
```

建议使用至少 `32` 字节随机值，也就是 `64` 个十六进制字符。

## 3. 启动 Bridge

### 3.1 推荐方式

先获取 Tailscale IPv4：

```bash
tailscale ip -4
```

然后启动 Bridge：

```bash
GHOST_BIND_ADDR=<tailscale-ip>:8080 \
GHOST_API_TOKEN=<your-token> \
go -C core/bridge run . serve
```

例如：

```bash
GHOST_BIND_ADDR=100.101.102.103:8080 \
GHOST_API_TOKEN=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
go -C core/bridge run . serve
```

### 3.2 端口优先级

Bridge 的监听地址优先级如下：

1. `GHOST_BIND_ADDR`
2. `serve` 命令的端口参数
3. 默认回退 `127.0.0.1:8080`

这意味着下面命令可以工作，但最后的 `8080` 实际会被 `GHOST_BIND_ADDR` 覆盖：

```bash
GHOST_BIND_ADDR=100.101.102.103:8080 \
GHOST_API_TOKEN=<your-token> \
go -C core/bridge run . serve 8080
```

推荐直接省略多余端口参数，避免误导。

### 3.3 CORS 说明

原生 App 和 iOS 快捷指令通常**不需要**配置 `GHOST_CORS_ORIGINS`，因为它们不是浏览器跨域场景。

只有在以下情况才需要设置：

- 用移动端浏览器直接调试 Bridge
- 用 WebView 加载一个独立 Web 页面并直连 Bridge

示例：

```bash
GHOST_BIND_ADDR=<tailscale-ip>:8080 \
GHOST_API_TOKEN=<your-token> \
GHOST_CORS_ORIGINS=http://localhost:5173,https://console.example.com \
go -C core/bridge run . serve
```

## 4. 快速连通性验证

先做一次最小可用测试：

```bash
curl -sS -X POST http://<tailscale-ip>:8080/api/agent \
  -H "Content-Type: application/json" \
  -H "X-API-Token: <your-token>" \
  -d '{"message":"Hello","trace_id":"mobile-test-1"}'
```

正常情况下会得到统一 envelope：

```json
{
  "status": "success",
  "payload": {
    "message": "...",
    "session_id": "...",
    "session_ended": false
  },
  "error": ""
}
```

## 5. 认证与通用约定

### 5.1 请求头

Bridge 支持两种鉴权头，二选一即可：

```http
Content-Type: application/json
X-API-Token: <your-token>
```

或：

```http
Content-Type: application/json
Authorization: Bearer <your-token>
```

### 5.2 Trace 约定

建议每次请求都传 `trace_id`，方便排查日志：

- iOS：`ios-<timestamp>`
- Android：`android-<timestamp>`
- Shortcuts：`shortcut-<timestamp>`

Bridge 会优先使用 JSON body 中的 `trace_id`；如果未提供，则会读取 `X-Trace-ID`；仍为空时，服务端自动生成。

### 5.3 返回 envelope

所有 HTTP JSON 响应都遵循统一外层结构：

```json
{
  "status": "success|error",
  "payload": {},
  "error": ""
}
```

注意：

- `status` 是 envelope 层状态，不等于 Agent 回合状态
- `payload.status` 只在 `awaiting_human` 这类业务载荷中出现
- 响应头会回带 `X-Trace-ID`
- 请求体上限是 `1 MiB`

## 6. 核心接口

### 6.1 `POST /api/agent`

用于发送用户消息，或在 `ask_human` 之后继续执行。

请求体：

```json
{
  "message": "Hello",
  "session_id": "optional-session-id",
  "trace_id": "ios-1741234567890"
}
```

普通成功响应通常是 HTTP `200`：

```json
{
  "status": "success",
  "payload": {
    "message": "Hello from Ghost-OS",
    "session_id": "sess-123",
    "session_ended": false
  },
  "error": ""
}
```

如果本轮触发 `ask_human`，返回通常是 HTTP `202`：

```json
{
  "status": "success",
  "payload": {
    "status": "awaiting_human",
    "session_id": "sess-123",
    "question_id": "q-123",
    "prompt": "Please confirm the next step"
  },
  "error": ""
}
```

说明：

- 手机端至少需要关心 `message`、`session_id`、`status`、`question_id`、`prompt`
- `session_ended` 与 `session_end` 属于额外返回字段，移动端第一阶段可先忽略，但建议保留兼容
- 客户端应忽略未知字段，避免后续协议扩展造成崩溃
- `session_id` 为空表示开启新会话；若提供了不存在的 `session_id`，Bridge 会返回 HTTP `404`（提示省略 `session_id` 以开启新会话），客户端应始终以响应中的 `session_id` 为准

### 6.2 `POST /api/questions/answer`

用于提交 `ask_human` 的答案，并由 Bridge 在同一次请求内自动续跑对应 session。

相比旧的“`/api/bus` 提交 `HUMAN_RESPONSE`，再调用一次 `/api/agent` 续跑”双请求编排，移动端优先使用这个高层接口；`/api/bus` 保留给低层兼容场景，不建议新的客户端直接依赖。

请求体：

```json
{
  "session_id": "sess-123",
  "question_id": "q-123",
  "answer": "Continue"
}
```

普通成功响应通常是 HTTP `200`：

```json
{
  "status": "success",
  "payload": {
    "message": "Continuing with your answer",
    "session_id": "sess-123",
    "session_ended": false
  },
  "error": ""
}
```

如果续跑后再次触发 `ask_human`，返回通常是 HTTP `202`，响应载荷形态与 `POST /api/agent` 一致：

```json
{
  "status": "success",
  "payload": {
    "status": "awaiting_human",
    "session_id": "sess-123",
    "question_id": "q-124",
    "prompt": "Please confirm the next step"
  },
  "error": ""
}
```

## 7. 基础交互流程

### 7.1 普通对话

1. `POST /api/agent` 发送文本
2. 读取 `payload.message`
3. 保存 `payload.session_id`
4. 后续多轮继续携带同一个 `session_id`

### 7.2 `ask_human` 续跑

1. `POST /api/agent`
2. 收到 `payload.status = "awaiting_human"`
3. 保存 `session_id`、`question_id`、`prompt`
4. 向用户展示 `prompt`
5. 用户输入答案后，`POST /api/questions/answer`
6. 直接读取新的 assistant 回复；如果再次返回 `awaiting_human`，重复本节流程即可

## 8. Swift 示例

下面示例使用 `URLSession`，适合 iOS 原生 App，也可直接改造成 SwiftUI `ObservableObject`。

```swift
import Foundation

struct APIEnvelope<T: Decodable>: Decodable {
    let status: String
    let payload: T
    let error: String
}

struct AgentPayload: Decodable {
    let message: String?
    let session_id: String
    let status: String?
    let question_id: String?
    let prompt: String?
    let session_ended: Bool?
}

final class BridgeClient {
    let baseURL: URL
    let token: String

    init(baseURL: URL, token: String) {
        self.baseURL = baseURL
        self.token = token
    }

    func sendMessage(message: String, sessionID: String? = nil) async throws -> AgentPayload {
        var body: [String: Any] = [
            "message": message,
            "trace_id": "ios-\(Int(Date().timeIntervalSince1970 * 1000))"
        ]
        if let sessionID, !sessionID.isEmpty {
            body["session_id"] = sessionID
        }
        return try await post(path: "/api/agent", body: body, as: AgentPayload.self)
    }

    func submitHumanResponse(sessionID: String, questionID: String, answer: String) async throws -> AgentPayload {
        let body: [String: Any] = [
            "session_id": sessionID,
            "question_id": questionID,
            "answer": answer
        ]
        return try await post(path: "/api/questions/answer", body: body, as: AgentPayload.self)
    }

    private func post<T: Decodable>(path: String, body: [String: Any], as type: T.Type) async throws -> T {
        let url = URL(string: path, relativeTo: baseURL)!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue(token, forHTTPHeaderField: "X-API-Token")
        request.timeoutInterval = 30
        request.httpBody = try JSONSerialization.data(withJSONObject: body)

        let (data, response) = try await URLSession.shared.data(for: request)
        let http = response as! HTTPURLResponse
        let envelope = try JSONDecoder().decode(APIEnvelope<T>.self, from: data)

        guard 200..<300 ~= http.statusCode else {
            throw NSError(domain: "BridgeHTTP", code: http.statusCode, userInfo: [
                NSLocalizedDescriptionKey: envelope.error
            ])
        }
        guard envelope.status == "success" else {
            throw NSError(domain: "BridgeAPI", code: 1, userInfo: [
                NSLocalizedDescriptionKey: envelope.error
            ])
        }
        return envelope.payload
    }
}
```

`ask_human` 处理逻辑：

```swift
let reply = try await client.sendMessage(message: "Ask me a question")

if reply.status == "awaiting_human",
   let questionID = reply.question_id,
   let prompt = reply.prompt {
    print(prompt)
    let continued = try await client.submitHumanResponse(
        sessionID: reply.session_id,
        questionID: questionID,
        answer: "Continue"
    )
    print(continued.message ?? "")
}
```

## 9. Kotlin 示例

下面示例使用 `OkHttp` 与 `kotlinx.serialization`。

```kotlin
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

@Serializable
data class ApiEnvelope<T>(
    val status: String,
    val payload: T,
    val error: String = ""
)

@Serializable
data class AgentPayload(
    val message: String? = null,
    val session_id: String,
    val status: String? = null,
    val question_id: String? = null,
    val prompt: String? = null,
    val session_ended: Boolean? = null
)

class BridgeClient(
    private val baseUrl: String,
    private val token: String,
    private val http: OkHttpClient = OkHttpClient(),
    private val json: Json = Json {
        ignoreUnknownKeys = true
    }
) {
    private val mediaType = "application/json; charset=utf-8".toMediaType()

    fun sendMessage(message: String, sessionId: String? = null): AgentPayload {
        val body = buildString {
            append("{")
            append("\"message\":${json.encodeToString(String.serializer(), message)}")
            if (!sessionId.isNullOrBlank()) {
                append(",\"session_id\":${json.encodeToString(String.serializer(), sessionId)}")
            }
            append(",\"trace_id\":${json.encodeToString(String.serializer(), "android-${System.currentTimeMillis()}")}")
            append("}")
        }
        return post("/api/agent", body, AgentPayload.serializer())
    }

    fun submitHumanResponse(sessionId: String, questionId: String, answer: String): AgentPayload {
        val body = """
            {
              "session_id": ${json.encodeToString(String.serializer(), sessionId)},
              "question_id": ${json.encodeToString(String.serializer(), questionId)},
              "answer": ${json.encodeToString(String.serializer(), answer)}
            }
        """.trimIndent()
        return post("/api/questions/answer", body, AgentPayload.serializer())
    }

    private fun <T> post(path: String, body: String, serializer: kotlinx.serialization.KSerializer<T>): T {
        val request = Request.Builder()
            .url(baseUrl.trimEnd('/') + path)
            .header("Content-Type", "application/json")
            .header("X-API-Token", token)
            .post(body.toRequestBody(mediaType))
            .build()

        http.newCall(request).execute().use { response ->
            val text = response.body?.string().orEmpty()
            val envelope = json.decodeFromString(ApiEnvelope.serializer(serializer), text)

            if (!response.isSuccessful) {
                error("HTTP ${response.code}: ${envelope.error}")
            }
            if (envelope.status != "success") {
                error("Bridge API error: ${envelope.error}")
            }
            return envelope.payload
        }
    }
}
```

`ask_human` 处理逻辑：

```kotlin
val reply = client.sendMessage("Ask me a question")

if (reply.status == "awaiting_human" && reply.question_id != null && reply.prompt != null) {
    println(reply.prompt)
    val continued = client.submitHumanResponse(reply.session_id, reply.question_id, "Continue")
    println(continued.message)
}
```

## 10. iOS 快捷指令建议

如果先做 `Shortcut MVP`，可以直接使用“获取 URL 内容”：

1. URL：`http://<tailscale-ip>:8080/api/agent`
2. 方法：`POST`
3. Header：`X-API-Token: <your-token>`
4. Header：`Content-Type: application/json`
5. Body：JSON

首次请求：

```json
{
  "message": "Hello",
  "trace_id": "shortcut-1"
}
```

如果返回 `awaiting_human`：

1. 从响应中取出 `session_id`
2. 从响应中取出 `question_id`
3. 弹输入框收集答案
4. 调用 `/api/questions/answer`
5. 直接展示返回的 assistant 回复

## 11. Session 管理建议

移动端至少要持久化以下字段：

- `session_id`
- 当前会话的本地消息列表
- 待回答问题的 `question_id`
- 最近一次请求的 `trace_id`

建议：

1. 一个聊天线程绑定一个 `session_id`
2. 当返回 `session_ended = true` 时，将该会话标记为结束
3. 结束会话后，新建对话时不要复用旧 `session_id`
4. `awaiting_human` 期间锁住输入态，避免同一 session 并发发送
5. iOS 可放 `UserDefaults` / `SwiftData`，Android 可放 `SharedPreferences` / `Room`

## 12. 错误处理与重试策略

### 12.1 常见错误码

- `401 Unauthorized`：Token 缺失或错误
- `403 Forbidden`：`Origin` 不在 CORS 白名单内
- `404 Not Found`：`session_id` 或 `question_id` 无效
- `409 Conflict`：会话已结束，或该 session 当前正在运行
- `413 Payload Too Large`：请求体超过 `1 MiB`
- `500 Internal Server Error`：Bridge 或底层 Provider 执行失败

### 12.2 重试原则

推荐策略：

1. **连接失败**：可安全重试
2. **读超时**：视为未知结果，不要立刻重复发送同一条消息
3. **`/api/agent` 已发出但未确认成功**：优先提示用户“结果未知”，避免重复回合
4. **`/api/questions/answer` 提交答案超时**：先检查本地是否仍处于 pending，再决定是否重试
5. 使用 `trace_id` 关联服务端日志，优先排查真实执行结果

## 13. 安全建议

1. 不要把 `GHOST_API_TOKEN` 留空；留空时鉴权会被关闭
2. 不要把 Bridge 直接暴露到公网
3. 优先只绑定到 Tailscale IP，不要默认监听 `0.0.0.0`
4. 用 Tailscale ACL 限制可访问设备
5. Token 放系统密钥链、加密存储或环境变量，不要硬编码进仓库

## 14. 手工测试流程

### 14.1 基础消息测试

```bash
curl -sS -X POST http://<tailscale-ip>:8080/api/agent \
  -H "Content-Type: application/json" \
  -H "X-API-Token: <your-token>" \
  -d '{"message":"Hello","trace_id":"ios-test-1"}'
```

### 14.2 触发 `ask_human`

```bash
curl -sS -X POST http://<tailscale-ip>:8080/api/agent \
  -H "Content-Type: application/json" \
  -H "X-API-Token: <your-token>" \
  -d '{"message":"Ask me a question","trace_id":"ios-test-2"}'
```

记录返回中的 `session_id` 和 `question_id`。

### 14.3 提交人工回答

```bash
curl -sS -X POST http://<tailscale-ip>:8080/api/questions/answer \
  -H "Content-Type: application/json" \
  -H "X-API-Token: <your-token>" \
  -d '{
    "session_id":"<session-id>",
    "question_id":"<question-id>",
    "answer":"My answer"
  }'
```

### 14.4 再次触发 `ask_human` 时继续回答

如果第 14.3 步返回的仍然是 `awaiting_human`，继续拿新的 `question_id` 调用 `POST /api/questions/answer` 即可，无需额外发送空消息续跑。

## 15. 验收标准

- Bridge 成功绑定到 Tailscale IP
- 手机端能通过 Token 调用 `/api/agent`
- 普通文本消息可以跨请求维持 `session_id`
- `ask_human` 可以提交答案并继续执行
- `trace_id` 能在客户端与服务端日志之间对齐
