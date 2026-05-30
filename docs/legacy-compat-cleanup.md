# Legacy Compat Cleanup

本次清理目标：只保留“历史展示兼容”，移除所有运行时 compat、旧参数映射与隐式迁移，统一改为显式 migration 或 fail-fast。

## Bridge
- `prompts_dir` 双根镜像
  现状位置：`core/bridge/config/tool_prompt_files_roots.go`、`system_prompt_files_roots.go`、`preset_storage.go`
  分类：运行时兼容 / 隐式迁移
  新行为：运行时只读单一 canonical root；`serve` 发现 `.ghost/prompts` 或双根并存时直接报错
  迁移命令：`bin/ghost-bridge migrate prompts-dir`

- `tool_prompt_overrides`
  现状位置：`core/bridge/config/config_file.go`
  分类：隐式迁移
  新行为：bridge 启动不再自动写盘迁移；`serve` 检测到该字段直接报错
  迁移命令：`bin/ghost-bridge migrate tool-prompts`

- legacy system prompt 文件 / `core_prompt -> prompt_library` / `memory` cards
  现状位置：`core/bridge/config/system_prompt_files*.go`、`preset_*.go`
  分类：隐式迁移 / 运行时兼容
  新行为：不再启动期自动清理或改写；`serve` 检测到 legacy system files、空 `prompt_library + 非空 core_prompt`、`insert_point=memory`、`preset.prompt_refs.memory` 时直接报错
  迁移命令：`bin/ghost-bridge migrate system-prompts`

- legacy session `*.json -> sqlite`
  现状位置：`core/bridge/session/storage_legacy.go`、`storage_list.go`、`storage_helpers.go`
  分类：隐式迁移
  新行为：`List/Load` 不再偷偷导入并删除 legacy JSON；`serve` 检测到 legacy session 文件直接报错
  迁移命令：`bin/ghost-bridge migrate sessions`

- orchestration `start/end`
  现状位置：`core/bridge/orchestration/internal/domain/group/*`、`core/shared/schema/defs/tasks.json`
  分类：运行时兼容
  新行为：validator 直接报错并提示迁移；schema 已移除 `start/end`
  迁移命令：`bin/ghost-bridge migrate orchestrations`

- `codex_cli.full_auto`
  现状位置：`core/bridge/tools/codex_cli*.go`、`drivers/native/src/codex_cli/actions.rs`
  分类：运行时兼容 / 旧参数映射
  新行为：参数一旦出现立即报错 `full_auto is removed; use sandbox=workspace-write`
  迁移命令：无；调用侧需改参数

## Web
- orchestration legacy localStorage 草稿
  现状位置：`apps/web/lib/orchestration-editor/legacyMigration.ts`、`hooks/config/useOrchestrationSectionState.ts`
  分类：隐式迁移
  新行为：页面加载不再自动导入；设置页显示显式迁移提示条，用户点击后才迁移
  迁移命令：Web UI 按钮

- session sidebar partitions legacy localStorage
  现状位置：`apps/web/hooks/useSessionSidebarPartitions.ts`
  分类：隐式迁移
  新行为：hydrate 不再自动 put/remove；侧栏顶部显示“迁移 / 丢弃”显式操作
  迁移命令：Web UI 按钮

- `settings=prompts`
  现状位置：`apps/web/lib/settingsQuery.ts`
  分类：运行时兼容
  新行为：不再隐式映射到 `prompts_library`，旧 query 直接失效并回到默认
  迁移命令：无；更新旧链接

## Serve Preflight
- 现状位置：`core/bridge/transport/transport_legacy_preflight.go`
- 行为：`bin/ghost-bridge serve` 启动前统一检测 legacy prompts / tool overrides / system prompts / session JSON / orchestration start-end；发现即退出非 0，并打印对应 migration 命令。
