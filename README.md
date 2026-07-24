# BookNav Next

自托管个人网址导航 · **Go API + Vue 3**

本仓库是 BookNav 的全新架构实现（前后端分离、单二进制 / Docker 部署）。  
旧版 Flask 实现见：[DEKVIW/book-nav](https://github.com/DEKVIW/book-nav)（已停止功能更新，仅作历史参考）。

---

## 技术栈

| 层 | 选型 |
|----|------|
| 后端 | Go · Chi · SQLite · Session Cookie + CSRF |
| 前端 | Vue 3 · TypeScript · Vite · Pinia · Vue Router |
| 图标 | Lucide |
| 部署 | Docker 多阶段构建 · 静态前端嵌入 |

---

## 仓库结构

```
book-nav-next/
├── apps/
│   ├── server/              # Go API 与静态资源服务
│   │   ├── cmd/server/      # 进程入口
│   │   ├── internal/        # 配置、路由、领域、仓储、任务
│   │   ├── migrations/      # SQL 迁移
│   │   └── webdist/         # 生产构建后的前端产物（运行时挂载/拷贝）
│   └── web/                 # Vue 3 前端
│       ├── public/          # 静态资源（光标、背景等）
│       └── src/
│           ├── app/         # 应用壳、路由、布局
│           ├── modules/     # portal（前台）/ admin（后台）
│           └── shared/      # API 客户端、状态、样式、通用组件
├── deploy/                  # Dockerfile、compose、远程启动脚本
├── scripts/                 # 本地开发与进程守护脚本
├── data/                    # 运行时数据（本地，不入库）
├── .env.example             # 环境变量模板
└── README.md
```

### 目录说明

| 路径 | 职责 |
|------|------|
| `apps/server` | REST API、鉴权、后台任务、SQLite、生产态静态托管 |
| `apps/web` | 前台导航与后台管理 SPA |
| `apps/web/src/modules/portal` | 首页、搜索、登录、跳转页等 |
| `apps/web/src/modules/admin` | 分类 / 网站 / 图标 / 任务 / 备份 / 设置等 |
| `deploy` | 镜像构建与 `docker compose` 生产编排 |
| `scripts` | Windows / Unix 本地启停与看门狗 |
| `data` | 数据库与上传文件目录（`.gitignore`） |

---

## 功能概览

**前台**

- 分类侧栏 + 页内分区滚动、子分类 Tab
- 站点卡片、搜索、精选
- 管理员：粘贴 URL 快加、重复链接处理、右键菜单、拖拽排序
- 登录 / 邀请码注册、外链跳转页
- 私有链接可见性控制

**后台**

- 网站与分类管理、图标管理
- 用户与邀请码、站点设置
- 导出 / 备份、死链检测、图标抓取等异步任务

---

## 本地开发

### 环境

- Go 1.22+
- Node.js 20+

### 启动

**终端 1 — API**

```bash
cd apps/server
go run ./cmd/server
```

默认：`http://127.0.0.1:8080`

**终端 2 — 前端**

```bash
cd apps/web
npm install
npm run dev
```

打开：`http://127.0.0.1:5173`（开发代理 `/api`、`/media` → `:8080`）

也可使用 `scripts/dev.ps1` / `scripts/dev.sh`。

### 默认管理员

| 项 | 默认 |
|----|------|
| 用户名 | `admin` |
| 密码 | `admin123` |

可通过 `.env` / `BOOKNAV_ADMIN_*` 覆盖，模板见 [`.env.example`](./.env.example)。

---

## Docker 部署

```bash
# 在仓库根目录
cp .env.example .env   # 按需修改密钥与管理员密码
docker compose -f deploy/docker-compose.prod.yml up -d --build
```

默认映射：`http://localhost:8988` → 容器 `8080`  
数据目录：`./data`（SQLite 与上传文件持久化）

开发向 compose 见 `deploy/docker-compose.yml`。

---

## API 前缀

| 前缀 | 说明 |
|------|------|
| `/api/v1/auth/*` | 登录 / 注册 / 会话 |
| `/api/v1/portal/*` | 前台聚合、搜索、快加 |
| `/api/v1/admin/*` | 管理端 |
| `/healthz` `/readyz` | 健康检查 |
| `/media/*` | 上传与图标等静态媒体 |

写操作需登录后下发的 `X-CSRF-Token`。

---

## 相关仓库

| 仓库 | 说明 |
|------|------|
| **[DEKVIW/book-nav-next](https://github.com/DEKVIW/book-nav-next)** | 当前维护版本（本仓库） |
| [DEKVIW/book-nav](https://github.com/DEKVIW/book-nav) | 旧版 Flask 实现，**不再维护** |

---

## 许可

以仓库 LICENSE 为准；未声明时请联系维护者。
