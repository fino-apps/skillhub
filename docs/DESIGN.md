# SkillHub 产品与架构方案

## 使用方式

SkillHub 是本地单用户的 Skill 工作台。项目里的本地 Skill 可以按原目录收录并保存独立的版本副本。GitHub Skill 以仓库和子目录收录，通过 commit 获取内容。两种来源都支持筛选、对比和生成安装命令。

首页保留搜索、可展开的列表和比对入口。本地首次收录保存初始副本，之后按需从原目录保存新副本。GitHub 收录时从指定分支、标签或 commit 读取一个 Skill，或扫描仓库 `skills/` 下的直接子目录。GitHub 预览会解析到固定 commit，再由用户保存引用。

比对页可选择已收录版本或本机目录。扫描本机项目后可选取 `.agents/skills` 和 `.codex/skills` 中发现的 Skill。比较的结果包含文件增删改，二进制文件只显示变化状态。

部署在本机时，后端监听回环地址。公开与私有 GitHub 仓库都使用 Git 的 HTTPS 方式。私有仓库凭据由运行后端的用户配置。

## 数据和实现

- **GitHub**保存 Skill 文件及历史。分支或 tag 用于选择引用，保存及安装绑定完整 commit SHA。读取时使用临时 checkout，比对完成后清理。
- **SQLite**保存 `Skill`、`SkillVersion` 和 `SkillTag`。本地 Skill 按真实原目录去重，版本按文件内容摘要去重。GitHub Skill 按仓库地址和相对路径去重，版本按 commit 去重。版本记录保存当时的 Skill 名称。
- **本地版本文件**按 Skill ID 和版本 ID 分开保存。先在副本目录内暂存完整文件，再以重命名发布。SQLite 事务保存版本记录并更新列表元数据，失败时清理这次的副本。删除本地条目时清理其记录、标签和副本。
- **Fino**从 `proto/skillhub/` 生成 Go API、服务和前端客户端。Go 应用层编排 GitHub 读取、目录比对及 SQLite 写入。React 复用现有的列表、弹窗、主题及差异组件。
- **Vercel skills**负责用户终端的 Codex 项目级和全局安装。生成命令时固定 CLI 版本和所选的本地副本或 GitHub commit。生成命令只表示可供执行，实际安装结果由终端 CLI 返回。

默认目录库为 `backend/skillhub-catalog.db`，本地副本位于 `backend/skillhub-snapshots/`。两者一起备份和恢复。

## 验收

验证本地批量收录、重复保存、修改原目录后保存新版本，以及移走原目录后的历史差异和安装命令。用 GitHub 来源核对固定 commit 读取、比对和安装。通过 Fino 生成、Go 测试和前端构建检查契约和产物的一致性。
