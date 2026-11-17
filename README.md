# SkillHub

SkillHub 是单机 Skill 技能库。直接导入或导出完整文件夹，也可以按 GitHub commit 收录第三方 Skill。已保存版本可离线查看、比对和导出。

## 单机运行

构建需要 Go 1.25.13 或更新版本、Node.js 20.19 或更新版本，以及用于 SQLite 的本机 C 编译器。首次构建先安装前端依赖。

```bash
npm --prefix frontend ci
bash backend/scripts/build-desktop.sh
.fino/release/skillhub
```

打开 `http://127.0.0.1:31101`。发布包内置前端和 API。GitHub 收录使用本机 Git 与 HTTPS 凭据。macOS 使用系统目录选择，Linux 在有图形桌面和 `zenity` 时使用目录选择器，其他环境可以输入绝对路径。

| 环境变量 | 默认值与用途 |
| --- | --- |
| `SERVER_HTTP_PORT` | `31101`，回环 HTTP 端口 |
| `SKILLHUB_DATA_DIR` | 系统用户配置目录下的 `SkillHub`，保存数据库和快照 |
| `SKILLHUB_EXPORT_DIR` | 系统下载目录下的 `SkillHub`，不可用时回退到数据目录的 `exports` |

备份时先停止服务，再复制整个数据目录。若显式设置 `SKILLHUB_DATABASE` 或 `SKILLHUB_SNAPSHOT_DIR`，备份时一并复制这些位置；这两个变量也可用来打开原数据库和快照目录。旧 GitHub 条目若只有 commit 记录，可在页面补齐对应版本副本。

## 日常使用

- **本机收录**。选择含 `SKILL.md` 的文件夹，或选择项目根目录后勾选需要收录的 Skill。同一真实目录只收录一次。
- **GitHub 收录**。粘贴仓库或 Skill 目录链接，预览实际 commit 后收录。更新时可比较候选与当前默认版本。支持单个 Skill 和目录下的多个候选。
- **版本和导出**。首次收录设为默认，后续保存可按需设为默认。首页导出默认版本，详情导出正在查看的版本。ZIP 用于跨机器交换。历史版本可删除，删除默认版本前先改选。
- **整理**。按名称、用途、来源、分类和标签查找；多选可批量修改分类和标签。

## 开发与验证

修改契约后从项目根目录重新生成。

```bash
fino doctor --output=. --targets=service,frontend
fino build --output=. --targets=service,frontend
go -C go test ./...
go -C backend test ./...
npm --prefix frontend run build
```

开发时分别运行 `go -C backend run ./cmd/skillhub-server` 与 `npm --prefix frontend run dev`，Vite 将业务 API 代理到本机后端。

[技能库架构方案](docs/SKILLHUB_ARCHITECTURE.md) · [前端页面设计](docs/SKILLHUB_FRONTEND_DESIGN.md)
