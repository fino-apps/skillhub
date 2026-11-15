# Skillhub Service

服务：`skillhub.v1.SkillhubService`

## HTTP 约定

- 请求 Body、Query 参数和响应数据使用 JSON。
- 响应 `Content-Type` 为 `application/json; charset=utf-8`。
- Body、Query 参数和响应字段名使用 `lowerCamelCase`；Path 参数以 URI 模板中的名称为准。
- `FieldMask` 使用逗号分隔的 `lowerCamelCase` 字段路径，例如 `displayName,description`。
- `Timestamp` 字段使用带时区的 RFC3339 字符串，例如 `2026-08-12T10:30:00.000+08:00`。

## 错误响应

默认 HTTP 模式下，错误响应格式为：

```json
{
  "code": "100060400",
  "message": "bad request"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `code` | string | fino 业务错误码 |
| `message` | string | 错误说明 |
| `details` | array | 可选的错误详情 |

HTTP 状态码与业务错误码分别表达传输状态和业务错误。

## List Skills

### 请求路径

```http
GET /skillhub/v1/skills
```

### 请求参数

#### Query 参数

| 参数名 | 参数类型 | 格式类型 | 是否必须 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `keyword` | `string` |  | 否 |  | 最大长度为 1024 |
| `kind` | `skillhub.SkillKind` |  | 否 |  |  |
| `category` | `string` |  | 否 |  | 最大长度为 128 |
| `tag` | `string` |  | 否 |  | 最大长度为 128 |
| `pageSize` | `integer` | `Int32` | 否 |  | 最小值为 1；最大值为 100 |
| `pageToken` | `string` |  | 否 |  | 最大长度为 512 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skills` | `Array<skillhub.v1.SkillView>` |  | 否 |  |  |
| `nextPageToken` | `string` |  | 否 |  |  |


## List Skill Filters

### 请求路径

```http
GET /skillhub/v1/skill-filters
```

### 请求参数

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `categories` | `Array<string>` |  | 否 |  |  |
| `tags` | `Array<string>` |  | 否 |  |  |


## Get Skill

### 请求路径

```http
GET /skillhub/v1/skills/{skill_id}
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | 最大长度为 36；UUID |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skill` | `skillhub.v1.SkillView` |  | 否 |  |  |


## Create Skill

### 请求路径

```http
POST /skillhub/v1/skills
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `name` | `string` |  | 是 |  | 最大长度为 64 |
| `description` | `string` |  | 是 |  | 最大长度为 1024 |
| `kind` | `skillhub.SkillKind` |  | 否 |  |  |
| `category` | `string` |  | 否 |  | 最大长度为 128 |
| `tags` | `Array<string>` |  | 否 |  | 最大长度为 100 |
| `notes` | `string` |  | 否 |  | 最大长度为 4096 |
| `repositoryUrl` | `string` |  | 否 |  | 最大长度为 2048 |
| `repositoryPath` | `string` |  | 否 |  | 最大长度为 1024 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skill` | `skillhub.v1.SkillView` |  | 否 |  |  |


## Update Skill

### 请求路径

```http
PATCH /skillhub/v1/skills/{skill_id}
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | 最大长度为 36；UUID |

#### Query 参数

| 参数名 | 参数类型 | 格式类型 | 是否必须 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `updateMask` | `string` | `FieldMask` | 否 |  |  |

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `name` | `string` |  | 否 |  | 最大长度为 64 |
| `description` | `string` |  | 否 |  | 最大长度为 1024 |
| `category` | `string` |  | 否 |  | 最大长度为 128 |
| `tags` | `Array<string>` |  | 否 |  | 最大长度为 100 |
| `notes` | `string` |  | 否 |  | 最大长度为 4096 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skill` | `skillhub.v1.SkillView` |  | 否 |  |  |


## Delete Skill

### 请求路径

```http
DELETE /skillhub/v1/skills/{skill_id}
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | 最大长度为 36；UUID |

#### Query 参数

| 参数名 | 参数类型 | 格式类型 | 是否必须 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `deleteContent` | `boolean` |  | 否 |  | 显式选择是否删除个人内容和快照；公开 Skill 只删除记录。 |

### 返回值

#### 返回对象

对象为空


## List Versions

### 请求路径

```http
GET /skillhub/v1/skills/{skill_id}/versions
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | 最大长度为 36；UUID |

