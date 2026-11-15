## `skillhub.Skill`

Skill 保存 GitHub 目录的索引和本地整理信息。

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  | 主键 |
| `name` | `string` |  | 否 |  | Skill 名称 |
| `description` | `string` |  | 否 |  | 用途说明 |
| `category` | `string` |  | 否 |  | 分类 |
| `notes` | `string` |  | 否 |  | 个人备注 |
| `repositoryUrl` | `string` |  | 否 |  | GitHub 来源地址 |
| `repositoryPath` | `string` |  | 否 |  | 仓库内 Skill 路径 |
| `createTime` | `string` | `Timestamp` | 否 |  | 创建时间 |
| `updateTime` | `string` | `Timestamp` | 否 |  | 最后更新时间 |


## `skillhub.SkillTag`

SkillTag 保存 Skill 的标签，同一 Skill 的标签唯一。

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  | 主键 |
| `skillId` | `string` |  | 否 |  | Skill ID |
| `tag` | `string` |  | 否 |  | 标签 |
| `createTime` | `string` | `Timestamp` | 否 |  | 创建时间 |
| `updateTime` | `string` | `Timestamp` | 否 |  | 最后更新时间 |


## `skillhub.SkillVersion`

SkillVersion 指向 GitHub 的固定 commit。

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  | 主键 |
| `skillId` | `string` |  | 否 |  | Skill ID |
| `name` | `string` |  | 否 |  | 该 commit 中的 Skill 名称 |
| `notes` | `string` |  | 否 |  | 版本说明 |
| `sourceRef` | `string` |  | 否 |  | 用户选择的 Git 引用 |
| `commitSha` | `string` |  | 否 |  | 完整 commit SHA |
| `createTime` | `string` | `Timestamp` | 否 |  | 创建时间 |
| `updateTime` | `string` | `Timestamp` | 否 |  | 最后更新时间 |
