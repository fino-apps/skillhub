# 项目协作指南

[README.md](README.md) 介绍如何运行项目、调用接口和重新生成。[Fino skill](skills/fino/SKILL.md) 记录生成目标、扩展点和验证约定。在这里维护本项目特有的协作要求。

## 文件归属

- API 和模型以 `proto/` 为准。包名、模块路径和显式依赖版本维护在 `package.yaml`，服务与基础设施选项维护在 `fino.yaml`。
- 修改生成结果时，先改对应的契约、配置或 Fino 模板，再重新生成。保留 `DO NOT EDIT` 文件和 `.fino/ir/` 的生成归属。
- 在项目扩展点实现业务规则。修改可合并的文件时保留生成标记，重建后确认手写代码仍在。新增自定义 RPC 后补齐业务实现。修改标准 CRUD 或自定义 RPC 时，核对请求字段在相关层的映射，并验证受影响的筛选、更新掩码等行为。
- `README.md`、`AGENTS.md` 和 `.fino/er.yaml` 由项目维护。契约或配置变化时同步检查 README、请求示例和其他受影响的项目文件。

## 生成与验证

- 根据改动选择已有且受影响的目标。在项目根目录运行 `fino build --output=. --targets=<targets>`，检查生成差异和项目扩展。修改环境或生成目标时，先用相同的 `--output`、`--targets` 运行 `fino doctor`。
- 按改动验证受影响行为。API 和后端分别在 `go/`、`backend/` 下运行 `go test ./...`。前端在 `frontend/` 下运行 `npm run build`，首次生成或干净检出时先执行 `npm ci`。启动装配或运行配置变化时，实际启动对应服务并验证受影响的调用。