#### Query 参数

| 参数名 | 参数类型 | 格式类型 | 是否必须 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `pageSize` | `integer` | `Int32` | 否 |  | 最小值为 1；最大值为 100 |
| `pageToken` | `string` |  | 否 |  | 最大长度为 512 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `versions` | `Array<skillhub.v1.VersionView>` |  | 否 |  |  |
| `nextPageToken` | `string` |  | 否 |  |  |


## Create Version

### 请求路径

```http
POST /skillhub/v1/skills/{skill_id}/versions
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | 最大长度为 36；UUID |

#### Query 参数

| 参数名 | 参数类型 | 格式类型 | 是否必须 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `expectedDigest` | `string` |  | 否 |  | 个人快照要求匹配当前目录摘要；公开 Skill 留空。；最大长度为 128 |

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `version` | `string` |  | 是 |  | 最大长度为 128 |
| `notes` | `string` |  | 否 |  | 最大长度为 4096 |
| `sourceRef` | `string` |  | 否 |  | 公开 Skill 使用完整 commit，个人 Skill 创建当前内容快照。；最大长度为 512 |
| `commitSha` | `string` |  | 否 |  | 最大长度为 64 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `version` | `skillhub.v1.VersionView` |  | 否 |  |  |


## Restore Version

### 请求路径

```http
POST /skillhub/v1/skills/{skill_id}/versions/{version_id}:restore
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | 最大长度为 36；UUID |
| `version_id` | `string` |  | 最大长度为 36；UUID |

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `expectedDigest` | `string` |  | 是 |  | 必须匹配当前内容摘要，防止预览后出现新的修改。；最大长度为 128 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skill` | `skillhub.v1.SkillView` |  | 否 |  |  |
| `backupVersion` | `skillhub.v1.VersionView` |  | 否 |  |  |


## List Files

### 请求路径

```http
POST /skillhub/v1/files:list
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `location` | `skillhub.v1.SkillLocation` |  | 是 |  |  |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `files` | `Array<skillhub.v1.FileView>` |  | 否 |  |  |
| `digest` | `string` |  | 否 |  |  |
| `resolvedCommitSha` | `string` |  | 否 |  |  |


## Get File

### 请求路径

```http
POST /skillhub/v1/files:get
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `location` | `skillhub.v1.SkillLocation` |  | 是 |  |  |
| `path` | `string` |  | 是 |  | 最大长度为 1024 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `file` | `skillhub.v1.FileView` |  | 否 |  |  |
| `text` | `string` |  | 否 |  |  |
| `resolvedCommitSha` | `string` |  | 否 |  |  |


## Put File

### 请求路径

```http
PUT /skillhub/v1/skills/{skill_id}/files
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | 最大长度为 36；UUID |

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `path` | `string` |  | 是 |  | 最大长度为 1024 |
| `text` | `string` |  | 否 |  |  |
| `expectedDigest` | `string` |  | 是 |  | 编辑前个人 Skill 整个目录的摘要；新建文件同样需要。；最大长度为 128 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `file` | `skillhub.v1.FileView` |  | 否 |  |  |
| `currentDigest` | `string` |  | 否 |  |  |


## Delete File

### 请求路径

```http
DELETE /skillhub/v1/skills/{skill_id}/files
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | 最大长度为 36；UUID |

#### Query 参数

| 参数名 | 参数类型 | 格式类型 | 是否必须 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `path` | `string` |  | 是 |  | 最大长度为 1024 |
| `expectedDigest` | `string` |  | 是 |  | 删除前个人 Skill 整个目录的摘要。；最大长度为 128 |

### 返回值

#### 返回对象

对象为空


## Scan Project

### 请求路径

```http
POST /skillhub/v1/projects:scan
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `projectPath` | `string` |  | 是 |  | 最大长度为 2048 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skills` | `Array<skillhub.v1.ProjectSkillView>` |  | 否 |  |  |


## Preview Import

### 请求路径

```http
POST /skillhub/v1/imports:preview
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `source` | `skillhub.v1.SkillLocation` |  | 是 |  |  |
| `targetSkillId` | `string` |  | 否 |  | 最大长度为 36；UUID |
| `projectPath` | `string` |  | 否 |  | 最大长度为 2048 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `files` | `Array<skillhub.v1.FileView>` |  | 否 |  |  |
| `sourceDigest` | `string` |  | 否 |  | 预览时完整源目录及目标当前目录的摘要。 |
| `targetDigest` | `string` |  | 否 |  |  |
| `resolvedCommitSha` | `string` |  | 否 |  |  |
| `status` | `skillhub.v1.ProjectSkillStatus` |  | 否 |  |  |


