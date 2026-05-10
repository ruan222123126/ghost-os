# Native Sandbox 越层审计

审计日期：2026-05-10

## 判定规则

- `KEEP`：保留在 `drivers/native`。职责是原子执行、Python runtime 绑定、进程/文件系统/HTTP 调用适配、stdout 捕获、open shim 等执行层机制。
- `MOVE_TO_BRIDGE`：迁到 `core/bridge`。职责是 allow/block 策略、预算默认值、工具可见性、URL/SSRF 规则、路径规则等业务或安全策略。
- `SPLIT`：拆分职责。Bridge 负责策略与契约决策，Driver 只接收显式参数并执行原子能力，失败直接显式返回。

原任务写“18 个文件”，实际清单去重后是 20 个唯一文件，且 `tool_runtime.rs` 重复列出一次。本文按所有唯一文件审计。`SandboxConfig` 定义在 `drivers/native/src/sandbox/mod.rs`，虽然不在清单中，但后续迁移策略时必须一并处理。

## 文件结论

| 文件 | 标注 | 当前职责 | 越层点 | 建议归属 |
|---|---|---|---|---|
| `drivers/native/src/sandbox/config_test.rs` | `MOVE_TO_BRIDGE` | 测试默认允许路径、媒体目录发现、环境变量路径合并。 | 测试对象是路径白名单与默认配置策略。 | 移到 Bridge 配置解析测试；Driver 不应决定默认业务路径。 |
| `drivers/native/src/sandbox/diff_engine.rs` | `KEEP` | 解析 unified diff 并对单文件内容生成补丁结果。 | 无明显越层；是文件写入原子能力的内部算法。 | 保留在 Driver，供 `APPLY_DIFF` 原子动作复用。 |
| `drivers/native/src/sandbox/diff_engine_test.rs` | `KEEP` | 覆盖 diff 解析、hunk 顺序、上下文匹配、CRLF 等行为。 | 无明显越层。 | 随 `diff_engine.rs` 保留在 Driver。 |
| `drivers/native/src/sandbox/executor_bootstrap.rs` | `KEEP` | 为 Python sandbox 注入顶层 helper、stdout 依赖、`subprocess`/`os` shim。 | helper 名称需和 Bridge 工具说明同步，但本文件本身是 runtime 注入。 | 保留在 Driver；Bridge 只维护公开工具描述与 schema。 |
| `drivers/native/src/sandbox/executor.rs` | `KEEP` | 管理 PyO3 GIL、执行锁、stdout 捕获、工具代理绑定、restricted imports 生命周期。 | 无明显越层。 | 保留在 Driver，作为 `SCRIPT_EXEC` 原子执行器。 |
| `drivers/native/src/sandbox/executor_test.rs` | `SPLIT` | 混合测试 Python 执行、helper、open shim、模块限制和危险脚本拦截。 | 模块 allowlist 与危险模式属于策略，open/subprocess shim 属于执行机制。 | 保留 runtime/shim 测试；策略断言迁到 Bridge 策略契约测试。 |
| `drivers/native/src/sandbox/file_tools/bindings/mod.rs` | `KEEP` | 汇总 PyO3 file tool binding 导出。 | 无明显越层。 | 保留在 Driver，作为 Python binding 入口。 |
| `drivers/native/src/sandbox/file_tools/bindings/read_write_py.rs` | `KEEP` | 把 read/list/write/apply_diff 的 Rust 实现适配为 Python 方法并记录 tool log。 | 日志摘要形态需与 Bridge 展示契约同步，但实现是 binding。 | 保留在 Driver；Bridge 负责展示/摘要消费逻辑。 |
| `drivers/native/src/sandbox/file_tools/bindings/search_py.rs` | `KEEP` | 把 search_files 结果转成 Python dict 列表并记录 tool log。 | 无明显越层。 | 保留在 Driver。 |
| `drivers/native/src/sandbox/file_tools/export.rs` | `SPLIT` | 校验导出源、构造 artifact 目标、复制文件、计算 hash、猜 MIME。 | `sessions/<session_id>/<artifact_id>` 布局和 artifact 命名属于 Bridge 会话/产物策略。 | Bridge 决定 artifact 元数据和目标路径；Driver 保留复制、stat、hash 原子操作。 |
| `drivers/native/src/sandbox/file_tools/mod.rs` | `KEEP` | 组织 file_tools 子模块与 re-export。 | 无明显越层。 | 保留在 Driver。 |
| `drivers/native/src/sandbox/file_tools/search.rs` | `KEEP` | 调用 `rg --json` 做精确文本搜索，解析并排序匹配。 | 依赖当前 path policy；搜索执行本身是原子能力。 | 保留 `rg` 调用与解析；路径准入策略迁出后接收已授权 root。 |
| `drivers/native/src/sandbox/path_policy.rs` | `MOVE_TO_BRIDGE` | canonicalize 路径，并执行读写 allowlist 与 blocked glob 检查。 | 路径 allow/block 是业务与安全策略，不应由 Driver 自行决定。 | 迁到 Bridge 配置/工具层；Driver 只做必要路径解析和文件系统错误暴露。 |
| `drivers/native/src/sandbox/restrictions.rs` | `SPLIT` | 安装 Python import/builtin 限制，并做脚本文本危险调用扫描。 | `allowed_modules`、危险调用列表、脚本长度是策略；PyO3 builtins hook 是执行机制。 | Bridge 产出明确限制配置；Driver 保留 hook 安装/恢复和显式执行失败。 |
| `drivers/native/src/sandbox/restrictions_open_shim.rs` | `KEEP` | 生成 Python `open()` text-only shim、import hook 代码和 blocked builtin 替换。 | 文件访问最终仍走 Driver 工具路径；本文件是 runtime shim。 | 保留在 Driver；Bridge 只决定 open 是否暴露及路径策略。 |
| `drivers/native/src/sandbox/shell_tools.rs` | `SPLIT` | 执行 shell 命令、处理超时、截断 stdout/stderr、序列化结果。 | 默认/最大超时和输出上限是策略；进程执行和 kill 是系统调用。 | Bridge 提供预算；Driver 保留 `bash/cmd` 启动、pipe 读取、超时 kill。 |
| `drivers/native/src/sandbox/tool_runtime.rs` | `SPLIT` | 保存 tool log、截断日志、记录错误、维护每脚本 web 请求计数。 | `max_tool_log_chars` 与 `max_web_requests` 是策略预算。 | Bridge 定义预算并随请求下发；Driver 保留计数执行和日志采集机制。 |
| `drivers/native/src/sandbox/tools.rs` | `KEEP` | 暴露 PyO3 `ToolsProxy` 方法，把 Python helper 路由到 native tool impl。 | 无明显越层。 | 保留在 Driver，作为 Python 与原子工具之间的薄绑定。 |
| `drivers/native/src/sandbox/web_security.rs` | `SPLIT` | 校验 HTTPS URL、阻断私网地址、执行 HTTP GET、限制响应大小并转文本。 | URL scheme、SSRF 地址范围、redirect/user-agent/大小上限都是策略；HTTP 调用是执行。 | Bridge 负责 URL/SSRF/预算策略；Driver 保留受参执行 fetch 与字节读取。 |
| `drivers/native/src/sandbox/web_tools.rs` | `SPLIT` | Python `fetch_webpage` binding，消费 web request budget，调用 web_security。 | 请求次数预算与 URL 安全规则属于 Bridge 策略。 | 保留 PyO3 binding；策略迁出后只执行 Bridge 下发的限制。 |

## 迁移边界

- Bridge 已有 `native_allowed_read_paths` / `native_allowed_write_paths` 配置并通过环境变量传给 native；后续应把这些策略对象收口到 Bridge，不再由 Driver 默认发现 `$HOME`、`/media/<user>` 等业务路径。
- Driver 仍应显式报错，不引入静默 fallback。策略缺失时应返回清晰错误，而不是扩大默认权限。
- 真正迁移时，优先把 `SandboxConfig` 拆成 Bridge policy DTO 与 Driver execution limits，避免继续用一个结构同时表达业务策略和系统调用参数。
