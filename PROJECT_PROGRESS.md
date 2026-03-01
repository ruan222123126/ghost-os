# Ghost-OS Progress Snapshot

更新日期：2026-03-01  
分支：main（与 origin/main 同步）  
阶段：MVP 骨架（主链路可用，能力仍在补齐）

## 1) 三层状态（Trinity）

| 层 | 当前状态 | 完成度 |
|---|---|---|
| Execution (`drivers/native`) | 原子链路已通；`BROWSER_QUERY` 已有基础能力，但完整 URL 仍较依赖 CDP | 20% |
| Central (`core/bridge`) | Agent/Session/Tool 主流程可用，新增三层记忆（L1/L2/L3）基础能力 | 72% |
| Perception (`apps/web`, `apps/cli`) | Web Console 与 CLI MVP 可用，已接入会话续跑 | 40% |

## 2) 已完成（精简）

- 三层边界与消息契约稳定，链路可追踪（`trace_id`）。
- 已修复 `BASH_EXEC` stub 假阳性语义：`drivers/native` 不再返回 `status=success + [stub]`，改为明确 `status=error`（未实现），并补充回归测试避免上层将“未执行”误判为“执行成功”。
- 已完成 `drivers/native` 入口轻量拆分：`main.rs` 仅保留协议入口与基础原子能力，新增 `action_router.rs`（action 分发）、`browser_query.rs`（浏览器查询流程）、`script_exec.rs`（脚本 worker 生命周期/超时/内存限制），降低 God File 耦合风险且保持行为兼容。
- 已完成 `drivers/native/src/sandbox/tools.rs` 职责收敛：将路径策略（`path_policy.rs`）、统一 diff 补丁引擎（`diff_engine.rs`）、URL/SSRF 与网页抓取（`web_security.rs`）拆分为独立模块，`ToolsProxy` 聚焦工具入口与日志编排，减少单文件耦合并提升复用性。
- 已修复 `validate_script_safety` 子串误判：由简单 `contains` 改为函数调用级检查，避免 `bash_exec(` 被错误命中 `exec(`，并保持对 `eval/exec/open/__import__` 等调用的拦截。
- Bridge 主流程可用：Provider 调用、Tool 路由、会话持久化。
- 新增并打通工具：`web_search`、`browser_action`、`ask_human`。
- `ask_human` 支持端到端闭环：暂停、回答、注入 tool result、续跑。
- 已完成 Agent 核心循环去耦：移除 `loop.go` 对 `ask_human`/`browser_action`/`script_exec` 的工具名硬编码，新增 `tools.ResultInterpreter` 扩展点，由工具侧声明“等待人工/多模态 content”语义；空转熔断升级为通用“连续不可执行 tool_call”护栏。
- 已修复关键稳定性问题：空 `finish_reason` 归一化、空 `script_exec` 循环熔断。
- 修复 OpenAI 兼容 tool_call 空参数误归一化问题：不再把空 `arguments` 静默转为 `{}`，Agent 侧对空参数返回错误 envelope 并阻止工具执行。
- 已新增 MVP 回归脚本 `scripts/e2e/mvp_regression.sh`，并补充关键中文注释。
- 已为当前无注释的核心源码文件补充模块级注释（Go/TS/TSX/Rust），提升跨层阅读与定位效率。
- 已完成 `core/bridge/app` 第二轮可读性增强：为关键函数补充意图与边界注释，统一错误语义说明。
- 已完成 `core/bridge/tools` 第三轮可读性增强：补充执行入口、结果归一化与 artifact 辅助函数注释。
- 已消除 `script_exec` 资源约束双端重复：Bridge 侧改为透传 `timeout_ms/max_memory_mb`（不再默认/裁剪），统一由 Native 执行层落地默认值与上限，降低规则漂移风险。
- 已完成 Session 列表链路职责收敛：`core/bridge/session` 新增 `ListMetadata()`，由存储层统一完成会话元数据聚合，`core/bridge/app` 不再执行 `List+逐个 Load` 组装逻辑，降低 usecase 层存储细节泄漏并减少列表查询开销。
- 新增 `core/bridge/memory` 模块：完成统一记忆接口、L2 Warm（内存+JSON持久化、24h窗口、LRU）与 L3 Cold（按月归档、按需检索）。
- 新增 `MemoryManager`：支持 L1→L2→L3 级联查询、L1→L3 归档、L3→L2 手动提升，并在查询/归档时回写会话记忆元数据。
- 扩展 Bridge Bus Action：新增 `MEMORY_QUERY`、`MEMORY_ARCHIVE`，并完成 schema 与多端 envelope 类型同步生成。
- 重构 `runAgentWithSession`：拆分为会话加载/历史构建/执行/持久化/归档的编排结构，并通过 `newBridgeService` 默认注入共享 `memoryManager`，避免请求级重复构建记忆管理器。
- 修复 `browser_action screenshot` 大 payload 风险：截图不再把 `image_base64` 写入 tool 文本历史，改为落盘 artifact 引用（`path/vision_path/hash/size`）并回传元数据。
- 新增多模态输入链路：`browser_action` 截图结果在 `tool` 语义内携带结构化 image content，OpenAI/Anthropic 适配层支持从本地文件/URL 构建 `image` content block 参与视觉推理。
- 已完成 LLM 图片资源解析职责收敛：`openai.go` 仅保留协议映射，图片本地读取/base64 与大小上限策略统一下沉到 `image_content.go`，并新增超限保护测试。
- 已完成 Web Chat Hook 轻量解耦：`apps/web/hooks/useBridgeChat.ts` 抽离统一 reply 渲染、会话解析与错误处理 helper，收敛 `awaiting_human/assistant` 重复分支并保持行为兼容。

## 3) 主要短板

- `BROWSER_QUERY` 对 CDP 依赖较强，非 CDP 环境信息不完整。
- Web/CLI 的 `ask_human` 交互仍偏基础（多问题队列、提示与回退策略待完善）。
- 记忆层仍是 MVP：尚未接入自动摘要、向量检索、GraphRAG 与时间衰减排序策略。
- 安全隔离与生产级稳定性仍有明显差距。

## 4) 下一步

- 扩展 `BROWSER_QUERY` 跨平台能力（macOS/Windows）并增强浏览器识别策略。
- 在三层记忆上继续补齐：自动摘要管线、向量索引接口、时间衰减与优先级召回策略。
- 强化安全策略、资源隔离与稳态压测。
- 完善 Web/CLI 体验，推进 MVP 向可交付版本演进。
