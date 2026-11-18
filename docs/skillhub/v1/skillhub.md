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
| `keyword` | `string` |  | 否 |  | 最大长度为 240 |
| `source` | `string` |  | 否 |  | 最大长度为 120 |
| `tags` | `Array<string>` |  | 否 |  | 最大长度为 30 |
| `uncategorized` | `boolean` |  | 否 |  |  |
| `category` | `string` |  | 否 |  | 最大长度为 128 |
| `pageSize` | `integer` | `Int32` | 否 |  | 最小值为 1；最大值为 100 |
| `pageToken` | `string` |  | 否 |  | 最大长度为 512 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skills` | `Array<skillhub.v1.SkillView>` |  | 否 |  |  |
| `nextPageToken` | `string` |  | 否 |  |  |


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


## Update Skill

### 请求路径

```http
PUT /skillhub/v1/skills/{skill_id}/metadata
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | 最大长度为 36；UUID |

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `category` | `string` |  | 否 |  | 最大长度为 128 |
| `tags` | `Array<string>` |  | 否 |  | 最大长度为 100 |
| `notes` | `string` |  | 否 |  | 最大长度为 4096 |
| `purpose` | `string` |  | 否 |  | 最大长度为 240 |
| `upstreamName` | `string` |  | 否 |  | 最大长度为 120 |
| `upstreamUrl` | `string` |  | 否 |  | 最大长度为 2048 |

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

### 返回值

#### 返回对象

对象为空


## Batch Delete Skills

### 请求路径

```http
POST /skillhub/v1/skills:batchDelete
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skillIds` | `Array<string>` |  | 是 |  | 最小长度为 1；最大长度为 100 |

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

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `sourceRef` | `string` |  | 否 |  | 最大长度为 200 |
| `commitSha` | `string` |  | 否 |  | 最小长度为 40；最大长度为 40 |
| `notes` | `string` |  | 否 |  | 最大长度为 4096 |
| `expectedDigest` | `string` |  | 否 |  | 本地版本要匹配比对时的原目录摘要。；最小长度为 64；最大长度为 64 |
| `localPath` | `string` |  | 否 |  | 最大长度为 2048 |
| `makeDefault` | `boolean` |  | 否 |  |  |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `version` | `skillhub.v1.VersionView` |  | 否 |  |  |


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
| `skills` | `Array<skillhub.v1.SkillCandidate>` |  | 否 |  |  |


## Preview Import

### 请求路径

```http
POST /skillhub/v1/imports:preview
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `source` | `skillhub.v1.GitHubSource` |  | 是 |  |  |
| `scanCollection` | `boolean` |  | 否 |  | 扫描仓库的 skills/ 集合。 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `resolvedCommitSha` | `string` |  | 否 |  |  |
| `skills` | `Array<skillhub.v1.SkillCandidate>` |  | 否 |  |  |


## Import Skill

### 请求路径

```http
POST /skillhub/v1/imports
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `repositoryUrl` | `string` |  | 否 |  | 本地收录时仓库字段留空，skill_paths 为本机绝对路径。；最大长度为 2048 |
| `sourceRef` | `string` |  | 否 |  | 最大长度为 200 |
| `commitSha` | `string` |  | 否 |  | 最小长度为 40；最大长度为 40 |
| `skillPaths` | `Array<string>` |  | 是 |  | 最大长度为 100 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `addedCount` | `integer` | `Int32` | 否 |  |  |
| `existingCount` | `integer` | `Int32` | 否 |  |  |
| `failures` | `Array<skillhub.v1.ImportFailure>` |  | 否 |  |  |


## Set Default Version

### 请求路径

```http
PUT /skillhub/v1/skills/{skill_id}/versions/{version_id}/default
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | UUID |
| `version_id` | `string` |  | UUID |

### 返回值

#### 返回对象

对象为空


## Delete Version

### 请求路径

```http
DELETE /skillhub/v1/skills/{skill_id}/versions/{version_id}
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | UUID |
| `version_id` | `string` |  | UUID |

### 返回值

#### 返回对象

对象为空


## Get Version Content

### 请求路径