## Import Skill

### 请求路径

```http
POST /skillhub/v1/imports
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `source` | `skillhub.v1.SkillLocation` |  | 是 |  |  |
| `targetSkillId` | `string` |  | 否 |  | 最大长度为 36；UUID |
| `projectPath` | `string` |  | 否 |  | 最大长度为 2048 |
| `expectedSourceDigest` | `string` |  | 是 |  | 匹配预览时的完整源目录摘要；导入已有 Skill 时也须匹配目标摘要。；最大长度为 128 |
| `expectedTargetDigest` | `string` |  | 否 |  | 最大长度为 128 |
| `includedFiles` | `Array<string>` |  | 是 |  | 明确列出要复制的相对文件路径，必须包含 SKILL.md。 |
| `version` | `skillhub.v1.VersionInput` |  | 是 |  |  |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skill` | `skillhub.v1.SkillView` |  | 否 |  |  |
| `version` | `skillhub.v1.VersionView` |  | 否 |  |  |


## Resolve Source

### 请求路径

```http
POST /skillhub/v1/sources:resolve
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `source` | `skillhub.v1.GitHubSource` |  | 是 |  |  |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `commitSha` | `string` |  | 否 |  |  |
| `name` | `string` |  | 否 |  |  |
| `description` | `string` |  | 否 |  |  |
| `digest` | `string` |  | 否 |  |  |


## Compare Skills

### 请求路径

```http
POST /skillhub/v1/skills:compare
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `left` | `skillhub.v1.SkillLocation` |  | 是 |  |  |
| `right` | `skillhub.v1.SkillLocation` |  | 是 |  |  |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `files` | `Array<skillhub.v1.FileChangeView>` |  | 否 |  |  |
| `leftDigest` | `string` |  | 否 |  |  |
| `rightDigest` | `string` |  | 否 |  |  |
| `leftCommitSha` | `string` |  | 否 |  |  |
| `rightCommitSha` | `string` |  | 否 |  |  |


## Get Install Commands

### 请求路径

```http
POST /skillhub/v1/install-commands:generate
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skills` | `Array<skillhub.v1.InstallSelection>` |  | 是 |  | 最大长度为 100 |
| `projectPath` | `string` |  | 否 |  | 项目级安装命令须在此项目中执行；全局安装时可留空。；最大长度为 2048 |
| `global` | `boolean` |  | 否 |  |  |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `commands` | `Array<string>` |  | 否 |  |  |


## 数据模型

### `skillhub.v1.SkillView`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  |  |
| `name` | `string` |  | 否 |  |  |
| `description` | `string` |  | 否 |  |  |
| `kind` | `skillhub.SkillKind` |  | 否 |  |  |
| `category` | `string` |  | 否 |  |  |
| `tags` | `Array<string>` |  | 否 |  |  |
| `notes` | `string` |  | 否 |  |  |
| `repositoryUrl` | `string` |  | 否 |  |  |
| `repositoryPath` | `string` |  | 否 |  |  |
| `currentDigest` | `string` |  | 否 |  |  |
| `originCommitSha` | `string` |  | 否 |  |  |
| `createTime` | `string` | `Timestamp` | 否 |  |  |
| `updateTime` | `string` | `Timestamp` | 否 |  |  |

### `skillhub.v1.VersionView`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  |  |
| `skillId` | `string` |  | 否 |  |  |
| `version` | `string` |  | 否 |  |  |
| `notes` | `string` |  | 否 |  |  |
| `digest` | `string` |  | 否 |  |  |
| `sourceRef` | `string` |  | 否 |  |  |
| `commitSha` | `string` |  | 否 |  |  |
| `createTime` | `string` | `Timestamp` | 否 |  |  |

