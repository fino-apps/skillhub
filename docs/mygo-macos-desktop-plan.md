# SkillHub macOS 桌面版实施计划

目标是在当前 Intel Mac（`x86_64`、macOS `13.7.8`）上，从 Finder 打开 `SkillHub.app` 即可使用技能库。应用随包提供前端和后端，支持离线浏览已保存版本、目录及 ZIP 收录、版本比对、导出和整库备份。GitHub 收录和更新沿用本机 Git，需要联网。首版验收产物为 ad hoc 签名的 `darwin/amd64` `.app`。

## 架构

```text
SkillHub.app（一个 Go 进程）
  MyGo / WKWebView → 127.0.0.1 随机端口
                     ├─ 页面和静态资源 → 内嵌的 React/Vite 构建产物
                     └─ /skillhub/v1 → Fino HTTP Handler → 现有业务层 → SQLite、快照
```

- **入口与装配**。以 `backend/cmd/skillhub-server` 为 MyGo 构建入口。用 `skillhub_desktop` 构建标签为手写的 `loadConfig` 和 `serve` 提供桌面实现，将 `buildServer` 移到同包共享文件。复用现有 `main.go`、`newInfra`、`migrateSchema`、`buildServer` 和生成的 `http.NewHandler`。参照 `api/run.go` 保留 endpoint 中间件。桌面入口只启动 HTTP，使用内置本地配置；普通构建继续使用现行服务装配。业务仍通过现有 Proto 和 HTTP 客户端调用。
- **页面与会话**。用 `go:embed` 打包桌面 Vite 产物，让页面、API、附件下载使用同一个 HTTP origin。SPA 回退只处理前端页面路径。监听 `127.0.0.1:0`，窗口用一次性启动凭证建立 `HttpOnly`、`SameSite=Strict` 会话，服务端校验 `Host`，拒绝跨源 API 请求；外链交由系统浏览器。退出时先关闭 HTTP，再关闭数据库。使用用户数据目录的进程锁，避免重复启动同时写库。
- **数据**。将数据库、快照、上传暂存和默认导出目录放在 `~/Library/Application Support/SkillHub/`，日志放在 `~/Library/Logs/SkillHub/`。路径在启动时解析为绝对路径，保留 SQLite 的 WAL、忙等待和外键设置。已有 Web/Docker 数据通过整库备份 ZIP 显式导入；迁移验收使用副本。应用替换或移动后仍打开同一份用户数据。
- **桌面交互**。先启用现有 `VITE_DESKTOP_DIRECTORY_PICKER=true` 和 macOS `osascript` 目录选择，沿用现有扫描和导出流程。实际验证 Finder 启动时的选择、取消和权限失败，以及 Git 的可用性和错误提示。桌面版使用目录导出或 ZIP 下载安装 Skill，隐藏依赖可访问服务地址的安装命令。随机端口使 Web 存储随 origin 变化，桌面首版的主题跟随系统，技能库数据由 SQLite 持久化。

## 实施顺序

1. **跑通最小包**。在临时项目用同一套 MyGo 库和 CLI 验证 Intel `.app`，从 Finder 启动，打开一页回环 HTTP，并通过 `CGO_ENABLED=1` 实际写入、读回一条 SQLite 记录。采用 `egoist/mygo v0.10.0` 作为待验证版本，其 Go 模块要求 `1.27.3`，Go 1.27 支持 Ventura。本机后端当前能解析到 Go `1.25.13`，桌面构建先单独准备 `1.27.3`。本机已有 Node `20.20.2` 和 Xcode Command Line Tools。检查 Mach-O 的架构和最低系统版本，跑通后再接项目。 [MyGo 版本](https://github.com/egoist/mygo/releases/tag/v0.10.0) · [模块 Go 版本](https://raw.githubusercontent.com/egoist/mygo/v0.10.0/go.mod) · [Go 1.27 系统要求](https://go.dev/doc/go1.27#darwin) · [MyGo CGO 构建说明](https://mygo.run/docs/distribution#platforms)
2. **接入真实业务**。按上述边界拆分手写启动代码，加入桌面配置、同源路由和退出处理。先用空库跑通列表，再用备份副本验证恢复及重启。MyGo 依赖版本同步到 `package.yaml`，确认服务重新生成后仍保留桌面扩展点，检查后端 Go 版本调整对现有 Docker 构建的影响。Proto、生成客户端和生成文件继续由 Fino 管理。
3. **固定构建入口并打包**。以 `backend/mygo.json` 固定应用 ID、最低系统版本 `13.0`、Go 入口 `./cmd/skillhub-server` 和输出目录 `dist`。仓库根目录提供一个构建脚本，用 `npm ci` 安装依赖并构建现有前端，设置空的 `VITE_API_BASE` 以使用同源请求，产物写入专用、已忽略的嵌入目录。脚本从 `backend/` 调用打包命令，生成 `backend/dist/darwin-amd64/SkillHub.app`。保留仓库现有 Linux 交付流程，桌面包可选生成 DMG。 [MyGo 项目配置](https://mygo.run/docs/configuration) · [MyGo CLI](https://mygo.run/docs/cli)

```sh
GOTOOLCHAIN=go1.27.3 go install github.com/egoist/mygo/cmd/mygo@v0.10.0
GOTOOLCHAIN=go1.27.3 CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=13.0 \
  GOFLAGS='-tags=skillhub_desktop' \
  mygo build -platform darwin/amd64 -skip-dmg
```

## 本机验收

| 检查 | 通过标准 |
| --- | --- |
| 构建与启动 | `file` 显示 `x86_64`，`otool -l` 显示最低系统版本不高于 `13.7.8`，`codesign --verify --strict` 通过。从 Finder 打开应用，在没有运行 Web 服务或 Docker 的情况下看到技能列表。 |
| 核心操作 | 导入本机目录和 ZIP，打开详情并刷新页面，新增版本后比对。验证目录导出、ZIP 下载的文件名和内容、备份导出与恢复。联网验证一次 GitHub 收录，断网后可浏览并导出已保存版本。 |
| WKWebView | 实测文件选择、下载、剪贴板和外链。当前机器 Safari 为 `18.6`，仍以应用内 WKWebView 为准。构建时显式声明经实测的 WebKit 兼容目标，并检查 `URL.canParse` 等 Vite 不负责补齐的运行时 API。 [WebKit 兼容记录](https://webkit.org/blog/14445/webkit-features-in-safari-17-0/) |
| 退出与数据 | 启动第二份应用时只有一个进程持有库；退出后回环端口释放。重新打开或替换 `.app` 后，元数据、默认版本和快照仍可用。未建立会话的请求无法读取或修改库。 |
| 项目回归 | 检查 `fino doctor/build --output=. --targets=service` 及生成差异，运行 API、后端 `go test ./...` 和前端 `npm run build`，验证普通 Web 服务可启动并读取技能列表。 |
