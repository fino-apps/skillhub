# SkillHub

SkillHub 是一个基于 Fino 的本地 Skill 管理项目。当前已生成 Go service、SQLite 仓储和 React 前端，并提供项目 Skill 扫描；版本管理、导入、线上比较、安装命令及标签筛选等业务接口仍待实现，不能将生成骨架视为完整产品。

## 本地运行

需要 Fino、Go 1.25.13 或更新版本、`protoc-gen-go`、`protoc-gen-go-grpc`，以及 Node.js 20.19 或更新版本。

```bash
fino doctor --output=. --targets=service,frontend
fino build --output=. --targets=service,frontend
cd backend
cp -n .env.example .env
go run ./cmd/skillhub-server
```

在另一个终端启动前端：

```bash
cd frontend
npm ci
npm run dev
```

默认 HTTP 服务监听 `127.0.0.1:31101`；前端地址见 Vite 启动输出。SQLite 默认保存于后端工作目录下的 `skillhub.db`。`.env` 包含生成的分页密钥，应保留且不要提交。

## 项目扫描

```bash
curl -fsS http://127.0.0.1:31101/skillhub/v1/projects:scan \
  -H 'Content-Type: application/json' \
  -d '{"projectPath":"/项目的绝对路径"}'
```

扫描 `.agents/skills` 和 `.codex/skills`，允许指向项目内的目录链接并按真实路径去重，拒绝指向项目外的链接。目前扫描只发现 Skill，尚未关联管理库或计算同步状态；结果标记为无同步基准。

## 检查

```bash
go -C go test ./...
go -C backend test ./...
cd frontend && npm run build
```

产品范围和待完成设计见 [设计文档](docs/DESIGN.md)。目前的 Skill CRUD 是 Fino 生成骨架，尚未完成标签、目录内容和版本等业务逻辑；不要用于重要数据。
