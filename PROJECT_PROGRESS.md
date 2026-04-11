# Ghost-OS 项目进展（摘要版）

更新日期：2026-04-11
当前阶段：MVP 稳定化（主链路可用，持续收口）
文档规则：本文件为硬上限摘要，`总行数不得超过 30 行`，超出时必须先压缩再提交。

## 总览
- 架构三层稳定：Execution（`drivers/native`）、Central（`core/bridge`）、Perception（`apps/web`、`apps/cli`、`apps/android`）。
- 主开发中心是 `core/bridge`；目标是稳定主链路、收紧边界、提升可测试性。
- Assistant 文本工具调用基线为 Tool-Tag：`<t:ID>JSON</t>` + `[TOOL_TAG_RESULT]`；仓库级 AGENTS 已补充“可并行即优先多代理”协作规则。

## 分层状态
- Central：会话、工具调度、SSE、配置、记忆增强、任务调度主线可用；shared schema 与 `trace_id` 契约已统一，`service_router/service_usecase_agent` 已完成超 300 行拆分收口；transport 已将 config/task/rss 直连入口收敛到 `DispatchAction`，并为 `TASK_LIST` 增加可选 `scope=user|system`；`pro/plan` 模式规则已从 orchestration 迁出到 `bridge/mode` 独立包；本次移除 `orchestration/transport` 配置 shim 透传层，统一直连 `bridge/config` 与 `orchestration` 导出契约，并完成 `session/storage` 锁与事务 helper 抽取以压平 `Load/LoadPage/Save/Delete` 的错误处理链路，同时完成 `bridge/config` 领域类型与默认值按域分文件重排（同包、无 API 破坏）；另已将 RSS 报告 Agent/Tool 组装迁至 orchestration，`bridge/rss` 去除对 `runtime/agent` 反向依赖并通过注入保持报告能力；并已完成 `service_usecase_agent/task_workflow_validation/tasks/types/task_usecase_runner` 与 `agent_contract/session_turn_state/run_registry/session_end_signal/service_usecase_human/session_turn_preparer_human_resume` 十处复杂函数热点拆分，`core/bridge` 非测试函数复杂度 `>10` 从 38 降至 28、`>15` 从 1 降至 0，函数长度 `>=50` 从 19 降至 15；并将 `bridge/rss/actions.go`、`bridge/skills/action_handler.go`、`bridge/config/tool_prompt_files.go` 拆分为同包多文件，分别下沉参数解析、系统任务执行、路径安全与工具提示词文件同步职责；并进一步去除 orchestration 内 runtime/rss/skills shim re-export，改为直连 `bridge/runtime`、`bridge/rss`、`bridge/skills`（仅保留 RSS 报告注入适配函数）；并移除 transport/rss 跨层 re-export shim 与 orchestration RSS 构造器转发，transport 测试改为直连 `bridge/rss`；并修复 `Service.SetRSSInbox` 忽略入参的误导性注入行为，改为显式复用 `SetRSSInboxService` 注入路径；并完成 `runtime/agent_runtime_factory` 按配置加载、工具装配、记忆装配拆分为同包多文件，主文件降至 179 行且构建函数拆小。
- Perception：Web/CLI 可用且已接通流式与用户图片输入；Android 已接入会话与展示基础但成熟度较低，`ConfigPanel` 已完成超 300 行拆分；Web chat 已完成 `hooks/chat` 流程收敛：SSE 事件投影下沉到 `lib/chatRuntime`（projector/runtime/toolTagProjection），`useChatState` 改为 reducer 驱动并补齐事件投影与状态收敛单测；并收敛 `lib/api/agent|sessions/parser` 的协议解析双轨：去除字段白名单冗余、枚举改为类型绑定，修复 `mode=plan` 与 `AgentErrorPayload.code` 漂移点。
- Execution：已支持截图、输入、脚本、窗口查询等原子能力；`CODEX_CLI` 现恢复经由 native `CODEX_CLI_START/STATUS` 原子 action 承载执行（bridge 仅保留命令映射与会话索引，不再直接 `exec.Command`），仍未达到生产完备；本次已补齐 native 关键路径测试（`input`/`sandbox`/`screen`/`framing`，新增 `sandbox/diff_engine` 独立测试），并将 CI Native 检查升级为默认 + `python-sandbox` 双矩阵（均 `timeout 60s cargo test`）。

