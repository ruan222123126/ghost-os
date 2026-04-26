# Ghost-OS 项目进展（摘要版）
更新日期：2026-04-26
当前阶段：MVP 稳定化（主链路可用，持续收口）
文档规则：仅保留阶段结论，`总行数 <= 30`。

## 总览
- 三层架构稳定：Execution（`drivers/native`）/ Central（`core/bridge`）/ Perception（`apps/web`、`apps/cli`、`apps/android`）。
- 当前主开发中心为 `core/bridge`，目标是稳定性、可观测性、契约一致性。
- Assistant 工具调用基线：`<t:ID>JSON</t>` + `[TOOL_TAG_RESULT]`。
- 协作基线：可并行即优先多代理；跨进程交互统一 `trace_id` 可追踪。

## 分层状态
- Central：会话、工具调度、SSE、任务、配置主链路可用；`ServiceResult` 契约已落地，transport 仅负责 HTTP 映射；`tools/contracts` 已去除对 `session.Session` 的直接依赖；工具执行器已补调用前 `ctx.Done` 快速失败与 tool_call 参数摘要的调试开关守卫；流契约已扩展 `completion_delta.kind=thinking` 并覆盖 codex/anthropic/openai(custom) reasoning 流转，OpenAI 兼容 provider 在工具往返后现会透传 assistant `reasoning_content` 以支持 thinking 模式续跑；复杂函数与超长文件持续拆分中。
- Perception：Web/CLI 已可用并接通流式与用户图片输入；Web 聊天状态和 API 解析链路已收敛并补测，新增 streaming thinking 面板（正文首 token 自动折叠、可手动展开/收起、结束清理）；Android 已接入会话与展示基础，成熟度低于 Web/CLI。
- Execution：截图、输入、脚本、窗口查询等原子能力可用；`CODEX_CLI` 已回归 native 原子 action；Wayland 截图已接入 ScreenCast+PipeWire 持续会话（`OpenPipeWireRemote` + `pipewiresrc` 单帧抓取），减少识别链路反复截图造成的暗屏；bridge/native 启动与首次截图日志现在会显式输出 persistent 配置、session 类型与 capture backend，便于定位“代码已改但实际仍跑在 X11”这类问题；native 关键路径测试与 CI 矩阵已补齐，`task.py` 的 `ping/agent/serve/build` 现在默认以 `python-sandbox` 特性编译 native，避免脚本链路触发 `SCRIPT_EXEC unavailable`，但整体仍未生产完备。

