---
项目名称: LibreChat
类别: 通用 AI 助手 / 工作台
GitHub: https://github.com/danny-avila/LibreChat
适合人群: 想替代 SaaS 聊天前端的团队
标签:
  - AI
  - 自托管
  - 多模型
  - 团队协作
  - 工具调用
创建时间: 2025-01-16
---

# 💬 LibreChat

## 📝 一句话简介

> 一个界面，聊遍所有主流 AI 模型——OpenAI、Anthropic、Google、开源模型全整合，自带工具调用和团队协作能力。

---

## 🎯 项目定位

LibreChat 的定位非常精准：**做开源世界里最成熟的"多模型统一聊天前端"**。它不是简单地套壳 ChatGPT，而是从架构层面设计了一套 **AI 提供商抽象层**，让你像换 SIM 卡一样自由切换底层的 AI 服务，同时保留了对话的完整上下文和工具能力。

如果说 Open WebUI 更偏向 Ollama/本地模型生态，那 LibreChat 的优势在于 **对商业 API 的支持深度** 和 **生产环境的成熟度**。

---

## ✨ 核心特性

### 1. 多提供商统一接入
支持的 AI 服务商包括：
- **OpenAI**（GPT-4o / GPT-4 / GPT-3.5 全系列）
- **Anthropic**（Claude 3.5 Sonnet / Opus / Haiku）
- **Google**（Gemini 全系列）
- **Azure OpenAI**
- **Ollama / Llama.cpp**（本地开源模型）
- **Groq / Together AI / Perplexity** 等新兴厂商
- 任何兼容 OpenAI API 格式的自定义端点

### 2. 工具调用与 Agent
- 完整的 **Function Calling / Tool Use** 支持
- 内置代码解释器（Code Interpreter）
- AI 可以自主执行 SQL 查询、生成图表
- 插件体系支持自定义工具扩展

### 3. 企业协作功能
- 预设提示词（Presets）：管理员统一配置，团队成员共享
- 对话分享：生成只读链接，对外分享单次对话
- 多语言界面（含中文）
- 用户管理 + 权限分级

### 4. 高级对话特性
- 分支对话（类似 ChatGPT 的"编辑消息后重新生成"）
- 对话搜索（支持语义搜索历史对话）
- 导入/导出对话（JSON 格式）
- 语音输入（Whisper API 集成）
- 多模态支持（图片理解，取决于后端模型）

---

## 🛠️ 技术栈

| 层级 | 技术选型 |
|------|----------|
| 前端 | React + TypeScript |
| 后端 | Node.js (Express) |
| 数据库 | MongoDB |
| 缓存 | Redis（可选，推荐） |
| 搜索引擎 | Meilisearch（对话搜索） |
| 部署 | Docker / Docker Compose / 手动部署 |

---

## 🚀 快速上手

```bash
# 1. 克隆仓库
git clone https://github.com/danny-avila/LibreChat.git
cd LibreChat

# 2. 复制配置文件
cp .env.example .env
# 编辑 .env，填入你的 API Key

# 3. Docker 启动
docker compose up -d

# 4. 访问
# http://localhost:3080
```

### 最小配置
只需要在 `.env` 中填入 **至少一个** AI 提供商的 API Key，即可使用。推荐先配置 OpenAI API Key 来快速体验。

---

## 📊 项目数据（截至整理时）

- ⭐ Stars: **21k+**
- 🍴 Forks: **3.5k+**
- 👥 贡献者: **200+**
- 📅 首次发布: 2023 年

---

## 👍 推荐理由

| 场景 | 适合度 |
|------|--------|
| 团队统一 AI 入口 | ⭐⭐⭐⭐⭐ |
| 多模型对比使用 | ⭐⭐⭐⭐⭐ |
| 需要代码解释器 | ⭐⭐⭐⭐ |
| 纯本地离线使用 | ⭐⭐⭐ |
| 简单个人用 | ⭐⭐⭐ |

---

## ⚠️ 注意事项

1. **MongoDB 依赖**：相比 SQLite，MongoDB 的运维成本略高，但 Docker Compose 一键部署已解决此问题
2. **API 成本**：LibreChat 本身免费，但调用的商业 API 仍需付费
3. **资源占用**：Node.js + MongoDB + Meilisearch 整体内存占用约 1-2GB
4. **本地模型体验**：Ollama 集成可用但不如 Open WebUI 那样原生深度整合

---

## 🔗 相关链接

- 官方文档：https://docs.librechat.ai
- Discord 社区：https://discord.librechat.ai
- Demo 演示站：https://demo.librechat.ai

---

## 💡 一句话总结

> **LibreChat = 多模型统一聊天网关 + 团队协作层 + 可编程 Agent，是面向生产力场景最成熟的开源方案。**
