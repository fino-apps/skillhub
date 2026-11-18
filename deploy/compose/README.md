# Docker Compose

命令默认在项目根目录执行。集群部署见 [Helm 文档](../helm/README.md)。

## 构建与发布

准备 GoReleaser 2.15+、项目 `go.mod` 要求的 Go，以及带 Buildx 的 Docker。Node.js 版本见 `frontend/package.json`，构建 hook 会安装前端依赖并生成静态资源。

以下命令按 `linux/amd64` 构建。SQLite 后端需要 Zig 编译 CGO。

### 本地构建

```sh
goreleaser check --config .goreleaser.yaml
CC='zig cc -target x86_64-linux-musl' \
  goreleaser release --config .goreleaser.yaml --snapshot --clean
```

Snapshot 将镜像加载到本机 Docker，归档和校验文件写入 `.fino/release/`。使用 `--skip=publish` 会跳过镜像构建。

启动前将 `.env` 的 `IMAGE_REPOSITORY` 和应用镜像标签与构建产物对齐，可以用以下命令核对。

```sh
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.yaml config --images
```

### 正式发布

在干净工作区切到带版本 Git tag 的提交，并登录 `dockers_v2.images` 指定的镜像仓库。正式构建按配置推送应用镜像。部署时可在 `.env` 中使用本次发布的版本标签，便于回滚。

```sh
CC='zig cc -target x86_64-linux-musl' \
  goreleaser release --config .goreleaser.yaml --clean
```

## 配置与启动

首次部署时创建 `.env`。更新时保留已有的 `PAGINATION_ENCODED_KEY`。

```sh
test -f deploy/compose/.env || cp deploy/compose/.env.example deploy/compose/.env
```

使用 Grafana 时，先在 `.env` 中设置 `GRAFANA_ADMIN_PASSWORD`。两份 Compose 通过外部网络 `infra` 连接，分别启动；共享平台已运行时可直接启动业务。

```sh
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.infra.yaml up -d --wait
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.yaml up -d --wait
```

前端默认访问 http://127.0.0.1:8082，Grafana 入口为 http://127.0.0.1:8082/monitor/。后端端口仅在 Docker 网络内开放。SQLite、快照、上传暂存和导出文件保存在 `skillhub_data` 卷。

只运行 SkillHub 时，也可以用 `docker network create infra` 创建共享网络，再单独启动业务 Compose。

## 日志与停止

```sh
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.yaml logs -f
docker compose --env-file deploy/compose/.env -f deploy/compose/compose.yaml down
```

确认其他服务已停用共享平台后，可对 `compose.infra.yaml` 执行 `down`。`down -v` 会删除持久化卷。

## 离线包

生成离线包前，先完成[本地构建](#本地构建)或拉取对应版本的应用镜像，并按 `.env` 拉取共享平台镜像。

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

xargs docker image save \
  -o "${PACKAGE_DIR}/images/images.tar" \
  < "${PACKAGE_DIR}/images/images.txt"

cp -R deploy/compose "${PACKAGE_DIR}/deploy/"
cp -R backend/configs "${PACKAGE_DIR}/backend/"

COPYFILE_DISABLE=1 tar -C "${PWD}/.fino/release" \
  -czf "${PACKAGE_DIR}.tgz" "${PACKAGE_NAME}"
(cd .fino/release && shasum -a 256 "${PACKAGE_NAME}.tgz" > "${PACKAGE_NAME}.tgz.sha256")
```

目标机导入并启动：

```sh
shasum -a 256 -c package-*.tgz.sha256
tar -xzf package-*.tgz
docker image load -i package-*/images/images.tar

cd package-*/
docker compose -f "deploy/compose/compose.infra.yaml" --env-file "deploy/compose/.env" up -d --wait
docker compose -f "deploy/compose/compose.yaml" --env-file "deploy/compose/.env" up -d --wait
```
