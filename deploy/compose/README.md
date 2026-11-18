# Docker Compose

`compose.yaml` 运行 SkillHub 后端和前端。`compose.infra.yaml` 提供可复用的日志、指标和链路观测服务，两者通过 Docker 网络 `infra` 连接。

命令默认在项目根目录执行。集群部署见 [Helm 文档](../helm/README.md)。

## 构建与发布

准备 GoReleaser 2.15+、项目 `go.mod` 要求的 Go，以及带 Buildx 的 Docker。Node.js 版本见 `frontend/package.json`，构建 hook 会安装前端依赖并生成静态资源。

以下命令生成 `linux/amd64` 镜像。SQLite 后端启用了 CGO，通过 Zig 编译并静态链接。部署到群晖等设备前，先确认目标机支持该镜像架构。

### 本地构建

```sh
goreleaser check --config .goreleaser.yaml
CC='zig cc -target x86_64-linux-musl' \
  goreleaser release --config .goreleaser.yaml --snapshot --clean
```

Snapshot 将镜像加载到本机 Docker，归档和校验文件写入 `.fino/release/`。使用 `--skip=publish` 会跳过镜像构建。

### 正式发布

在干净工作区切到带版本 Git tag 的提交，并登录 `dockers_v2.images` 指定的镜像仓库。正式构建按配置推送应用镜像。部署时可在 `.env` 中使用本次发布的版本标签，便于回滚。

```sh
CC='zig cc -target x86_64-linux-musl' \
  goreleaser release --config .goreleaser.yaml --clean
```

## 配置与启动

### 准备配置

首次部署时从示例创建 `deploy/compose/.env`。更新时保留已有的 `PAGINATION_ENCODED_KEY`，以免已发出的分页 token 失效。

```sh
test -f deploy/compose/.env || cp deploy/compose/.env.example deploy/compose/.env
```

将 `IMAGE_REPOSITORY`、`SKILLHUB_SERVER_IMAGE` 和 `SKILLHUB_FRONTEND_IMAGE` 与本地构建产物或要部署的发布版本对齐。可以用下方命令核对完整镜像名和标签。启用共享观测平台时，在同一份 `.env` 中设置 `GRAFANA_ADMIN_PASSWORD`。

```sh
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.yaml config --images
```

### 启动

首次运行共享观测平台时，先启动平台，再启动业务。共享平台已运行时，直接执行第二条命令。

```sh
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.infra.yaml up -d --wait
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.yaml up -d --wait
```

SkillHub 使用本地 SQLite，可以只启动业务。首次部署时用 `docker network create infra` 创建业务所需的网络，已有同名网络可直接复用。

### 访问与数据

通过 Tailscale 直接访问时，在群晖的 `deploy/compose/.env` 中统一配置访问入口，再按前面的业务 Compose 命令启动。

```dotenv
SKILLHUB_FRONTEND_BIND=100.118.125.84
SKILLHUB_FRONTEND_PORT=30011
SKILLHUB_URL=http://100.118.125.84:30011
```

`SKILLHUB_FRONTEND_BIND` 和 `SKILLHUB_FRONTEND_PORT` 决定宿主机监听的地址和端口。`SKILLHUB_URL` 填浏览器实际打开的地址，供后端校验 API 请求。使用群晖 HTTPS 反向代理时，可以保留回环绑定，将 `SKILLHUB_URL` 设为 HTTPS 地址，并让代理保留浏览器的 Host。

本机访问时可使用默认入口 `http://127.0.0.1:8082`，并留空 `SKILLHUB_URL`。启用共享观测平台后，Grafana 入口是同一地址下的 `/monitor/`。

SQLite、快照、上传暂存和导出文件保存在 `skillhub_data` 卷，应用重建时会复用该卷。备份 SQLite 和文件时，先停后端，再备份整卷。

## 查看状态、日志与停止

后端启动后会在 `skillhub_data` 卷中自动创建 `logs/app.log`，容器重建时会保留日志。

使用 `ps` 查看容器状态，使用 `tail` 跟踪后端日志。前端和容器启动脚本的输出仍可通过 `docker compose logs` 查看。

```sh
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.yaml ps
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.yaml exec skillhub-server tail -f /app/skillhub-data/logs/app.log
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.yaml logs -f
```

停止业务时执行以下命令。

```sh
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.yaml down
```

确认其他服务已停用共享平台后，可对 `compose.infra.yaml` 执行 `down`。`down -v` 会删除持久化卷。

## 离线包

离线包包含镜像、Compose 配置和后端配置，可用于传到群晖或其他 Docker 主机。打包前先完成[本地构建](#本地构建)或拉取应用镜像，再按 `.env` 拉取共享平台镜像。确认目标机的架构与打包的镜像一致。

包内包含当前 `.env`，请按凭据文件保管。迁移已有 SkillHub 时，另行备份和恢复 `skillhub_data` 卷。

```sh
PACKAGE_NAME="package-$(date +%Y%m%d-%H%M%S)"
PACKAGE_DIR="${PWD}/.fino/release/${PACKAGE_NAME}"

mkdir -p "${PACKAGE_DIR}/deploy" "${PACKAGE_DIR}/backend" "${PACKAGE_DIR}/images"

for compose in compose.infra.yaml compose.yaml; do
  docker compose \
    -f "deploy/compose/${compose}" \
    --env-file "deploy/compose/.env" \
    config --images
done | sort -u > "${PACKAGE_DIR}/images/images.txt"

#xargs docker image save \
#  -o "${PACKAGE_DIR}/images/images.tar" \
#  < "${PACKAGE_DIR}/images/images.txt"

cp -R deploy/compose "${PACKAGE_DIR}/deploy/"
cp -R backend/configs "${PACKAGE_DIR}/backend/"

COPYFILE_DISABLE=1 tar -C "${PWD}/.fino/release" \
  -czf "${PACKAGE_DIR}.tgz" "${PACKAGE_NAME}"
(cd .fino/release && shasum -a 256 "${PACKAGE_NAME}.tgz" > "${PACKAGE_NAME}.tgz.sha256")
```

将生成的 `.tgz` 与 `.tgz.sha256` 一起传到目标机，在存放这对文件的目录中校验、解包并导入镜像。

```sh
shasum -a 256 -c package-*.tgz.sha256
tar -xzf package-*.tgz
docker image load -i package-*/images/images.tar

cd package-*/
docker compose -f "deploy/compose/compose.infra.yaml" --env-file "deploy/compose/.env" up -d --wait
docker compose -f "deploy/compose/compose.yaml" --env-file "deploy/compose/.env" up -d --wait
```
