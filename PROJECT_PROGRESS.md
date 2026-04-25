# Ghost-OS 项目进展（摘要版）

更新日期：2026-04-24
当前阶段：MVP 稳定化（主链路可用，持续收口）
文档规则：仅保留阶段结论，`总行数 <= 30`。

## 总览
- 三层架构稳定：Execution（`drivers/native`）/ Central（`core/bridge`）/ Perception（`apps/web`、`apps/cli`、`apps/android`）。
- 当前主开发中心为 `core/bridge`，目标是稳定性、可观测性、契约一致性。
- Assistant 工具调用基线：`<t:ID>JSON</t>` + `[TOOL_TAG_RESULT]`。
- 协作基线：可并行即优先多代理；跨进程交互统一 `trace_id` 可追踪。

## 分层状态
- Central：会话、工具调度、SSE、任务、配置主链路可用；`ServiceResult` 契约已落地，transport 仅负责 HTTP 映射；`tools/contracts` 已去除对 `session.Session` 的直接依赖；工具执行器已补调用前 `ctx.Done` 快速失败与 tool_call 参数摘要的调试开关守卫；复杂函数与超长文件持续拆分中。
- Perception：Web/CLI 已可用并接通流式与用户图片输入；Web 聊天状态和 API 解析链路已收敛并补测；Android 已接入会话与展示基础，成熟度低于 Web/CLI。
- Execution：截图、输入、脚本、窗口查询等原子能力可用；`CODEX_CLI` 已回归 native 原子 action；Wayland 截图已接入 ScreenCast+PipeWire 持续会话（`OpenPipeWireRemote` + `pipewiresrc` 单帧抓取），减少识别链路反复截图造成的暗屏；bridge/native 启动与首次截图日志现在会显式输出 persistent 配置、session 类型与 capture backend，便于定位“代码已改但实际仍跑在 X11”这类问题；native 关键路径测试与 CI 矩阵已补齐，`task.py` 的 `ping/agent/serve/build` 已补 bridge/native 产物对齐，但整体仍未生产完备。

## 当前可用能力
- 会话：创建、流式输出、持久化、恢复、分页历史、草稿态展示。
- 工具：`script_exec`、`screen_action`、`screen_control`、`web_search`、`image_generate`、`tfind` 可运行；`text_input` 已合并进 `screen_control action=text_input` 并从独立工具目录移除；`read_and_summarize` 已从运行时与契约移除；`screen_action` 已为 OCR/icon 链路增加短 TTL 截图复用（`reuse_cache`/`cache_ttl_ms`），降低高频识别场景重复截图带来的闪屏感。
- 任务：`agent_message` 与 `workflow` 基线可用，支持创建、查询、调度、手动执行；Web bridge 已修复无 body POST 透传，任务“运行”按钮不再误报 `invalid JSON body`。
- Agent：支持后端 `mode=plan`（仅输出编排文本，禁止工具与任务执行）。
- Workflow：`if/loop` 运行时与画布编辑可用；工具节点按 `input_schema` 固定参数；支持节点复制与连线编辑；`screen_control` 已收敛为 atomic-only，屏幕编排动作已移除 `ocr_scan`、将 `click_text` 收敛为 `find_text`、并支持 `click` 坐标编辑后直连原子点击；`find_icon` 支持模板上传与即时识别预览，预览请求默认 `max_results=1`，native 匹配在 single-result 路径并行化+提前剪枝，缓解“检测中”长时间卡住；`hover_after_match` 现在会在前端测试与工作流运行时都透传为真实 `MOUSE_MOVE` 悬停，若 native 产物过旧则显式提示重编 `drivers/native`；运行时仍支持 `workflow_template_data_url` 自动落盘并输出 `exists/match_count/primary_match` 便于流程判断。
- Web 配置：General / Tasks / Skills / Tools 页面可用；Prompts 页面与独立 `/api/prompts/system` 接口已落地，当前前端仅展示全局模板、核心提示词与完整渲染预览，已补 UI/Hook/API/parser 测试；Tools 仍只负责每个工具的 prompt 覆盖与启停；会话侧栏新增“历史分组开关”，关闭后按最近活跃时间降序单列表展示；分区右键菜单已支持“新建/重命名/删除”（删除后会话回收到未分类）；会话分区本地持久化已修复初始化阶段的空状态覆盖问题（刷新后不再丢失），并为 `localStorage` 读写异常增加显式错误日志；Runtime 的 GraphQL 设置已收敛为两个真实生效开关（tool runtime/text sanitize），移除 source/policy JSON 录入项；Workflow 工具参数编辑对 `screen_control` 隐藏 `mode` 与顶层 `display_id`，保留必要输入键，`click` 步骤编辑弹窗已支持“获取鼠标位置”取点（轮询当前坐标、Enter 确认、Esc 取消，并回写 `display_id` 以对齐原子点击缩放），相对坐标模式现会在运行时先读取当前鼠标位置再叠加偏移，避免误用屏幕绝对坐标；`find_icon` 步骤编辑弹窗已切换为新版图像配置样式（上传预览 + 动作切换 + 状态在左按钮在右），并补充 `/workflow -> /workflow/new` 重定向入口；会话侧栏设置按钮在收起态保持左下角固定；Skills 缺失用户目录时自动创建 `~/.ghost-os/skills`；消息区 assistant 文本已支持 Markdown/GFM 渲染与多语言代码块标签展示（轻量无高亮、含纯文本快路径），tool 卡片支持会话 artifact 图片预览。

## 当前约束
- Native、GUI executor、GraphQL runtime 仍在快速演进，不按生产级承诺；memory augmentation 与 memory tool 链路已从 bridge/runtime/orchestration 移除。
- RSS `system_action`（`RSS_INBOX_POLL`、`RSS_BRIEFING_BUILD`）已从任务契约与启动编排移除，运行时仅保留用户任务类型。
- 当前优先级：稳定性、可观测性、契约一致性；不做无边界扩功能。
- 详细历史请查 `git log`；本文件仅保留阶段摘要。