```http
GET /skillhub/v1/skills/{skill_id}/versions/{version_id}/content
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | UUID |
| `version_id` | `string` |  | UUID |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `description` | `string` |  | 否 |  |  |
| `files` | `Array<string>` |  | 否 |  |  |


## Get Version File

### 请求路径

```http
GET /skillhub/v1/skills/{skill_id}/versions/{version_id}/file
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | UUID |
| `version_id` | `string` |  | UUID |

#### Query 参数

| 参数名 | 参数类型 | 格式类型 | 是否必须 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `path` | `string` |  | 是 |  | 最大长度为 2048 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `text` | `string` |  | 否 |  |  |
| `size` | `integer` | `Int32` | 否 |  |  |
| `binary` | `boolean` |  | 否 |  |  |
| `tooLarge` | `boolean` |  | 否 |  |  |


## Download Zip

### 请求路径

```http
GET /skillhub/v1/skills/{skill_id}/versions/{version_id}/archive
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | UUID |
| `version_id` | `string` |  | UUID |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `archive` | `google.api.HttpBody` |  | 否 |  |  |
| `filename` | `string` |  | 否 |  |  |


## Select Local Directory

### 请求路径

```http
POST /skillhub/v1/directories:select
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `path` | `string` |  | 否 |  |  |


## Export Version To Directory

### 请求路径

```http
POST /skillhub/v1/skills/{skill_id}/versions/{version_id}:export
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | UUID |
| `version_id` | `string` |  | UUID |

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `directory` | `string` |  | 否 |  | 最大长度为 2048 |
| `chooseDirectory` | `boolean` |  | 否 |  |  |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `path` | `string` |  | 否 |  |  |


## Repair Version

### 请求路径

```http
POST /skillhub/v1/skills/{skill_id}/versions/{version_id}:repair
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `skill_id` | `string` |  | UUID |
| `version_id` | `string` |  | UUID |

### 返回值

#### 返回对象

对象为空


## List Facets

### 请求路径

```http
GET /skillhub/v1/facets
```

### 请求参数

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `categories` | `Array<string>` |  | 否 |  |  |
| `sources` | `Array<string>` |  | 否 |  |  |
| `tags` | `Array<string>` |  | 否 |  |  |


## Batch Update Skill Metadata

### 请求路径

