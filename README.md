# Skillhub

这是一个用 Fino 生成的 Go 服务。默认示例提供 Resource 的 HTTP/gRPC CRUD，数据保存在内存中，重启后清空。

## 本地运行

需要 Go 1.25.13 或更新版本、Fino，以及 PATH 中的 `protoc-gen-go` 和 `protoc-gen-go-grpc`。在项目根目录执行：

```bash
fino doctor --targets=api,service
fino build --targets=api,service
cd backend
test -f .env || cp .env.example .env
go run ./cmd/skillhub-server
```

默认 HTTP、gRPC、调试端口分别为 `31101`、`31102`、`31103`。在 `backend/.env` 中修改端口，后续启动沿用已有的环境文件和分页密钥。用 Ctrl-C 停止服务。

本地演示配置监听所有网卡，HTTP 和 gRPC 默认未启用认证。对外部署前配置访问控制，并为每个部署设置独立的 `PAGINATION_ENCODED_KEY`，重启时沿用该值。

服务默认加载 `backend/configs/` 中的 YAML。需要接入 Nacos 时，按 `backend/.env.example` 配置 `NACOS_SERVER_ADDR` 和对应的命名空间、分组及凭据。服务启动时优先读取远端 YAML，读取或加载失败时使用本地配置。

## 调用接口

启动后，在另一个终端创建一条资源：

```bash
curl -fsS http://localhost:31101/skillhub/v1/resources \
  -H 'Content-Type: application/json' -d '{"name":"first resource"}'
```

将响应中的 `id` 填入下方命令，查询刚创建的资源：

```bash
id=替换为返回的id
curl -fsS "http://localhost:31101/skillhub/v1/resources/$id"
```

更多接口见 [API 文档](docs/) 和 [OpenAPI](openapi/)。这两个目录在生成后可用。

## 修改契约与重新生成

从 [proto/](proto/) 修改 RPC、消息或 HTTP 绑定，再在项目根目录执行：

```bash
fino build --targets=api,service
go -C go test ./...
go -C backend test ./...
```

需要 Go 客户端或 Web 前端时，将 `client` 或 `frontend` 加入构建目标。生成前端还需要 Node.js 和 npm，可用 `fino doctor --targets=frontend` 检查。可编辑 [ER 配置](.fino/er.yaml) 调整生成的 [模型 ER 图](docs/er.md)。

应用入口在 `backend/cmd/skillhub-server/`，业务实现位于 `backend/internal/skillhub/`。协作与维护约定见 [AGENTS.md](AGENTS.md)。

## 部署

生成服务后，可查看 [Docker Compose](deploy/compose/README.md) 和 [Helm](deploy/helm/README.md) 的部署说明。Helm 的环境覆盖放在 `deploy/helm/environments/`。
