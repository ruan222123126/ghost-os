# LibreChat — 多模型统一聊天的自托管替代方案

---

## 🏷 项目名片

| 属性 | 内容 |
|------|------|
| **仓库地址** | [danny-avila/LibreChat](https://github.com/danny-avila/LibreChat) |
| **开源协议** | MIT License |
| **主要语言** | JavaScript (React / Node.js) |
| **部署方式** | Docker / Docker Compose |
| **适合人群** | 想替代 SaaS 聊天前端的团队、需要有工具调用能力的开发者 |

---

## 🎯 一句话定位

> 如果你想要一个能同时对接 OpenAI、Anthropic、Google、Azure 等所有主流模型，还带工具调用（Function Calling）和代码解释器的自建聊天前端——LibreChat 是目前最成熟的开源选择。

---

## 🔥 核心特性

### 1. 全模型统一入口
支持 **OpenAI**（GPT-4o、o1 系列）、**Anthropic Claude**、**Google Gemini**、**Azure OpenAI**、**AWS Bedrock**、**Mistral**、**Groq**、**Ollama** 本地模型、**DeepSeek** 等 20+ 后端。所有模型在一个界面里自由切换，无需反复登录不同平台。

### 2. 工具调用 (Tool Use / Function Calling)
这是 LibreChat 的杀手锏——支持模型的 **Function Calling** 能力。AI 可以在对话中自动调用外部 API、执行计算、查询数据库。例如让 AI 直接查天气、发邮件、操作日历，全部在聊天框内完成。

### 3. 代码解释器 (Code Interpreter)
内置沙箱化的代码执行环境，AI 可以写 Python 代码→运行→返回结果，类似 ChatGPT Code Interpreter 的开源替代。数据分析、图表生成、文件处理一气呵成。

### 4. RAG 与文件系统
上传 PDF、Word、代码文件，AI 能基于文件内容回答问询。支持 **向量检索**（需配合外部向量库），也支持直接在对话中上传临时文件。

### 5. AI 智能体 (Agents)
支持创建自定义 Agent，配置系统提示词、可用工具、模型选择，打造专属的自动化助手。

### 6. 预设消息 / 提示词库
内置可复用的提示词模板系统（Presets），团队可以共享优质 Prompt，一键调用。

### 7. 多模态对话
支持图片理解（视觉模型）、图片生成（DALL·E 集成），语音输入通过浏览器原生 API 实现。

### 8. 多用户与权限
完整的用户注册 / 登录系统，支持 OAuth（Google、GitHub、OpenID），角色分 Admin 和普通用户，适合团队共用。

---

## 🏗 技术架构概览

```
┌──────────────────────────────────────────┐
│          Frontend (React / Vite)          │
│   聊天界面 · Agent 管理 · 管理后台         │
└──────────────────┬───────────────────────┘
                   │ REST API / WebSocket (流式)
┌──────────────────▼───────────────────────┐
│          Backend (Node.js / Express)      │
│   用户认证 · 模型路由 · Tool Calling      │
│   文件处理 · 代码沙箱 · RAG 管道          │
└──────────────────┬───────────────────────┘
                   │
      ┌────────────┼────────────┐
      ▼            ▼            ▼
┌──────────┐ ┌──────────┐ ┌──────────┐
│ OpenAI   │ │ Anthropic│ │  Ollama  │
│ Google   │ │  Azure   │ │  Groq    │
│ Bedrock  │ │  Mistral │ │ DeepSeek │  …20+ 模型后端
└──────────┘ └──────────┘ └──────────┘
```

- **前端**：React + Vite —— 生态成熟，组件丰富，响应式设计
- **后端**：Node.js + Express —— 异步 I/O 天然适配多 API 并发调用，流式响应稳定
- **数据库**：MongoDB（默认）—— 文档型数据库适合存储对话记录、用户配置等半结构化数据
- **沙箱**：代码解释器通过 Docker 容器隔离执行，安全可控

---

## 📦 快速安装（Docker Compose）

```bash
# 1. 克隆仓库
git clone https://github.com/danny-avila/LibreChat.git
cd LibreChat

# 2. 复制并编辑环境变量
cp .env.example .env
# 编辑 .env，填入你要用的 API Key

# 3. 一键启动
docker compose up -d
```

浏览器打开 `http://localhost:3080`，注册即用。

---

## 🎭 适用场景

| 场景 | 匹配度 | 说明 |
|------|--------|------|
| 团队共享 AI 入口 | ⭐⭐⭐⭐⭐ | 多用户管理 + 统一计费，一份部署全员用 |
| 需要 Function Calling | ⭐⭐⭐⭐⭐ | 对工具调用支持最成熟的开源聊天前端 |
| 数据分析 + AI | ⭐⭐⭐⭐⭐ | 内置代码解释器，直接跑 Python |
| 个人日常 AI 使用 | ⭐⭐⭐⭐ | 功能全但部署比 Open WebUI 略重 |
| 移动端使用 | ⭐⭐⭐ | 响应式 Web，无原生 App |

---

## 🔄 与同类工具的对比

| 维度 | LibreChat | Open WebUI | LobeHub |
|------|-----------|------------|---------|
| 部署难度 | ★★★ (需 MongoDB) | ★★☆ (SQLite) | ★★☆ (简单) |
| 模型兼容性 | ★★★★★ | ★★★★★ | ★★★★ |
| Tool Calling | ★★★★★ (核心卖点) | ★★★ (插件实现) | ★★★★ |
| 代码解释器 | ★★★★★ (内置沙箱) | ★★★ (插件) | ★★★ |
| RAG 能力 | ★★★★ | ★★★★★ | ★★★★ |
| UI 美观度 | ★★★ | ★★★★ | ★★★★★ |
| 社区活跃度 | ★★★★ (15k+ Star) | ★★★★★ (30k+ Star) | ★★★★★ |

---

## ⚠️ 注意事项

1. **MongoDB 依赖**：LibreChat 强依赖 MongoDB，部署比单 SQLite 方案多一个服务，内存占用稍高（建议 2GB+）
2. **API 费用**：本身免费，但调用的云端模型 API 仍需付费，建议设置用量上限
3. **代码沙箱安全**：代码解释器默认在 Docker 容器内运行，需注意不要挂载敏感目录
4. **更新频繁**：项目高度活跃，大版本可能涉及数据迁移，升级前务必看 Release Notes

---

## 📚 延伸资源

- [官方文档](https://docs.librechat.ai/)
- [Discord 社区](https://discord.gg/NGaa9RPCft)
- [演示站点](https://librechat-librechat.hf.space/)

---

> **一句话总结**：如果你需要的不仅是聊天，而是让 AI 真正能干活的平台——LibreChat 是目前开源世界里 Tool Calling 和代码执行能力最完整的方案。

---

*整理时间：2025年*