## 当前可用能力
- 会话：创建、流式输出、持久化、恢复、分页查看历史与草稿态展示。
- 工具：`script_exec`、`screen_action`、`computer_use`、`memory_manage`、`rss_fetch`、`web_search` 等可运行；`tfind` 已支持 `kind=tool|skill`、会话级 `dynamic_skill_loads` 与 skill 依赖工具自动装载；`core/bridge/tools` 已完成根包与 `web/memory/screen/graphql/contracts` 子包分层，外部构造器 API 保持兼容。
- 浏览器工具：`browser_control` 与 native `BROWSER_LAUNCH` 已下线，相关调用会显式报错（`tool not found` / `unsupported action`）。
- 任务：`agent_message` 与 `workflow` 基线可用，支持创建、查询、调度与手动执行。
- Agent：新增后端 `mode=plan`（`AGENT_SEND`），仅产出任务编排文本并显式禁止工具/任务执行；`mode` 显式优先于消息内 `pro/prox` 前缀。
- Web 配置：General 页面已去除底板卡片，改为线性分区样式（分隔线 + 标题层级）区分不同配置域。
- Web 配置：Tasks 分页 workflow 编辑器画布已按最新 Architecture React 稿 1:1 还原（侧栏/节点/属性抽屉），保存策略升级为实时自动保存（800ms 防抖、失败可重试、状态可见）并保留 `Ctrl/Cmd+S` 立即保存，同时补齐本地草稿缓存（含节点坐标）以保证刷新后画布布局可恢复；节点拖拽改为 `requestAnimationFrame` 节流并在释放时 flush，同时禁用节点 `transform` 过渡，修复拖动滑移；左侧节点库图标已移除，收起态改为在原图标位显示首字母并与折叠按钮/返回图标中轴对齐；workflow 左侧折叠按钮已切换为主页面同款 panel-left SVG 图标，新增右上齿轮悬浮窗用于配置 schedule 与 session 导入，start 变量卡片支持点击全局 portal 弹窗编辑（name/type/required/value/description），并将 name/value 语义拆分（name 为 string 且 max 100，type 仅约束 value）；背景板已改为 canvas 网格，支持滚轮缩放、拖拽平移与 `Shift+滚轮` 平移，右侧属性面板关闭后不再占画布宽度；修复 create 基线已标记 saved 时 `Ctrl/Cmd+S` 被跳过的问题，手动保存改为 force flush；start 变量值输入现按类型即时约束（boolean 下拉、number 数字输入、object/array 实时校验并失焦格式化，非法时禁用保存），变量名输入改为本地草稿并在失焦/回车时提交，避免逐字触发 autosave；本次进一步将 autosave 控制器改为单实例并将 `Ctrl/Cmd+S` 监听改为 capture + 最新回调 ref，修复状态机在渲染期不稳定导致保存状态长期停留 idle 的问题，并修复保存进行中时手动 Save 提前返回导致“点击无反应”的交互问题；workflow 左下 Save 状态位已改为可点击保存按钮并保留状态色反馈，同时补齐保存失败/校验失败浮层显式提示与按钮按压动画反馈；设置页新增 Skills 分页与卡片化删除管理（`/api/skills`，覆盖 repo/user 两类来源），并新增 Tools 分页（`/api/tools`）：一工具一卡片、仅支持启用/禁用（映射 `tool_blocklist`）与提示词覆盖编辑（文件源为 `~/.ghost-os/prompts/tools/*.md`，并与 `~/.ghost/prompts/tools/*.md` 双向同步，按最新文件更新时间对齐；同时自动迁移旧 `tool_prompt_overrides`）；本次前端 tools 列表改为摘要卡片，提示词编辑需点击卡片进入弹层，不在列表中直接裸露文本域；不支持新增/删除工具。
- Workflow：在 start `inputs` 基础上，后端 workflow 节点已扩展 `if/loop`（含 schema、校验与运行时图遍历执行）；Web 画布已新增 `if/loop` 节点编辑，并完成语义收敛：`loop` 改为固定 `loop_id` 的 `start/end` 成对节点（ID/role/loop_id 只读、删除任一端联动删除、导入后端单 loop 节点时自动展开为 `*-start/*-end`，保存时再编译回后端 `max_iterations + body/exit`），`if` 回归普通单节点条件分支（不再使用 `start/end + if_id` 配对）；`TASK_RUN_NOW` 入参保持不变；本次完成编辑器代码收口（移除组件侧重复 `workflowTaskToDraft`、拆分 Start 变量编辑与画布背景/节点层、start 输入名规则与后端正则对齐），修复前端 `tsc` 的 workflow loop 类型冲突，并新增节点卡片右键菜单（Copy/Delete）与节点复制能力（含 loop 成对复制）。

## 当前约束
- Native、Memory、GUI executor、GraphQL runtime 仍处快速演进区，不按生产级承诺。
- 当前优先级是稳定性、可观测性、契约一致性，不做无边界扩功能。
- 详细变更历史请查 `git log`；本文件仅保留阶段摘要。
