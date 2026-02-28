# Ghost-OS Progress Snapshot

更新日期：2026-02-28  
分支：main（与 origin/main 同步）  
阶段：MVP 骨架（主链路可用，能力仍在补齐）

## 1) 三层状态（Trinity）

| 层 | 当前状态 | 完成度 |
|---|---|---|
| Execution (`drivers/native`) | 原子链路已通；`BROWSER_QUERY` 已有基础能力，但完整 URL 仍较依赖 CDP | 20% |
| Central (`core/bridge`) | Agent/Session/Tool 主流程可用，支持 `ask_human` 暂停与恢复 | 65% |
| Perception (`apps/web`, `apps/cli`) | Web Console 与 CLI MVP 可用，已接入会话续跑 | 40% |

## 2) 已完成（精简）

- 三层边界与消息契约稳定，链路可追踪（`trace_id`）。
- Bridge 主流程可用：Provider 调用、Tool 路由、会话持久化。
- 新增并打通工具：`web_search`、`browser_action`、`ask_human`。
- `ask_human` 支持端到端闭环：暂停、回答、注入 tool result、续跑。
- 已修复关键稳定性问题：空 `finish_reason` 归一化、空 `script_exec` 循环熔断。
- 已新增 MVP 回归脚本 `scripts/e2e/mvp_regression.sh`，并补充关键中文注释。

## 3) 主要短板

- `BROWSER_QUERY` 对 CDP 依赖较强，非 CDP 环境信息不完整。
- Web/CLI 的 `ask_human` 交互仍偏基础（多问题队列、提示与回退策略待完善）。
- 安全隔离与生产级稳定性仍有明显差距。

## 4) 下一步

- 扩展 `BROWSER_QUERY` 跨平台能力（macOS/Windows）并增强浏览器识别策略。
- 强化安全策略、资源隔离与稳态压测。
- 完善 Web/CLI 体验，推进 MVP 向可交付版本演进。
