---
项目名称: Open WebUI
类别: 通用 AI 助手 / 工作台
GitHub: https://github.com/open-webui/open-webui
适合人群: 想自建 ChatGPT 类入口的人
标签:
  - AI
  - 自托管
  - WebUI
  - LLM
  - RAG
创建时间: 2025-01-16
---

# 🚀 Open WebUI

## 📝 一句话简介

> 把你的本地或云端大模型，装进一个和 ChatGPT 一模一样（甚至更好用）的网页界面里。

---

## 🎯 项目定位

Open WebUI 是目前 **GitHub 上最活跃的自托管 AI 聊天前端项目之一**（星标增长极快）。它的目标非常明确：让你用 Docker 一行命令就能跑起一个功能完备的 AI 对话入口，后端可以接 Ollama、OpenAI API、甚至任何兼容 OpenAI 格式的模型服务。

---

## ✨ 核心特性

### 1. 多模型无缝切换
- 同时接入 Ollama（本地模型）+ OpenAI API（云端模型）
- 在聊天界面 **一键切换模型**，甚至可以在同一对话中换模型继续聊
- 支持模型分组管理，按场景分配不同模型

### 2. 完整的 RAG 支持
- 内置文档上传 → 自动向量化 → 基于文档回答
- 支持 PDF、TXT、Markdown、CSV 等常见格式
- 可配置 Embedding 模型和向量数据库

### 3. 插件生态
- 社区插件市场，一键安装
- 支持自定义函数调用（Function Calling）
- 可扩展搜索引擎、图片生成等外部能力

### 4. 企业级功能
- 多用户管理 + RBAC 权限控制
- 对话历史持久化，支持搜索和标签
- Web 搜索集成（需配置搜索引擎 API）
- 支持语音输入、Markdown 渲染、代码高亮

### 5. UI 体验
- 几乎 1:1 复刻 ChatGPT 交互，零学习成本
- 支持深色/浅色主题
- 响应式设计，手机端同样好用

---

## 🛠️ 技术栈

| 层级 | 技术选型 |
|------|----------|
| 前端 | SvelteKit + Tailwind CSS |
| 后端 | Python (FastAPI) |
| 数据库 | PostgreSQL / SQLite |
| 向量存储 | ChromaDB |
| 部署 | Docker / Docker Compose |

---

## 🚀 快速上手

```bash
# 一行命令启动（需要先装 Docker）
docker run -d -p 3000:8080 \
  -v open-webui:/app/backend/data \
  -e OLLAMA_BASE_URL=http://host.docker.internal:11434 \
  --name open-webui \
  ghcr.io/open-webui/open-webui:main

# 浏览器打开
# http://localhost:3000
```

### 前置依赖
- 本地安装 [Ollama](https://ollama.com) 并下载至少一个模型（可选，也可以纯用云端 API）
- Docker（推荐）或 Python 3.11+

---

## 📊 项目数据（截至整理时）

- ⭐ Stars: **60k+**（增长极快）
- 🍴 Forks: **6.5k+**
- 📦 社区插件: **200+**
- 👥 贡献者: **500+**

---

## 👍 推荐理由

| 场景 | 适合度 |
|------|--------|
| 个人本地 AI 入口 | ⭐⭐⭐⭐⭐ |
| 小团队共享 AI 服务 | ⭐⭐⭐⭐⭐ |
| 企业私有化部署 | ⭐⭐⭐⭐ |
| 需要 RAG 知识库 | ⭐⭐⭐⭐⭐ |
| 需要插件扩展 | ⭐⭐⭐⭐ |

---

## ⚠️ 注意事项

1. **硬件要求**：如果纯跑本地模型，显存/内存是关键瓶颈，WebUI 本身对资源要求不高
2. **版本更新快**：项目迭代速度极快，建议关注 Release Notes 后再升级
3. **网络依赖**：部分功能（如 Web 搜索、云端 API）需要外网连接
4. **安全配置**：如果暴露到公网，务必开启认证和 HTTPS

---

## 🔗 相关链接

- 官方文档：https://docs.openwebui.com
- Discord 社区：https://discord.gg/5rJgQTnV4s
- 插件市场：https://openwebui.com

---

## 💡 一句话总结

> **Open WebUI = 开源版 ChatGPT 前端 + Ollama 管理面板 + RAG 知识库，三合一的私有 AI 中枢。**