### `skillhub.v1.SkillLocation`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `currentSkillId` | `string` |  | 否 |  | 应用层校验恰好指定一种来源。 |
| `versionId` | `string` |  | 否 |  |  |
| `localPath` | `string` |  | 否 |  |  |
| `github` | `skillhub.v1.GitHubSource` |  | 否 |  |  |

### `skillhub.v1.GitHubSource`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `repositoryUrl` | `string` |  | 是 |  | 最大长度为 2048 |
| `skillPath` | `string` |  | 否 |  | 空路径表示 Skill 位于仓库根目录。；最大长度为 1024 |
| `ref` | `string` |  | 是 |  | 最大长度为 512 |

### `skillhub.v1.FileView`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `path` | `string` |  | 否 |  |  |
| `size` | `integer` | `Int64` | 否 |  |  |
| `binary` | `boolean` |  | 否 |  |  |
| `digest` | `string` |  | 否 |  |  |
| `excluded` | `boolean` |  | 否 |  |  |
| `sensitive` | `boolean` |  | 否 |  |  |

### `skillhub.v1.ProjectSkillView`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `path` | `string` |  | 否 |  |  |
| `name` | `string` |  | 否 |  |  |
| `description` | `string` |  | 否 |  |  |
| `digest` | `string` |  | 否 |  |  |
| `skillId` | `string` |  | 否 |  |  |
| `baseVersionId` | `string` |  | 否 |  |  |
| `baseDigest` | `string` |  | 否 |  |  |
| `status` | `skillhub.v1.ProjectSkillStatus` |  | 否 |  |  |

### `skillhub.v1.VersionInput`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `version` | `string` |  | 是 |  | 最大长度为 128 |
| `notes` | `string` |  | 否 |  | 最大长度为 4096 |
| `sourceRef` | `string` |  | 否 |  | 公开 Skill 使用完整 commit，个人 Skill 创建当前内容快照。；最大长度为 512 |
| `commitSha` | `string` |  | 否 |  | 最大长度为 64 |

### `skillhub.v1.FileChangeView`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `path` | `string` |  | 否 |  |  |
| `kind` | `skillhub.v1.FileChangeKind` |  | 否 |  |  |
| `binary` | `boolean` |  | 否 |  |  |
| `leftText` | `string` |  | 否 |  |  |
| `rightText` | `string` |  | 否 |  |  |

### `skillhub.v1.InstallSelection`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skillId` | `string` |  | 否 |  |  |
| `versionId` | `string` |  | 否 |  |  |


## 枚举

<a id="skillhub-skillkind"></a>

### `skillhub.SkillKind`

SkillKind 区分由管理库维护的 Skill 和只记录来源的公开 Skill。

| 枚举值 | 数值 | 说明 |
|---|---:|---|
| `SKILL_KIND_UNSPECIFIED` | 0 |  |
| `SKILL_KIND_PERSONAL` | 1 |  |
| `SKILL_KIND_PUBLIC` | 2 |  |

<a id="skillhub-v1-projectskillstatus"></a>

### `skillhub.v1.ProjectSkillStatus`

| 枚举值 | 数值 | 说明 |
|---|---:|---|
| `PROJECT_SKILL_STATUS_UNSPECIFIED` | 0 |  |
| `PROJECT_SKILL_STATUS_NEW` | 1 |  |
| `PROJECT_SKILL_STATUS_UNCHANGED` | 2 |  |
| `PROJECT_SKILL_STATUS_PROJECT_CHANGED` | 3 |  |
| `PROJECT_SKILL_STATUS_LIBRARY_CHANGED` | 4 |  |
| `PROJECT_SKILL_STATUS_CONFLICT` | 5 |  |
| `PROJECT_SKILL_STATUS_NO_BASELINE` | 6 |  |

<a id="skillhub-v1-filechangekind"></a>

### `skillhub.v1.FileChangeKind`

| 枚举值 | 数值 | 说明 |
|---|---:|---|
| `FILE_CHANGE_KIND_UNSPECIFIED` | 0 |  |
| `FILE_CHANGE_KIND_ADDED` | 1 |  |
| `FILE_CHANGE_KIND_MODIFIED` | 2 |  |
| `FILE_CHANGE_KIND_DELETED` | 3 |  |
