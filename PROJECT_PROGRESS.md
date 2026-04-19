# Ghost-OS 项目进展（摘要版）

更新日期：2026-04-18
当前阶段：MVP 稳定化（主链路可用，持续收口）
文档规则：仅保留阶段结论，`总行数 <= 30`。

## 总览
- 三层架构稳定：Execution（`drivers/native`）/ Central（`core/bridge`）/ Perception（`apps/web`、`apps/cli`、`apps/android`）。
- 当前主开发中心为 `core/bridge`，目标是稳定性、可观测性、契约一致性。
- Assistant 工具调用基线：`<t:ID>JSON</t>` + `[TOOL_TAG_RESULT]`。
- 协作基线：可并行即优先多代理；跨进程交互统一 `trace_id` 可追踪。

## 分层状态
- Central：会话、工具调度、SSE、任务、配置主链路可用；`ServiceResult` 契约已落地，transport 仅负责 HTTP 映射；`tools/contracts` 已去除对 `session.Session` 的直接依赖；复杂函数与超长文件持续拆分中。
- Perception：Web/CLI 已可用并接通流式与用户图片输入；Web 聊天状态和 API 解析链路已收敛并补测；Android 已接入会话与展示基础，成熟度低于 Web/CLI。
- Execution：截图、输入、脚本、窗口查询等原子能力可用；`CODEX_CLI` 已回归 native 原子 action；native 关键路径测试与 CI 矩阵已补齐，但整体仍未生产完备。

## 当前可用能力
- 会话：创建、流式输出、持久化、恢复、分页历史、草稿态展示。
- 工具：`script_exec`、`screen_action`、`memory_manage`、`rss_fetch`、`web_search`、`tfind` 可运行。
- 任务：`agent_message` 与 `workflow` 基线可用，支持创建、查询、调度、手动执行。
- Agent：支持后端 `mode=plan`（仅输出编排文本，禁止工具与任务执行）。
- Workflow：`if/loop` 运行时与画布编辑可用；工具节点按 `input_schema` 固定参数；支持节点复制与连线编辑；`screen_control` 已收敛为 atomic-only，屏幕编排动作已移除 `ocr_scan`、将 `click_text` 收敛为 `find_text`、并支持 `click` 坐标编辑后直连原子点击；`find_icon` 支持模板上传与即时识别预览，预览请求默认 `max_results=1`，native 匹配在 single-result 路径并行化+提前剪枝，缓解“检测中”长时间卡住；前端测试勾选“悬停鼠标”后会透传 `hover_after_match` 并在命中后执行真实 `MOUSE_MOVE` 悬停；运行时仍支持 `workflow_template_data_url` 自动落盘并输出 `exists/match_count/primary_match` 便于流程判断。
- Web 配置：General / Tasks / Skills / Tools 页面可用；Tools 支持启用/禁用与提示词覆盖管理；会话侧栏新增“历史分组开关”，关闭后按最近活跃时间降序单列表展示；分区右键菜单已支持“新建/重命名/删除”（删除后会话回收到未分类）；会话分区本地持久化已修复初始化阶段的空状态覆盖问题（刷新后不再丢失），并为 `localStorage` 读写异常增加显式错误日志；Runtime 的 GraphQL 设置已收敛为两个真实生效开关（tool runtime/text sanitize），移除 source/policy JSON 录入项；Workflow 工具参数编辑对 `screen_control` 隐藏 `mode` 与顶层 `display_id`，保留必要输入键，`click` 步骤编辑弹窗已切换为新版坐标样式（位置基准 + X/Y），`find_icon` 步骤编辑弹窗已切换为新版图像配置样式（上传预览 + 动作切换 + 状态在左按钮在右），并补充 `/workflow -> /workflow/new` 重定向入口；会话侧栏设置按钮在收起态保持左下角固定；Skills 缺失用户目录时自动创建 `~/.ghost-os/skills`；消息区 assistant 文本已支持 Markdown/GFM 渲染与多语言代码块标签展示（轻量无高亮、含纯文本快路径）。

## 当前约束
- Native、Memory、GUI executor、GraphQL runtime 仍在快速演进，不按生产级承诺。
- RSS `system_action`（`RSS_INBOX_POLL`、`RSS_BRIEFING_BUILD`）已从任务契约与启动编排移除，运行时仅保留用户任务类型。
- 当前优先级：稳定性、可观测性、契约一致性；不做无边界扩功能。
- 详细历史请查 `git log`；本文件仅保留阶段摘要。
