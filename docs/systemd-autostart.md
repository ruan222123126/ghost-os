# Ghost-OS 开机自启动

当前仓库提供的是 `systemd --user` 方案，目标是把 `bridge + web` 作为当前用户的长期服务运行。默认 web 服务使用 `dev` 模式直接跑当前源码；如果你需要 production 模式，可以显式切换。

## 运行方式

- `ghost-os-bridge.service`：启动 `bin/ghost-bridge serve`
- `ghost-os-web.service`：默认启动 `apps/web` 的 `next dev`，也支持显式切换到 production `next start`
- `ghost-os.target`：统一拉起整套服务
- `ghost-os-session-env.desktop`：用户进入图形会话后导入 `DISPLAY/WAYLAND_DISPLAY/XDG_RUNTIME_DIR/DBUS_SESSION_BUS_ADDRESS`，并重启服务以拿到桌面会话环境

## 安装

在仓库根目录执行：

```bash
bash scripts/autostart/install-systemd-user.sh
```

这个脚本会做四件事：

1. 运行 `python3 task.py build`，生成 `bin/ghost-bridge` 和 `bin/native`
2. 运行 `pnpm --dir apps/web install --frozen-lockfile`
3. 安装并启用用户级 `systemd` 单元和桌面自启动文件

如果你已经手动构建过，可以跳过构建阶段：

```bash
bash scripts/autostart/install-systemd-user.sh --skip-build
```

## 配置文件

安装脚本会在 `~/.config/ghost-os/autostart.env` 写入默认值：

```dotenv
GHOST_CONFIG_PATH=/home/your-user/.ghost-os/config.toml
GHOST_BRIDGE_URL=http://127.0.0.1:8080
GHOST_WEB_START_MODE=dev
GHOST_WEB_HOST=127.0.0.1
GHOST_WEB_PORT=3000
```

其中：

- `GHOST_CONFIG_PATH` 指向 bridge 主配置文件
- `GHOST_BRIDGE_URL` 是 web 代理 bridge 的地址
- `GHOST_WEB_START_MODE` 控制 web 用 `dev` 还是 `prod` 模式启动
- `GHOST_WEB_HOST` / `GHOST_WEB_PORT` 控制 web 服务监听地址

Web 现在支持在未设置 `GHOST_API_TOKEN` 时，自动从 `GHOST_CONFIG_PATH` 指向的 `config.toml` 读取 `api_token`，不需要再在 web 和 bridge 各配一份 token。

如果 `ghost-os-bridge.service` 的服务环境没有继承你交互式 shell 里的 Node/nvm `PATH`，可以在 `config.toml` 里显式设置 `codex_cli_path=/abs/path/to/codex`、`node_bin_path=/abs/path/to/node`，或通过 `GHOST_CODEX_CLI_PATH`、`GHOST_NODE_BIN_PATH` 覆盖，避免 `codex_cli` 报 `spawn codex failed` 或 `/usr/bin/env: 'node': No such file or directory`。

如果要切到 production：

```bash
pnpm --dir apps/web build
sed -i 's/^GHOST_WEB_START_MODE=.*/GHOST_WEB_START_MODE=prod/' ~/.config/ghost-os/autostart.env
systemctl --user restart ghost-os-web.service
```

## 常用命令

```bash
systemctl --user status ghost-os.target --no-pager
systemctl --user restart ghost-os-bridge.service
systemctl --user restart ghost-os-web.service
journalctl --user -u ghost-os-bridge.service -f
journalctl --user -u ghost-os-web.service -f
```

## 行为边界

- 这是用户级服务。默认表现是“重启后用户登录即自动启动”。
- 如果你要“机器刚开机、用户未登录也启动用户服务”，需要额外执行 `sudo loginctl enable-linger $USER`。
- `drivers/native` 依赖图形会话环境；因此即便启用了 linger，真正的桌面截图/输入能力仍要等用户图形会话可用后才完整工作。
