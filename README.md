<div align="center">

<a href="https://linux.do/" title="LINUX DO 社区">
  <img src="assets/linux-do.svg" width="88" height="88" alt="LINUX DO" />
</a>

### [LINUX&nbsp;DO](https://linux.do/)

**本项目在 [LINUX DO](https://linux.do/) 社区分享与交流** · 欢迎同好围观、反馈、吹水

[![LINUX DO](https://img.shields.io/badge/Community-LINUX%20DO-1c1c1e?style=for-the-badge&labelColor=ffb003&logoColor=white)](https://linux.do/)
[![Docker Image](https://img.shields.io/badge/Docker-yilan666%2Fbooknav--next-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://hub.docker.com/r/yilan666/booknav-next)
[![GitHub](https://img.shields.io/badge/GitHub-book--nav--next-181717?style=for-the-badge&logo=github)](https://github.com/DEKVIW/book-nav-next)

</div>

---

# BookNav Next

自托管个人网址导航 · **Go API + Vue 3** · Docker 一键部署

本仓库是 BookNav 的全新架构实现（前后端分离、单二进制 / Docker 部署）。  
旧版 Flask 实现见：[DEKVIW/book-nav](https://github.com/DEKVIW/book-nav)（已停止功能更新，仅作历史参考）。

> 演示 / 交流欢迎到 [LINUX DO](https://linux.do/) 发帖讨论。

| 项 | 内容 |
| --- | --- |
| 镜像 | [`yilan666/booknav-next`](https://hub.docker.com/r/yilan666/booknav-next) |
| 源码 | [`DEKVIW/book-nav-next`](https://github.com/DEKVIW/book-nav-next) |
| 默认端口 | `8989`（容器内 `8080`） |
| 数据 | `./data`（SQLite、上传、备份） |
| 向量库 | 同 compose 内 **Qdrant**（可选 AI 搜索） |

---

## 一键启动（拉取镜像，推荐）

适合：**不想本机构建**，直接从 Docker Hub 拉镜像跑起来（**含 Qdrant**）。

### 1. 准备 `docker-compose.yml`

任选一处空目录，下载根目录 compose（或 clone 整个仓库）：

```bash
# 方式 A：只下载 compose（最小）
mkdir -p booknav-next && cd booknav-next
curl -fsSL -o docker-compose.yml \
  https://raw.githubusercontent.com/DEKVIW/book-nav-next/main/docker-compose.yml

# 方式 B：clone 仓库后在根目录操作
git clone https://github.com/DEKVIW/book-nav-next.git
cd book-nav-next
```

### 2.（可选）环境变量

```bash
cp .env.example .env
# 至少修改：BOOKNAV_ADMIN_PASSWORD、BOOKNAV_SECRET_KEY
# 可选：BOOKNAV_HOST_PORT、BOOKNAV_IMAGE_TAG
```

不创建 .env 也可启动（compose 内仍有默认值）。**上线前务必修改默认管理员密码。**

### 3. 拉取并启动

```bash
docker compose pull
docker compose up -d
```

### 4. 检查

```bash
docker compose ps
curl -sS http://127.0.0.1:8989/healthz
curl -sS http://127.0.0.1:8989/version
```

浏览器打开：**http://127.0.0.1:8989**

| 项 | 默认 |
|----|------|
| 管理员用户名 | `admin` |
| 管理员密码 | `admin123` |

### 5. 常用运维命令

```bash
# 日志
docker compose logs -f booknav

# 停止 / 再启动（保留 ./data）
docker compose stop
docker compose start

# 更新到最新镜像
docker compose pull
docker compose up -d

# 停止并删除容器（默认不删 ./data 目录）
docker compose down
```

### 6. 向量 / AI 搜索（Qdrant 已随 compose 启动）

1. 登录后台 → **站点设置 → 向量配置**  
2. Qdrant 地址使用容器内：`http://qdrant:6333`（compose 已写入 `BOOKNAV_QDRANT_URL`）  
3. collection 建议：`booknav_next`  
4. 配置 Embedding（如 SiliconFlow + `bge-m3`）→ 启用 → 保存 → 后台任务 **全量向量索引**

宿主机访问 Qdrant：`http://127.0.0.1:6333`（仅本机映射）。

### 7. 数据目录权限（Linux）

镜像以 UID **65532** 写 `/data`。若无法创建数据库：

```bash
mkdir -p data/uploads data/backups data/imports data/qdrant
sudo chown -R 65532:65532 data
docker compose up -d
```

---

## 技术栈

| 层 | 选型 |
|----|------|
| 后端 | Go · Chi · SQLite · Session Cookie + CSRF |
| 前端 | Vue 3 · TypeScript · Vite · Pinia · Vue Router |
| 图标 | Lucide |
| 向量 | Qdrant（可选） |
| 部署 | Docker 多阶段构建 · Hub 镜像 · 静态前端嵌入 |

---

## 仓库结构

```
book-nav-next/
├── apps/
│   ├── server/              # Go API 与静态资源服务
│   └── web/                 # Vue 3 前端
├── deploy/                  # Dockerfile、多种 compose 变体
├── docker-compose.yml       # 【推荐】拉取 Hub 镜像 + Qdrant
├── scripts/                 # 本地开发与进程守护
├── data/                    # 运行时数据（本地，不入库）
├── assets/                  # README 资源（含 LINUX DO logo）
├── .env.example             # 唯一环境变量模板（复制为 .env）
└── README.md
```

| 路径 | 职责 |
|------|------|
| `apps/server` | REST API、鉴权、后台任务、SQLite、静态托管 |
| `apps/web` | 前台导航与后台管理 SPA |
| `docker-compose.yml` | **拉镜像一键跑**（booknav + qdrant） |
| `deploy/` | 从源码 build、Hub 变体等进阶编排 |
| `data` | 数据库与上传（`.gitignore`） |

---

## 功能概览

**前台**

- 分类侧栏 + 页内分区滚动、子分类 Tab  
- 站点卡片、搜索、精选、AI 搜索（需配置向量/模型）  
- 管理员：粘贴 URL 快加、重复链接、右键菜单、拖拽排序  
- 登录 / 邀请码注册、外链跳转页、私有链接可见性  

**后台**

- 网站与分类、图标、用户与邀请码、站点设置  
- 向量 / AI 配置、导出备份、死链检测、图标抓取、向量索引等任务  

---

## 本地开发（源码）

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

打开：`http://127.0.0.1:5173`（代理 `/api`、`/media` → `:8080`）

也可使用 `scripts/dev.ps1` / `scripts/dev.sh`。

### 默认管理员

| 项 | 默认 |
|----|------|
| 用户名 | `admin` |
| 密码 | `admin123` |

可通过 `.env` / `BOOKNAV_ADMIN_*` 覆盖，见 [`.env.example`](./.env.example)。

---

## 从源码构建镜像（可选）

```bash
cp .env.example .env
docker compose -f deploy/docker-compose.prod.yml up -d --build
```

默认：`http://localhost:8988` → 容器 `8080`。  
仅 pull、不 build 的变体：`deploy/docker-compose.hub.yml`。

---

## API 前缀

| 前缀 | 说明 |
|------|------|
| `/api/v1/auth/*` | 登录 / 注册 / 会话 |
| `/api/v1/portal/*` | 前台聚合、搜索、快加 |
| `/api/v1/admin/*` | 管理端 |
| `/healthz` `/readyz` | 健康检查 |
| `/media/*` | 上传与图标等静态媒体 |

写操作需登录后的 `X-CSRF-Token`。

## 许可

以仓库 LICENSE 为准；未声明时请联系维护者。