## 当前可用能力
- 会话：创建、流式输出、持久化、恢复、分页历史、草稿态展示。
- 工具：`script_exec`、`screen_action`、`screen_control`、`web_search`、`tfind` 可运行；`script_exec` helper 已恢复并标准化 `tools.search_files(query, path='.', max_results=50)`，用于纯文本检索并返回稳定排序的结构化结果（`path/line/text`）；`web_rooter` 与旧任务管理工具已从 bridge 工具目录、运行时注册、提示引导与文档中移除；`text_input` 已合并进 `screen_control action=text_input` 并从独立工具目录移除；`read_and_summarize` 已从运行时与契约移除；`screen_action` 已为 OCR/icon 链路增加短 TTL 截图复用（`reuse_cache`/`cache_ttl_ms`），降低高频识别场景重复截图带来的闪屏感。
- 任务：`agent_message` 与 `workflow` 基线可用，支持创建、查询、调度、手动执行；Web bridge 已修复无 body POST 透传，任务“运行”按钮不再误报 `invalid JSON body`；运行日志已扩展 `node_results`（节点输入/输出快照、开始/结束时间、并发完成序 `completed_seq` 与 `branch_id`），前端解析已兼容历史 run 的 `node_results` 缺失/`null`。
- Agent：支持后端 `mode=plan`（仅输出编排文本，禁止工具与任务执行）。
- Workflow：`if/loop` 运行时与画布编辑可用；工具节点按 `input_schema` 固定参数；支持节点复制与连线编辑；`screen_control` 已收敛为 atomic-only，屏幕编排动作已移除 `ocr_scan`、将 `click_text` 收敛为 `find_text`、并支持 `click` 坐标编辑后直连原子点击；`find_icon` 支持模板上传与即时识别预览，预览请求默认 `max_results=1`，native 匹配在 single-result 路径并行化+提前剪枝，缓解“检测中”长时间卡住；`hover_after_match` 现在会在前端测试与工作流运行时都透传为真实 `MOUSE_MOVE` 悬停，若 native 产物过旧则显式提示重编 `drivers/native`；运行时仍支持 `workflow_template_data_url` 自动落盘并输出 `exists/match_count/primary_match` 便于流程判断；Workflow `screen_control` 现新增 `workflow_steps`（多步顺序执行、失败即停、与 `action/params` 共存时报错），前端编排器已改为 0/1/多步分支同步并支持从任务定义回填步骤队列。
- Web 配置：General / Tasks / Skills / Tools 页面可用；设置导航已移除“偏好”分组并新增“提示词”分组；提示词设置已拆分为“库 / 预览”双页（兼容 `settings=prompts` 自动落到库页），继续复用独立 `/api/prompts/system` 接口：库页已改为提示词卡片列表 + 点击弹层编辑（`name`/`insert_point`/`content`/`active`），同插入点（当前 `core_job`）前后端共同保证最多一个激活，后端持久化 `prompt_library` 并由激活卡片反向编译 `core_prompt`，预览页只读展示完整渲染提示词，且“库 / 预览”已保持与其他设置页一致的左右宽度；`global_template` / `tool_prompt` / `tool_key_spec` 已从 system prompt 存储、API 契约与运行时链路移除，legacy prompt 文件会在加载时自动清理，且已补 UI/Hook/API/parser 测试；Tools 仍只负责每个工具的 prompt 覆盖与启停；会话侧栏“历史分组开关”在开关两种模式下均统一按最近活跃时间降序展示（分区顺序保持不变，分区内忽略拖拽顺序，拖拽仅改变分区归属）；分区右键菜单已支持“新建/重命名/删除”（删除后会话回收到未分类）；会话分区本地持久化已修复初始化阶段的空状态覆盖问题（刷新后不再丢失），并为 `localStorage` 读写异常增加显式错误日志；Runtime 的 GraphQL 设置已收敛为两个真实生效开关（tool runtime/text sanitize），移除 source/policy JSON 录入项；Workflow 工具参数编辑对 `screen_control` 隐藏 `mode` 与顶层 `display_id`，保留必要输入键，`click` 步骤编辑弹窗已支持“获取鼠标位置”取点（轮询当前坐标、Enter 确认、Esc 取消，并回写 `display_id` 以对齐原子点击缩放），相对坐标模式现会在运行时先读取当前鼠标位置再叠加偏移，避免误用屏幕绝对坐标；屏幕编排步骤的追加/移动/删除/编辑现会统一同步到 `screen_control` 参数，删除 `click` 步骤后不再执行遗留 `click_icon`；`find_icon` 步骤编辑弹窗已切换为新版图像配置样式（上传预览 + 动作切换 + 状态在左按钮在右），并补充 `/workflow -> /workflow/new` 重定向入口；Tasks 列表中的工作流摘要已改为显示“总步骤数（排除 start/end）”，不再显示 agent-only 步骤，并新增“日志”按钮与弹窗查看 run 级 `node_results` 明细；会话侧栏设置按钮在收起态保持左下角固定；Skills 缺失用户目录时自动创建 `~/.ghost-os/skills`，且 Skills 列表删除按钮已固定为单行展示；消息区 assistant 文本已支持 Markdown/GFM 渲染与多语言代码块标签展示（轻量无高亮、含纯文本快路径），代码块右上角可直接复制，并新增 `assistant_markdown_enabled` 运行时开关（TOML + Web 设置可控，关闭后纯文本渲染），tool 卡片支持会话 artifact 图片预览；聊天消息流样式已切到黑白极简版（用户黑底右对齐、assistant 标签化输出、thinking/tool 卡片统一可折叠状态条，thinking 在流式阶段固定排在对应输出上方，并在回合结束后以折叠卡片保留，历史同步时按对应 assistant 锚点保位，不再漂移到列表顶部）；Processing Logic（thinking）现已在 Web 端按会话写入本地持久化并在历史加载/同步时回填，解决刷新后丢失；会话侧栏展开宽度已由 `w-72` 调整为等价 `w-76`（`w-[19rem]`），左侧列表横向空间更宽；设置弹层遮罩模糊由 `backdrop-blur-sm` 下调到 `backdrop-blur-[2px]`，并为左右滚动容器补 `overscroll-contain + touch-pan-y` 以缓解上下滚动迟滞。
- Web 配置：提示词库空状态文案已去除内层卡片容器（圆角/边框/背景），仅保留外层设置卡片，避免二层嵌套视觉。

## 当前约束
- Native、GUI executor、GraphQL runtime 仍在快速演进，不按生产级承诺；memory mode 已重新接入（配置开关 + Memory 提示段 + 日记忆文件自动建档），后续以稳定性验证为主。
- RSS `system_action`（`RSS_INBOX_POLL`、`RSS_BRIEFING_BUILD`）已从任务契约与启动编排移除，运行时仅保留用户任务类型。
- 当前优先级：稳定性、可观测性、契约一致性；不做无边界扩功能。
- 详细历史请查 `git log`；本文件仅保留阶段摘要。
