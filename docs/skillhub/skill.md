## `skillhub.Skill`

Skill 保存来源和整理信息。

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  | 主键 |
| `name` | `string` |  | 否 |  | Skill 名称 |
| `description` | `string` |  | 否 |  | 用途说明 |
| `category` | `string` |  | 否 |  | 分类 |
| `notes` | `string` |  | 否 |  | 个人备注 |
| `repositoryUrl` | `string` |  | 否 |  | GitHub 来源地址 |
| `repositoryPath` | `string` |  | 否 |  | 仓库内 Skill 路径 |
| `localPath` | `string` |  | 否 | '' | 本机原始 Skill 目录；GitHub 来源留空 |
| `sourceKey` | `string` |  | 否 | '' | 收录来源标识 |
| `importKind` | `string` |  | 否 | '' | folder、github 或 zip |
| `purpose` | `string` |  | 否 | '' | 用户整理的一行用途 |
| `upstreamName` | `string` |  | 否 | '' | 展示的上游名称 |
| `upstreamUrl` | `string` |  | 否 | '' | 展示的上游链接 |
| `defaultVersionId` | `string` |  | 否 | '' | 默认导出版本 |
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

SkillVersion 指向本机快照或 GitHub 的固定 commit。

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  | 主键 |
| `skillId` | `string` |  | 否 |  | Skill ID |
| `name` | `string` |  | 否 |  | 该版本的 Skill 名称 |
| `notes` | `string` |  | 否 |  | 版本说明 |
| `sourceRef` | `string` |  | 否 |  | 用户选择的 Git 引用 |
| `commitSha` | `string` |  | 否 |  | 完整 commit SHA |
| `contentDigest` | `string` |  | 否 | '' | 本地版本的内容摘要 |
| `description` | `string` |  | 否 | '' | 该版本的原始用途 |
| `upstreamCommit` | `string` |  | 否 | '' | 归档声明的上游 commit |
| `upstreamRepository` | `string` |  | 否 | '' | 归档声明的上游仓库 |
| `upstreamPath` | `string` |  | 否 | '' | 归档声明的仓库内路径 |
| `createTime` | `string` | `Timestamp` | 否 |  | 创建时间 |
| `updateTime` | `string` | `Timestamp` | 否 |  | 最后更新时间 |
