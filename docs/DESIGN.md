# SkillHub 产品与架构方案

## 使用方式

SkillHub 是本地单用户的 GitHub Skill 工作台。自有 Skill 集中放在 GitHub 仓库的 `skills/<name>/` 中。使用 Git 管理文件、发布和回退，在 SkillHub 收录、筛选、对比和生成安装命令。第三方仓库可以按 Skill 子目录单独收录。

首页保留搜索、可展开的列表和比对入口。收录时从指定分支、标签或 commit 读取一个 Skill，或扫描仓库 `skills/` 下的直接子目录。预览显示发现的 Skill 和解析得到的 commit，保存只读取这个 commit。检查新版本时对比最近收录版本与候选 commit，再由用户保存。

比对页可选择已收录版本或本机目录。扫描本机项目后可选取 `.agents/skills` 和 `.codex/skills` 中发现的 Skill。比较的结果包含文件增删改，二进制文件只显示变化状态。

部署在本机时，后端监听回环地址。公开与私有 GitHub 仓库都使用 Git 的 HTTPS 方式。私有仓库凭据由运行后端的用户配置。

## 数据和实现

- **GitHub**保存 Skill 文件及历史。分支或 tag 用于选择引用，保存及安装绑定完整 commit SHA。读取时使用临时 checkout，比对完成后清理。
- **SQLite**保存 `Skill`、`SkillVersion` 和 `SkillTag`。Skill 以规范化后的仓库地址和相对路径唯一，版本以 Skill ID 和 commit SHA 唯一。版本记录也保存对应的 Skill 名称，确保名称变更后仍可安装历史版本。
- **事务**覆盖批量收录及其初始版本、追加版本及列表元数据更新、分类备注及标签保存。重复收录报告已有条目，重复保存 commit 返回冲突。删除目录记录会同时删除其版本和标签。
- **Fino**从 `proto/skillhub/` 生成 Go API、服务和前端客户端。Go 应用层编排 GitHub 读取、目录比对及 SQLite 写入。React 复用现有的列表、弹窗、主题及差异组件。
- **Vercel skills**负责用户终端的 Codex 项目级和全局安装。生成命令时固定 CLI 版本、仓库、Skill 子目录和 commit。生成命令只表示可供执行，实际安装结果由终端 CLI 返回。

默认目录库为 `backend/skillhub-catalog.db`。可以从 GitHub 重新收录 Skill；分类、备注、标签和收藏的版本列表随 SQLite 备份恢复。

## 验收

使用含两个 Skill 和两个 commit 的 Git 仓库，验证批量收录、重复收录、预览后分支移动、历史版本差异、分类备注与标签、项目和全局安装命令。通过 Fino 生成、Go 测试和前端构建检查契约和产物的一致性。
