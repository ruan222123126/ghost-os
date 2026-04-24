# 后端 Agent 工具能力实测（2026-03-28）

## 测试目标
对当前后端 Agent 的工具调用能力做一次全量探测：每个工具至少调用一次，明确哪些可用、哪些不可用、哪些需要进一步调试。

## 测试环境
- 时间：2026-03-28
- Bridge 实例：`http://127.0.0.1:18080`
- 调用入口：`POST /api/agent`（GraphQL tool runtime）
- Provider：`67`（`codex`，模型 `gpt-5.3-codex`）

为避免会话历史干扰和工具选择收敛导致的误判，本次用临时测试配置启动同代码后端：
- `max_turns = 1`（每次只执行一次 completion，聚焦“是否成功触发并执行工具”）
- `memory_augmentation_enabled = false`
- `tool_allowlist` 临时扩展为全部待测工具
- 会话/任务/RSS/artifact 路径落在 `/tmp/ghost-os-test/*`

说明：在 `max_turns = 1` 下，`/api/agent` 顶层通常返回 `max turns exceeded: 1`，这是测试策略预期；判定依据是 session 内 `role=tool` 的 `tool_result.status`。

## 逐工具结果

| 工具 | 实测结果 | 结论 | 关键信息 |
|---|---|---|---|
| `send_file` | `tool_result.status=error` | 需调试 | `EXPORT_FILE ... path not in allowed directories` |
| `script_exec` | `tool_result.status=success` | 可用 | 脚本输出正常返回 |
| `set_project_root` | `tool_result.status=success` | 可用 | 成功持久化并即时生效 |
| `codex_cli` | `tool_result.status=error` | 需配置 | `codex_cli requires native_persistent=true` |
| `web_search` | `tool_result.status=success` | 可用 | Tavily 查询返回结果 |
| `feed_manage` | `tool_result.status=success` | 可用 | `operation=list` 返回空列表 |
| `rss_fetch` | `tool_result.status=success` | 可用 | 成功抓取 `https://hnrss.org/frontpage` |
| `screen_action` | `tool_result.status=success` | 可用 | `action=screenshot` 成功产出图片 artifact |
| `browser_control` | `tool_result.status=error` | 依赖前置会话 | `browser session "nope" not found`（需先 `connect/launch`） |
| `text_input` | `tool_result.status=success` | 可用 | 返回 `typed=true` |
| `computer_use` | `tool_result.status=error` | 需调试 | `model_completion_failed ... tls: bad record MAC` |
| `task_manage` | `tool_result.status=success` | 可用 | `operation=list` 返回 `[]` |
| `tfind` | `tool_result.status=success` | 可用 | `action=search` 成功执行 |
| `ask_human` | API 返回 `awaiting_human` | 可用 | 返回 `question_id/prompt/options`，进入等待人工状态 |

## 汇总
- 可用（10）：`script_exec`、`set_project_root`、`web_search`、`feed_manage`、`rss_fetch`、`screen_action`、`text_input`、`task_manage`、`tfind`、`ask_human`
- 需配置/前置条件（2）：`codex_cli`、`browser_control`
- 需调试（2）：`send_file`、`computer_use`

## 需要优先调试的问题
1. `send_file` 路径准入问题
- 现象：相对路径和 `/tmp/ghost-os-test/demo.txt` 都报 `path not in allowed directories`。
- 影响：文件回传链路不可用。

2. `computer_use` 上游模型链路不稳定
- 现象：`model_completion_failed`，底层 TLS 报 `bad record MAC`。
- 影响：GUI executor 无法稳定启动。

## 附：原始结果
- 机器可读结果：`/tmp/ghost-os-test/tool_probe_results.ndjson`
- 聚合 JSON：`/tmp/ghost-os-test/tool_probe_results.array.json`