```http
PATCH /skillhub/v1/skills:organize
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `changes` | `Array<skillhub.v1.SkillMetadataChange>` |  | 是 |  | 最大长度为 100 |

### 返回值

#### 返回对象

对象为空


## Upload Zip

### 请求路径

```http
POST /skillhub/v1/uploads
```

### 请求参数

#### Body 请求对象

| type | description |
|---|---|
| `string` |  |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `uploadId` | `string` |  | 否 |  |  |
| `skills` | `Array<skillhub.v1.SkillCandidate>` |  | 否 |  |  |


## Commit Upload

### 请求路径

```http
POST /skillhub/v1/uploads/{upload_id}:commit
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `upload_id` | `string` |  | UUID |

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skillPaths` | `Array<string>` |  | 是 |  | 最大长度为 100 |
| `targetSkillId` | `string` |  | 否 |  | UUID |
| `makeDefault` | `boolean` |  | 否 |  |  |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `addedCount` | `integer` | `Int32` | 否 |  |  |
| `existingCount` | `integer` | `Int32` | 否 |  |  |
| `failures` | `Array<skillhub.v1.ImportFailure>` |  | 否 |  |  |


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


## Get Install Command

### 请求路径

```http
POST /skillhub/v1/install-commands:generate
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skillId` | `string` |  | 是 |  | 最大长度为 36；UUID |
| `versionId` | `string` |  | 是 |  | 最大长度为 36；UUID |
| `global` | `boolean` |  | 否 |  |  |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `command` | `string` |  | 否 |  |  |


## 数据模型

### `skillhub.v1.SkillView`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  |  |
| `name` | `string` |  | 否 |  |  |
| `description` | `string` |  | 否 |  |  |
| `category` | `string` |  | 否 |  |  |
| `tags` | `Array<string>` |  | 否 |  |  |
| `notes` | `string` |  | 否 |  |  |
| `repositoryUrl` | `string` |  | 否 |  |  |
| `repositoryPath` | `string` |  | 否 |  |  |
| `localPath` | `string` |  | 否 |  |  |
| `importKind` | `string` |  | 否 |  |  |
| `purpose` | `string` |  | 否 |  |  |
| `upstreamName` | `string` |  | 否 |  |  |
| `upstreamUrl` | `string` |  | 否 |  |  |
| `defaultVersion` | `skillhub.v1.VersionView` |  | 否 |  |  |
| `createTime` | `string` | `Timestamp` | 否 |  |  |
| `updateTime` | `string` | `Timestamp` | 否 |  |  |

### `skillhub.v1.VersionView`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  |  |
| `skillId` | `string` |  | 否 |  |  |
| `name` | `string` |  | 否 |  |  |
| `notes` | `string` |  | 否 |  |  |
| `sourceRef` | `string` |  | 否 |  |  |
| `commitSha` | `string` |  | 否 |  |  |
| `contentDigest` | `string` |  | 否 |  |  |
| `available` | `boolean` |  | 否 |  |  |
| `upstreamCommit` | `string` |  | 否 |  |  |
| `upstreamRepository` | `string` |  | 否 |  |  |
| `upstreamPath` | `string` |  | 否 |  |  |
| `createTime` | `string` | `Timestamp` | 否 |  |  |

### `skillhub.v1.SkillCandidate`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `path` | `string` |  | 否 |  |  |
| `name` | `string` |  | 否 |  |  |
| `description` | `string` |  | 否 |  |  |

### `skillhub.v1.GitHubSource`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `repositoryUrl` | `string` |  | 否 |  | 必须是 GitHub HTTPS 仓库地址。 |
| `skillPath` | `string` |  | 否 |  | 空路径表示 Skill 位于仓库根目录。 |
| `ref` | `string` |  | 否 |  |  |

### `skillhub.v1.ImportFailure`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `path` | `string` |  | 否 |  |  |
| `message` | `string` |  | 否 |  |  |

### `google.api.HttpBody`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `contentType` | `string` |  | 否 |  |  |
| `data` | `string` | `Bytes` | 否 |  |  |
| `extensions` | `Array<google.protobuf.Any>` |  | 否 |  |  |

### `google.protobuf.Any`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `typeUrl` | `string` |  | 否 |  |  |
| `value` | `string` | `Bytes` | 否 |  |  |

### `skillhub.v1.SkillMetadataChange`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `skillId` | `string` |  | 否 |  |  |
| `purpose` | `string` |  | 否 |  |  |
| `category` | `string` |  | 否 |  |  |
| `addTags` | `Array<string>` |  | 否 |  |  |
| `removeTags` | `Array<string>` |  | 否 |  |  |
| `revision` | `string` |  | 否 |  |  |

### `skillhub.v1.SkillLocation`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `currentSkillId` | `string` |  | 否 |  | 应用层校验恰好指定一种来源。 |
| `versionId` | `string` |  | 否 |  |  |
| `localPath` | `string` |  | 否 |  |  |
| `github` | `skillhub.v1.GitHubSource` |  | 否 |  |  |

### `skillhub.v1.FileChangeView`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `path` | `string` |  | 否 |  |  |
| `kind` | `skillhub.v1.FileChangeKind` |  | 否 |  |  |
| `binary` | `boolean` |  | 否 |  |  |
| `leftText` | `string` |  | 否 |  |  |
| `rightText` | `string` |  | 否 |  |  |
| `detail` | `string` |  | 否 |  |  |


## 枚举

<a id="skillhub-v1-filechangekind"></a>

### `skillhub.v1.FileChangeKind`

| 枚举值 | 数值 | 说明 |
|---|---:|---|
| `FILE_CHANGE_KIND_UNSPECIFIED` | 0 |  |
| `FILE_CHANGE_KIND_ADDED` | 1 |  |
| `FILE_CHANGE_KIND_MODIFIED` | 2 |  |
| `FILE_CHANGE_KIND_DELETED` | 3 |  |
