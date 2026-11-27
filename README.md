# SkillHub

SkillHub 是单机 Skill 技能库。直接导入或导出完整文件夹，也可以按 GitHub commit 收录第三方 Skill。已保存版本可离线查看、比对和导出。

## macOS 桌面版

在 Intel Mac 上准备 Go、Node.js 和 Xcode Command Line Tools，再从仓库根目录运行：

```sh
./build-desktop.sh
open .fino/desktop/darwin-amd64/SkillHub.app
```

脚本使用 Go 1.27.4 和 MyGo v0.10.0。应用随包提供前端和本地 Go 服务，运行时可以直接从 Finder 启动。数据库和快照保存在 `~/Library/Application Support/SkillHub/`，日志在 `~/Library/Logs/SkillHub/app.log`。已有技能库可以通过备份 ZIP 导入。GitHub 收录需要本机 Git 和网络。

## 前后端分开部署

从项目根目录用 `.goreleaser.yaml` 构建两份 Linux AMD64 镜像。需要 Go 1.27.4 或更新版本、符合 `frontend/package.json` 要求的 Node.js、GoReleaser 2.15+、Zig 和 Docker。SQLite 需要 CGO，下面用 Zig 编译并静态链接到后端程序。GoReleaser 会安装前端依赖并构建网页，将镜像加载到本机 Docker。

```sh
CC='zig cc -target x86_64-linux-musl' \
  goreleaser release --config .goreleaser.yaml --snapshot --clean
```

首次部署时从 `.env.example` 创建 `.env`，并准备 Fino 的共享 Docker 网络。后端会在 `skillhub_data` 卷中自动创建 `logs/app.log`。

```sh
test -f deploy/compose/.env || cp deploy/compose/.env.example deploy/compose/.env
docker network inspect infra >/dev/null 2>&1 || docker network create infra
docker compose --env-file deploy/compose/.env \
  -f deploy/compose/compose.yaml up -d --wait
```

打开 `http://127.0.0.1:8082`。Caddy 提供页面，并通过 Docker 网络转发 `/skillhub/v1` 请求。SQLite、快照和导出文件保存在 `skillhub_data` 卷。容器部署时，可以直接从浏览器上传当前设备上的目录或 ZIP，无需挂载该目录。前端生产构建默认关闭桌面目录选择器。如果后端运行在有桌面环境的机器上，可在构建前端时设置 `VITE_DESKTOP_DIRECTORY_PICKER=true` 启用。

| 环境变量 | 默认值与用途 |
| --- | --- |
| `SERVER_HTTP_PORT` | `31101`，后端 HTTP 端口 |
| `SERVER_GRPC_PORT` | `31102`，Fino 本机 gRPC 端口 |
| `SERVER_DEBUG_PORT` | `31103`，Fino 本机调试端口，提供 `/readyz` 和 `/healthz` |
| `SKILLHUB_EXPORT_DIR` | 默认使用后端工作目录下的 `skillhub-data/exports` |

本地数据默认保存在 `backend/skillhub-data`。完整部署目录可在停止后端后复制 Docker 卷或本地数据目录。自定义 `db.dsn` 或 `SKILLHUB_SNAPSHOT_DIR` 后，复制时一并保存对应位置。版本快照以 ZIP 保存在快照目录。缺少快照的 GitHub 版本可按原 commit 重新拉取。

## 日常使用

目录、ZIP 和 GitHub 导入都可以在预览下方展开「整理信息（选填）」，在确认收录前填写短用途、分类、标签、来源名称、来源链接和备注。填写的信息应用到本批新增的 Skill。来源名称留空时默认显示本机目录、ZIP 或 GitHub，也可通过详情页的「编辑整理信息」调整。「全部来源」按技能库中显示的来源名称筛选。

- **目录收录**。在「导入文件」中选择当前设备上包含 `SKILL.md` 的目录，可以预览并勾选要收录的 Skill。浏览器保留相对路径并打包上传，新增 Skill 的来源显示为「本机目录」。最多支持 10000 个文件，文件总大小和打包大小均不得超过 64 MiB。
- **GitHub 收录**。粘贴仓库或 Skill 目录链接，预览实际 commit 后收录。来源链接留空时使用 GitHub 仓库地址。更新时可比较候选与当前默认版本。
- **版本和导出**。首次收录设为默认，后续保存可按需设为默认。在「版本」页点击版本号查看对应快照。首页导出默认版本，详情导出正在查看的版本；两处均可下载 ZIP 到当前设备，解压后得到一个以 Skill 命名的文件夹，也可选择或填写目录直接导出。目录由后端访问：本机部署时可导出到本机目录，容器部署时填写容器内的绝对路径。历史版本可删除，删除默认版本前先改选。
- **比对**。在 Skill 详情页的「版本」中点击「比对」。默认以当前 Skill 的默认版本为基准，可选择该 Skill 的其他版本、库里的另一个 Skill 或本地目录。
- **ZIP 导入和下载**。在首页点击「导入文件」，再选择「改用 ZIP 文件」。上传支持最多 64 MiB 的单个 ZIP，预览后可选择要导入的条目。扫描结果支持分批收录，每批最多 100 项。ZIP 上传暂存保留 24 小时，后续上传或确认导入时清理过期目录，过期后需要重新上传。上传、确认导入和附件下载的接口都在 SkillHub 的 proto 中声明。
- **数据备份与恢复**。在技能库首页的「…」菜单中选择「导出技能库备份」或「导入备份」。备份 ZIP 包含整库的 Skill 元数据、标签、历史版本、默认版本选择和已保存的快照，最大 256 MiB。预览会提示同 ID、同来源、同名或有相同版本内容的条目，可选择合并到已有 Skill、独立导入或跳过。合并按内容摘要去重，校验保留副本，优先补齐默认版本和同 ID 版本的缺失或损坏快照，并保留主条目的默认版本和整理信息。预览可展开查看将加入、补齐或跳过的版本。没有快照的版本仍保留记录。跨机器恢复会保留原始本机来源路径。
- **安装命令**。详情页生成的命令会从当前 SkillHub 地址下载 ZIP，在目标机器上需要 `curl` 和 `unzip`。直接调用安装命令 API 时，运行前设置 `SKILLHUB_URL` 为可访问的服务地址，例如 `http://127.0.0.1:8082`。
- **整理**。按名称、用途、来源、分类和标签查找。多选可批量修改分类和标签，或确认后删除所选 Skill 及其已保存版本。详情页的「… → 合并到其他 Skill」可预览版本变动，确认后保留选定主条目并移除当前重复条目。预览后条目或版本发生变化时需要重新预览。旧收录来源随主条目保留，并随整库备份恢复。

备份下载使用 `GET /skillhub/v1/library/backup`。交互式恢复先向 `POST /skillhub/v1/library/backup:preview` 的 multipart `file` 字段上传 ZIP，再向 `POST /skillhub/v1/library/backup:commit` 提交返回的 `uploadId` 和逐条选择。原 `POST /skillhub/v1/library/backup` 仍可直接导入并跳过已有条目。库内合并先调用 `POST /skillhub/v1/skills/{skill_id}:previewMerge`，再向 `POST /skillhub/v1/skills/{skill_id}:merge` 提交相同目标和预览返回的 `revision`。

批量删除接口为 `POST /skillhub/v1/skills:batchDelete`，请求体格式为 `{"skillIds":["01900000-0000-7000-8000-000000000001"]}`。每次提交 1–100 个 Skill ID，重复 ID 只处理一次，任一 ID 不存在时整批回滚。

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
