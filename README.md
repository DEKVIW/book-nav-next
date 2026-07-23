# BookNav

自托管网址导航站（重构版）· **Go API + Vue 3 + 机甲科幻 UI**

旧 Flask 实现仅作能力参考。本仓库为干净重写。  
设计文档：[`docs/README.md`](./docs/README.md)

---

## 技术栈

| 层 | 选型 |
|----|------|
| 后端 | Go · Chi · SQLite (modernc) · Session Cookie + CSRF |
| 前端 | Vue 3 · TypeScript · Vite · Pinia · Vue Router |
| UI | 机甲舰桥设计令牌 |
| AI | 可降级搜索（SSE 骨架）；Qdrant 可选后续接入 |
| 部署 | Docker 多阶段 · 单二进制 |

---

## 功能清单（已实现）

### 前台
- 单页导航：分类 **页内滚动 + 子分类 Tab**（不进独立分类页）
- 链接卡片、精选、搜索
- 管理员：**粘贴 URL 快加**、重复链接三分支、右键菜单、拖拽排序
- 登录 / 邀请码注册、过渡跃迁页
- 私有链接可见性（关联表，无字符串串号）

### 后台（指挥桥）
- 总览统计、网站/分类 CRUD、邀请码
- 用户列表（超管）、站点设置、操作任务
- 导出 JSON、本地备份、死链检测 Job、图标抓取 Job

---

## 仓库结构

```
book-nav/
├── apps/server/     # Go API
├── apps/web/        # Vue 前端
├── deploy/          # Docker
├── docs/            # 重构文档
├── scripts/         # dev 脚本
└── data/            # 运行时数据（gitignore）
```

---

## 本地开发

### 前置
- Go 1.22+
- Node.js 20+

### 启动

**终端 1 — API**

```powershell
cd apps/server
go run ./cmd/server
```

默认：`http://127.0.0.1:8080`  
首次启动会创建超管并写入演示分类/链接。

**终端 2 — 前端**

```powershell
cd apps/web
npm install
npm run dev
```

打开：`http://127.0.0.1:5173`（已代理 `/api` → `:8080`）

### 默认账号

| 项 | 默认 |
|----|------|
| 用户名 | `admin` |
| 密码 | `admin123` |

可通过环境变量 `BOOKNAV_ADMIN_*` / `.env` 修改（见 `.env.example`）。

---

## 生产构建

```powershell
# 前端
cd apps/web; npm run build
# 将 dist 拷到 apps/server/webdist 后：
cd ../server; go build -o bin/booknav ./cmd/server
```

或 Docker：

```powershell
cd deploy
docker compose up -d --build
```

访问 `http://localhost:8988`

AI/Qdrant（可选）：

```powershell
docker compose --profile ai up -d
```

---

## API 摘要

| 前缀 | 说明 |
|------|------|
| `/api/v1/auth/*` | 登录注册 |
| `/api/v1/portal/*` | 前台聚合/搜索/快加 |
| `/api/v1/admin/*` | 管理端 |
| `/healthz` `/readyz` | 探针 |

统一响应：`{ success, data, error }`。写操作需 `X-CSRF-Token`（登录后下发）。

---

## 产品约定

| 点 | 说明 |
|----|------|
| 分类导航 | 页内定位，**不做** `/category/:id` 主路径 |
| 交互保留 | 拖拽、右键、粘贴快加、重复检测 |
| 后台 | 能力对齐旧版，SPA + REST |

---

## 文档

| 文档 | 内容 |
|------|------|
| [docs/05-frontend-interactions.md](./docs/05-frontend-interactions.md) | 前台交互规格 |
| [docs/06-ui-mecha-design.md](./docs/06-ui-mecha-design.md) | 机甲 UI |
| [docs/10-implementation-roadmap.md](./docs/10-implementation-roadmap.md) | 路线图 |

---

## 许可

按你的仓库策略自行补充 LICENSE。
