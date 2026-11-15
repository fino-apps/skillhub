## `skillhub.Skill`

Skill 保存管理信息；个人内容位于受控的相对目录。

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  | 主键 |
| `name` | `string` |  | 否 |  | Skill 名称 |
| `description` | `string` |  | 否 |  | 用途说明 |
| `kind` | [`skillhub.SkillKind`](skill.md#skillhub-skillkind) |  | 否 |  | 来源类型 |
| `category` | `string` |  | 否 |  | 分类 |
| `notes` | `string` |  | 否 |  | 个人备注 |
| `contentPath` | `string` |  | 否 |  | 个人内容在数据目录内的相对路径 |
| `repositoryUrl` | `string` |  | 否 |  | GitHub 来源地址 |
| `repositoryPath` | `string` |  | 否 |  | 仓库内 Skill 路径 |
| `currentDigest` | `string` |  | 否 |  | 个人当前内容摘要 |
| `originCommitSha` | `string` |  | 否 |  | 复制为个人 Skill 时的来源 commit |
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

SkillVersion 指向个人不可变快照或公开来源的固定 commit。

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  | 主键 |
| `skillId` | `string` |  | 否 |  | Skill ID |
| `version` | `string` |  | 否 |  | 版本标识 |
| `notes` | `string` |  | 否 |  | 版本说明 |
| `digest` | `string` |  | 否 |  | 目录内容摘要 |
| `snapshotPath` | `string` |  | 否 |  | 个人快照在数据目录内的相对路径 |
| `sourceRef` | `string` |  | 否 |  | 公开来源的分支或 tag |
| `commitSha` | `string` |  | 否 |  | 公开来源解析出的完整 commit |
| `createTime` | `string` | `Timestamp` | 否 |  | 创建时间 |
| `updateTime` | `string` | `Timestamp` | 否 |  | 最后更新时间 |


## `skillhub.ProjectSkill`

ProjectSkill 保存业务项目与上次导入的共同基准。

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  | 主键 |
| `projectPath` | `string` |  | 否 |  | 本机项目绝对路径 |
| `skillPath` | `string` |  | 否 |  | Skill 相对项目的路径 |
| `skillId` | `string` |  | 否 |  | 管理库 Skill ID |
| `baseVersionId` | `string` |  | 否 |  | 上次导入的基准版本 ID |
| `baseDigest` | `string` |  | 否 |  | 上次导入的内容摘要 |
| `createTime` | `string` | `Timestamp` | 否 |  | 创建时间 |
| `updateTime` | `string` | `Timestamp` | 否 |  | 最后更新时间 |


<a id="skillhub-skillkind"></a>

## `skillhub.SkillKind`

SkillKind 区分由管理库维护的 Skill 和只记录来源的公开 Skill。

| 枚举值 | 数值 | 说明 |
|---|---:|---|
| `SKILL_KIND_UNSPECIFIED` | 0 |  |
| `SKILL_KIND_PERSONAL` | 1 |  |
| `SKILL_KIND_PUBLIC` | 2 |  |
