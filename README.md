# SkillHub

SkillHub 是一个基于 Fino 的本地 Skill 管理项目。当前已生成 Go service、SQLite 仓储和 React 前端，并提供项目 Skill 扫描、本地和 GitHub 来源收录及初始版本查看；已有 Skill 的更新、版本恢复、线上比较、安装命令及标签筛选等业务接口仍待实现，不能将生成骨架视为完整产品。

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

默认 HTTP 服务监听 `127.0.0.1:31101`；前端地址见 Vite 启动输出。开发环境下 Vite 将 `/skillhub` 请求代理到后端；需要连接其他服务时可设置 `VITE_API_BASE`。SQLite 默认保存于后端工作目录下的 `skillhub.db`。`.env` 包含生成的分页密钥，应保留且不要提交。

## 项目扫描

```bash
curl -fsS http://127.0.0.1:31101/skillhub/v1/projects:scan \
  -H 'Content-Type: application/json' \
  -d '{"projectPath":"/项目的绝对路径"}'
```

支持直接扫描包含 `SKILL.md` 的单个 Skill 目录、名为 `skills` 的集合目录，也支持扫描项目的 `.agents/skills` 和 `.codex/skills`；允许指向项目内的目录链接并按真实路径去重，拒绝指向项目外的链接。目前扫描只发现 Skill，尚未关联管理库或计算同步状态；结果标记为无同步基准。

## 收录 Skill

扫描项目后勾选 Skill，预览文件并确认收录。疑似敏感文件默认不选；收录时会复制所选文件并创建初始版本。公开 Skill 填写 GitHub 仓库地址、仓库内路径及分支或标签，预览后只保存来源和解析到的 commit，不复制其文件。

同名个人 Skill 或已收录的相同公开来源会提示冲突，不会自动覆盖；更新已有 Skill 和差异对比尚未实现。个人内容和快照默认保存在 `backend/skillhub-data/`，可用 `SKILLHUB_DATA_DIR` 更改；备份时请停止服务，并同时备份该目录与 SQLite 数据库。

## 检查

```bash
go -C go test ./...
go -C backend test ./...
cd frontend && npm run build
```

产品范围和待完成设计见 [设计文档](docs/DESIGN.md)。目前的 Skill CRUD 仍是 Fino 生成骨架，标签编辑、已有 Skill 更新、版本恢复及文件编辑等业务逻辑尚未完成。
