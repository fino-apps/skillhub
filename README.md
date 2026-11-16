# SkillHub

SkillHub 是本机运行的 Skill 工作台。可以收录项目中的 Skill 并保存版本副本，也可以收藏 GitHub Skill 的固定 commit。两种来源都支持整理、比对和生成 Codex 安装命令。

## 本地运行

需要 Fino、Git、Go 1.25.13 或更新版本，以及 `protoc-gen-go` 和 `protoc-gen-go-grpc`。前端构建需要 Node.js 20.19 或更新版本，运行安装命令需要 Node.js 22.20 或更新版本。

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

本地 HTTP 默认监听 `127.0.0.1:31101`，Vite 将 `/skillhub` 请求代理到后端，可用 `VITE_API_BASE` 指定服务地址。新目录库为 `backend/skillhub-catalog.db`。本地版本副本默认保存在 `backend/skillhub-snapshots/`，可通过 `SKILLHUB_SNAPSHOT_DIR` 设置目录。备份时先停服务，再同时备份 SQLite 文件和副本目录。

## 本地 Skill

- 在「添加 Skill → 本地目录」输入项目、Skill 集合或单个 Skill 的绝对路径。扫描后勾选条目，收录时保存初始版本副本。按原目录的真实路径去重。
- 在原项目编辑文件后，从条目点击「保存新版本」。可以先比对原目录和最近版本，再保存并填写备注。同一条目内相同文件内容只保存一次。
- 点击「比对版本」选择任意两个版本，也可以从某一版本安装到 Codex。原目录移动或删除后，已存版本依然可比对和安装。
- 副本保存普通文件和目录，保留脚本的属主执行权限，跳过 `.git`、`node_modules` 和 `.venv` 目录。Skill 内容里的符号链接会提示收录失败。单文件最多 20 MiB，每份副本最多 64 MiB、10,000 个文件。

## GitHub Skill

- 自有 Skill 可以集中放在一个 GitHub 仓库的 `skills/<name>/SKILL.md`。添加时填写仓库和分支或标签，选择「批量扫描仓库的 skills/ 目录」，预览并勾选条目。
- 单个 Skill 填仓库内路径。Skill 在仓库根目录时留空。同一个仓库目录只收录一次。
- 展开条目，点击「检查新版本」，输入分支、标签或 commit。页面展示最近收录版本与候选版本的文件差异，保存后记录候选的完整 commit SHA。
- 私有仓库沿用运行 SkillHub 的用户已配置的 Git HTTPS 凭据。首次配置后，可用 `git ls-remote https://github.com/owner/repo` 验证。生成的安装命令由执行它的用户提供自己的 GitHub 凭据。

## 比对与安装

Skill 库按「Skill / 用途 / 来源」展示。点击名称展开详情，可以在「编辑整理信息」的备注中记录请求示例。

「比对 Skill」可以选择两个收录版本，也可以输入本机 Skill 路径。输入本机项目路径后，先扫描并从发现的 Skill 中选择。比对只读，单个变更文件最大 1 MiB，单次变更内容总量最大 8 MiB。

在对应版本点击「安装」，选择「安装到项目」或「全局安装」，再复制命令到本机终端。项目级安装须先进入目标项目。命令由 Vercel `skills` 从选定的本地版本副本或 GitHub commit 安装到 Codex，重装可能替换同名 Skill。

## 检查

```bash
go -C go test ./...
go -C backend test ./...
cd frontend && npm run build
```

架构和接口范围见 [设计文档](docs/DESIGN.md)。
