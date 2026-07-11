# LobeHub — 颜值与 Agent 协作并重的 AI 工作台

---

## 🏷 项目名片

| 属性 | 内容 |
|------|------|
| **仓库地址** | [lobehub/lobehub](https://github.com/lobehub/lobehub) |
| **开源协议** | Apache License 2.0 |
| **主要语言** | TypeScript (Next.js / React) |
| **部署方式** | Docker / Vercel / Railway / 自托管 |
| **适合人群** | 重视产品体验、热衷于 Agent 协作和插件扩展的开发者 |

---

## 🎯 一句话定位

> 它是开源 AI 聊天前端里"长得最好看的那个"——但不止于好看。Agent 市场、插件生态、多模型支持、视觉生成，让它在功能深度上也毫不妥协。

---

## 🔥 核心特性

### 1. 模型广泛支持
对接 **OpenAI**、**Anthropic Claude**、**Google Gemini**、**Mistral**、**Groq**、**DeepSeek**、**Moonshot（月之暗面）**、**智谱 GLM**、**通义千问**、**Ollama 本地模型**等，覆盖国内外主流 API。模型切换只需在下拉菜单点一下。

### 2. Agent 市场（核心差异化）
内置 **Agent Marketplace**，用户可以创建、分享、一键安装各类角色 Agent——客服、翻译、代码审查、周报生成、心理咨询……社区持续贡献。每个 Agent 可以绑定专属的模型、知识库和插件。

### 3. 插件生态
支持 **Function Calling 插件**——天气查询、网页搜索、信息检索、图片生成，通过插件机制让 AI 获得"动手"能力。插件开发规范清晰，社区贡献活跃。

### 4. 多模态能力
- **视觉理解**：上传图片，视觉模型直接解读
- **图片生成**：对话中一键调用 DALL·E / 本地 Stable Diffusion 出图
- **语音输入**：浏览器端语音识别，解放双手

### 5. 知识库（RAG）
上传文档构建私有知识库，让 Agent 基于你的资料回答问题。支持 PDF、Markdown、TXT 等常见格式。

### 6. 主题与个性化
提供 **暗色/亮色/自动** 三种模式，十几种主题色，字体大小、聊天密度均可调节。UI 设计堪比商业 SaaS 产品。

### 7. 国际化
原生支持中、英、日、韩、法、德等十几种语言，中国开发者尤为友好。

### 8. 部署灵活
支持 **Docker 一键部署**、**Vercel 免费托管**、**Railway**、**Zeabur** 等多种方式。十分钟内能从零到上线公网访问。

---

## 🏗 技术架构概览

```
┌──────────────────────────────────────────┐
│        Frontend (Next.js / React 18)       │
│   SSR 首屏优化 · 聊天界面 · Agent 市场      │
│   插件面板 · 知识库管理 · 设置中心          │
└──────────────────┬───────────────────────┘
                   │ REST API + Server Actions
┌──────────────────▼───────────────────────┐
│          Backend (Next.js API Routes)     │
│   模型代理 · 用户认证 · 插件调度           │
│   RAG 管道 · 图片生成 · 文件存储           │
└──────────────────┬───────────────────────┘
                   │
      ┌────────────┼────────────┐
      ▼            ▼            ▼
┌──────────┐ ┌──────────┐ ┌──────────┐
│ OpenAI   │ │ Anthropic│ │  Ollama  │
│ Google   │ │  Groq    │ │ DeepSeek │
│ Moonshot │ │  智谱    │ │ 通义千问  │ …国内外模型
└──────────┘ └──────────┘ └──────────┘
```

- **前端框架**：Next.js 14+ App Router —— React Server Components + 流式渲染，首屏性能优异
- **状态管理**：Zustand —— 轻量、无模板代码
- **UI 组件**：Ant Design 5 —— 企业级组件库的中文生态最优选择
- **数据存储**：PostgreSQL（推荐）+ Drizzle ORM，也支持本地 JSON 文件模式，灵活起步
- **文件存储**：本地 / AWS S3 / Cloudflare R2 可选

---

## 📦 快速安装

### 方式一：Vercel 免费托管（最快）

点击 LobeHub 官方提供的 Deploy Button，授权 GitHub 账号，填入 API Key，三分钟上线公网可访问的专属 AI 入口。

### 方式二：Docker 部署

```bash
docker run -d -p 3210:3210 \
  -e OPENAI_API_KEY=sk-xxxx \
  -e ACCESS_CODE=your_password \
  lobehub/lobe-chat
```

浏览器打开 `http://localhost:3210`，输入访问密码即可使用。

---

## 🎭 适用场景

| 场景 | 匹配度 | 说明 |
|------|--------|------|
| 颜值党首选 AI 前端 | ⭐⭐⭐⭐⭐ | UI 设计在开源项目中属于第一梯队 |
| Agent 爱好者 | ⭐⭐⭐⭐⭐ | Agent 市场 + 插件生态，可玩性极高 |
| 中文用户 | ⭐⭐⭐⭐⭐ | 原生中文界面 + 国产模型支持 |
| 快速上线公网 | ⭐⭐⭐⭐⭐ | Vercel 免费一键部署 |
| 本地纯离线 | ⭐⭐⭐ | 部分功能依赖云端 API，纯离线场景不如 Jan |

---

## 🔄 与同类工具的对比

| 维度 | LobeHub | Open WebUI | LibreChat |
|------|---------|------------|-----------|
| 部署难度 | ★★☆ (Vercel 极简) | ★★☆ | ★★★ |
| UI 美观度 | ★★★★★ (第一梯队) | ★★★★ | ★★★ |
| Agent 生态 | ★★★★★ (内置市场) | ★★★ | ★★★★ |
| Tool Calling | ★★★★ (插件体系) | ★★★ | ★★★★★ |
| 模型兼容性 | ★★★★ | ★★★★★ | ★★★★★ |
| 国际化 | ★★★★★ (十几种语言) | ★★★ | ★★★ |
| 社区活跃度 | ★★★★★ (35k+ Star) | ★★★★★ | ★★★★ |

---

## ⚠️ 注意事项

1. **Vercel 免费额度**：免费计划有带宽和执行时间限制，高频使用建议自建 Docker
2. **API Key 安全**：Vercel 部署时环境变量直接暴露在服务端，但仍建议设置 ACCESS_CODE 作为访问密码
3. **Agent 质量参差**：Agent 市场开放贡献，部分 Agent 质量一般，建议优先使用高下载量的
4. **数据库选择**：本地 JSON 模式适合尝鲜，正式使用建议上 PostgreSQL

---

## 📚 延伸资源

- [官方文档](https://lobehub.com/docs)
- [GitHub Discussions](https://github.com/lobehub/lobehub/discussions)
- [LobeHub 官网](https://lobehub.com/)

---

> **一句话总结**：LobeHub 是"既好看又能打"的代表——Agent 市场、插件生态和一流 UI 设计让它成为最令人愉悦的开源 AI 前端体验。

---

*整理时间：2025年*
