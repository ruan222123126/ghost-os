# Open WebUI — 自建 ChatGPT 级 AI 工作台

---

## 🏷 项目名片

| 属性 | 内容 |
|------|------|
| **仓库地址** | [open-webui/open-webui](https://github.com/open-webui/open-webui) |
| **开源协议** | MIT License |
| **主要语言** | Python / Svelte |
| **部署方式** | Docker / 手动安装 |
| **适合人群** | 想自建 ChatGPT 类入口的个人、团队、企业 |

---

## 🎯 一句话定位

> 把 ChatGPT 的体验搬回家，还能接上任何模型——本地的、云端的、开源的、闭源的，统统塞进一个界面里。

---

## 🔥 核心特性

### 1. 多模型统一网关
支持接入 **Ollama**（本地模型）、**OpenAI API**、**Anthropic Claude**、**Google Gemini**、**Azure OpenAI** 等数十种后端，在同一个聊天界面里自由切换模型，甚至可以在对话中途换模型。

### 2. RAG（检索增强生成）
内置 **文档上传 + 向量检索** 能力。把 PDF、Word、Excel、Markdown 等文档扔进去，模型就能基于你的私有知识库回答问题。底层用 ChromaDB 做向量存储，开箱即用。

### 3. 插件生态
拥有类似 "应用商店" 的 **插件市场**（Plugin Marketplace），社区贡献了大量扩展——网页搜索、代码执行、图片生成、语音合成……安装即用。

### 4. 多模态支持
支持 **图片输入**（视觉模型）、**语音输入**（语音转文字）、**图片生成**（DALL·E / Stable Diffusion 集成），覆盖文本之外的交互形态。

### 5. 多用户与权限管理
内置 **角色权限系统**（Admin / User / Pending），适合团队或家庭共享一台机器部署，各自拥有独立的聊天记录和模型配额。

### 6. 界面高度可定制
从主题颜色到聊天布局，从模型参数预设到快捷指令（Prompts），几乎所有 UI 细节都可以配置。

---

## 🏗 技术架构概览

```
┌─────────────────────────────────────────┐
│              Frontend (SvelteKit)        │
│   聊天界面 · 文档管理 · 管理后台           │
└─────────────────┬───────────────────────┘
                  │ REST API / WebSocket
┌─────────────────▼───────────────────────┐
│           Backend (FastAPI / Python)     │
│   用户认证 · 模型路由 · RAG 管道          │
└─────────────────┬───────────────────────┘
                  │
     ┌────────────┼────────────┐
     ▼            ▼            ▼
┌─────────┐ ┌─────────┐ ┌──────────┐
│ Ollama  │ │ OpenAI  │ │ ChromaDB │
│(本地模型)│ │(云端API) │ │(向量存储) │
└─────────┘ └─────────┘ └──────────┘
```

- **前端**：SvelteKit —— 响应快，打包小，开发体验好
- **后端**：FastAPI —— 高性能异步 Python，原生支持 WebSocket 流式输出
- **数据存储**：默认 SQLite（轻量），也支持 PostgreSQL（大规模部署）
- **向量数据库**：ChromaDB，用于 RAG 文档检索

---

## 📦 快速安装（Docker）

```bash
# 最简部署——一条命令跑起来
docker run -d -p 3000:8080 \
  --name open-webui \
  -v open-webui:/app/backend/data \
  ghcr.io/open-webui/open-webui:main
```

如果需要连接本地 Ollama：

```bash
docker run -d -p 3000:8080 \
  --name open-webui \
  --add-host=host.docker.internal:host-gateway \
  -v open-webui:/app/backend/data \
  ghcr.io/open-webui/open-webui:main
```

浏览器打开 `http://localhost:3000`，注册第一个账号即自动成为管理员。

---

## 🎭 适用场景

| 场景 | 匹配度 | 说明 |
|------|--------|------|
| 个人 AI 日常使用 | ⭐⭐⭐⭐⭐ | 完全替代 ChatGPT 网页版 |
| 团队共享 AI 入口 | ⭐⭐⭐⭐⭐ | 多用户 + 权限管理，一份部署全员用 |
| 私有知识库问答 | ⭐⭐⭐⭐⭐ | RAG 能力成熟，文档即知识 |
| 企业内网 AI 网关 | ⭐⭐⭐⭐ | 审计日志、模型路由、统一入口 |
| 移动端使用 | ⭐⭐⭐ | PWA 支持，但无原生 App |

---

## 🔄 与同类工具的对比

| 维度 | Open WebUI | LibreChat | LobeHub |
|------|------------|-----------|---------|
| 部署难度 | ★★☆ (简单) | ★★★ (中等) | ★★☆ (简单) |
| 模型兼容性 | ★★★★★ | ★★★★ | ★★★★ |
| RAG 能力 | ★★★★★ (内置) | ★★★ (需配置) | ★★★★ |
| 插件生态 | ★★★★★ | ★★★ | ★★★★ |
| UI 美观度 | ★★★★ | ★★★ | ★★★★★ |
| 社区活跃度 | ★★★★★ (30k+ Star) | ★★★★ | ★★★★★ |

---

## ⚠️ 注意事项

1. **硬件需求**：仅做网关转发时资源消耗极低（1GB 内存足够）；若开启本地 RAG 则需额外内存给 ChromaDB
2. **Ollama 依赖**：本地模型体验依赖 Ollama 安装和模型下载，后者动辄数 GB 起步
3. **网络与密钥**：连接云端 API 需自行管理 API Key，建议通过环境变量注入
4. **版本更新**：项目迭代极快，docker pull 更新前建议备份 `data` 目录

---

## 📚 延伸资源

- [官方文档](https://docs.openwebui.com/)
- [Discord 社区](https://discord.gg/5rJgQTnV4s)
- [插件市场](https://openwebui.com/plugins/)

---

> **一句话总结**：如果你只能选一个自建 AI 前端，Open WebUI 是目前综合实力最强的选择——生态大、功能全、部署简单。

---

*整理时间：2025年*
