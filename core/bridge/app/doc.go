// Package app 是 Bridge 的启动与装配入口。
//
// 这里仅负责 CLI 启动、命令分发，以及把独立的 transport/runtime
// 等子包装配成可运行进程；具体的配置、会话编排和任务逻辑
// 已下沉到各自职责包中。
package app
