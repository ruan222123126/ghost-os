# Ghost-OS Progress Snapshot

更新日期：2026-02-28  
分支：main（与 origin/main 同步）  
定位：MVP 骨架阶段（Bridge/Native 最小链路已通，Web Console MVP 可用）

## 1) 三层进度总览（Trinity）

| 层 | 目标 | 当前状态 | 完成度 |
|---|---|---|---|
| Execution (`drivers/native`, Rust) | 原子执行能力（截图/输入/窗口查询） | 最小链路已通，核心原子动作待补齐 | 15% |
| Central (`core/bridge`, Go) | 状态编排、LLM 路由、工具调用 | 主流程可用，具备基础安全与可观测性 | 60% |
| Perception (`apps/web`, `apps/cli`) | Web/CLI 交互层 | Web Console MVP 可用，CLI 已完成基础收敛 | 35% |

## 2) 已完成（概览）

- 三层职责边界与通信契约已稳定，跨层请求可追踪。
- Bridge 主链路（CLI/HTTP/Agent/Provider/Tool）已可用。
- Native 与 Bridge 最小闭环已打通，可完成基础连通与执行验证。
- `SCRIPT_EXEC` 已成为统一工具执行入口，并具备基础沙盒与安全护栏。
- Web Console MVP 已可用，CLI MVP 已完成基础能力收敛。
- CI 与工程基线已建立，核心测试与构建流程可运行。

## 3) 当前主要短板

- Native 原子能力（截图、输入模拟、窗口树/查询）尚未进入可用阶段。
- CLI 交互体验与高级能力仍处于待建设状态。
- 整体仍是 MVP 骨架，距离生产级稳定性与完备安全策略有明显差距。

## 4) 下一步重点

- 补齐 Native 原子执行能力，完成可用级截图/输入/窗口查询。
- 持续强化安全策略与资源隔离，提升系统稳态可靠性。
- 完善 Web/CLI 体验层，推进从 MVP 到可交付版本。
