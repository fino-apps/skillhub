# SkillHub

SkillHub 是单机 Skill 技能库。直接导入或导出完整文件夹，也可以按 GitHub commit 收录第三方 Skill。已保存版本可离线查看、比对和导出。

## 前后端分开部署

从项目根目录用 `.goreleaser.yaml` 构建两份 Linux AMD64 镜像。需要 Go 1.25.13 或更新版本、符合 `frontend/package.json` 要求的 Node.js、GoReleaser 2.15+、Zig 和 Docker。SQLite 需要 CGO，下面用 Zig 编译并静态链接到后端程序。GoReleaser 会安装前端依赖并构建网页，将镜像加载到本机 Docker。

```sh
CC='zig cc -target x86_64-linux-musl' \
  goreleaser release --config .goreleaser.yaml --snapshot --clean
```

首次部署时从 `.env.example` 创建 `.env`，并准备 Fino 的共享 Docker 网络。已有网络可以直接复用。

```sh
test -f deploy/compose/.env || cp deploy/compose/.env.example deploy/compose/.env
docker network inspect infra >/dev/null 2>&1 || docker network create infra
docker compose --env-file deploy/compose/.env \
  -f deploy/compose/compose.yaml up -d --wait
```

打开 `http://127.0.0.1:8082`。Caddy 提供页面，并通过 Docker 网络转发 `/skillhub/v1` 请求。SQLite、快照和导出文件保存在 `skillhub_data` 卷。容器部署时，导入路径必须能被后端容器访问，也可以从浏览器导入或下载 ZIP。

| 环境变量 | 默认值与用途 |
| --- | --- |
| `SERVER_HTTP_PORT` | `31101`，后端 HTTP 端口 |
| `SERVER_GRPC_PORT` | `31102`，Fino 本机 gRPC 端口 |
| `SERVER_DEBUG_PORT` | `31103`，Fino 本机调试端口，提供 `/readyz` 和 `/healthz` |
| `SKILLHUB_EXPORT_DIR` | 默认使用后端工作目录下的 `skillhub-data/exports` |

本地数据默认保存在 `backend/skillhub-data`。备份时先停止后端，再复制 Docker 卷或本地数据目录。自定义 `db.dsn` 或 `SKILLHUB_SNAPSHOT_DIR` 后，备份时一并复制对应位置。版本快照以 ZIP 保存在快照目录。缺少快照的 GitHub 版本可按原 commit 重新拉取。

## 日常使用

- **本机收录**。选择含 `SKILL.md` 的文件夹，或选择项目根目录后勾选需要收录的 Skill。同一真实目录只收录一次。
- **GitHub 收录**。粘贴仓库或 Skill 目录链接，预览实际 commit 后收录。更新时可比较候选与当前默认版本。支持单个 Skill 和目录下的多个候选。
- **版本和导出**。首次收录设为默认，后续保存可按需设为默认。首页导出默认版本，详情导出正在查看的版本。ZIP 用于跨机器交换。历史版本可删除，删除默认版本前先改选。
- **ZIP 导入和下载**。上传支持最多 64 MiB 的单个 ZIP，预览后可选择要导入的条目。扫描结果支持分批收录，每批最多 100 项。ZIP 预览暂存保留 24 小时，后续上传或确认导入时清理过期目录，过期后需要重新上传。上传、确认导入和附件下载的接口都在 SkillHub 的 proto 中声明。
- **整理**。按名称、用途、来源、分类和标签查找；多选可批量修改分类和标签。

## 开发与验证

修改契约后从项目根目录重新生成。

```bash
fino doctor --output=. --targets=api,service
fino build --output=. --targets=api,service
go -C go test ./...
go -C backend test ./...
npm --prefix frontend run build
```

前端继续使用仓库已有的 API 客户端。调整 HTTP 契约时，增加 `frontend` 生成目标并重新构建前端。

本地开发先从 `backend/.env.example` 创建 `backend/.env`。已有本地数据位于其他目录时，可在停服后迁入 `backend/skillhub-data`，也可通过 `db.dsn` 和 `SKILLHUB_SNAPSHOT_DIR` 继续使用原位置。执行 `npm --prefix frontend ci`，再分别运行 `go -C backend run ./cmd/skillhub-server` 与 `npm --prefix frontend run dev`。Vite 将业务 API 代理到本机后端，本机目录选择和 Git 使用当前用户的环境。

## 后端分层

`cmd/skillhub-server` 负责配置、建表和依赖装配。Fino 生成的 HTTP 与 gRPC 路由进入 `app`，由 App 转换 Proto 入参、组装响应并映射业务错误。`domain/service` 管理收录、版本快照和元数据规则，通过 `domain/repo` 的仓储接口访问数据。`infra/repo` 负责 SQLite 查询、事务和游标分页。
