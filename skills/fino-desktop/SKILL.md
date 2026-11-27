---
name: fino-desktop
description: 将已有 Fino Go 服务和 Vite 前端封装为 macOS 桌面应用，或维护其 MyGo 启动、嵌入资源、原生交互、应用图标及 .app/.dmg 构建。用于 Fino 桌面接入、打包和排障。
---

# Fino Desktop

用 MyGo 提供原生窗口，在同一 Go 进程内运行 Fino 服务并提供内嵌的 Vite 页面。先查项目现有桌面入口、Go 和 npm 依赖、HTTP 路由及打包配置，沿用已有平台和发布范围。涉及契约、生成目标或模块依赖时，参照项目根目录下的 `skills/fino/SKILL.md`。

## 接入 Fino

- 保留生成的服务入口、HTTP Handler、endpoint 中间件及业务实现。在 `backend/cmd/<service>-server/` 保留生成的 `main.go`，把 `buildServer` 等共用装配放在无标签的手写文件。桌面与服务器文件分别实现 `loadConfig` 和 `serve`，使用互补标签 `darwin && <app>_desktop` 和 `!darwin || !<app>_desktop`。
- 把桌面版前端生产产物写到 Go 入口包的专用目录（如 `desktop-assets/`），用 `go:embed all:desktop-assets` 和 `fs.Sub` 提供静态资源。桌面前端使用相对 API 地址，原生功能通过现有前端配置按需开启。先产出静态资源，再编译带桌面标签的 Go 包。
- 监听 `127.0.0.1:0`，用监听器的实际地址让窗口加载页面。按项目 API 路由前缀转发给 Fino Handler（有多个前缀时逐一覆盖），SPA 回退只处理页面导航。复用现有本地 HTTP 保护，确保启动凭证仅可用一次、会话 Cookie 为 `HttpOnly` 和 `SameSite=Strict`，校验 Host 和跨源写请求。用户主动打开的外部 HTTP(S) 链接交给系统浏览器。
- 先确定应用是否需要离线运行，梳理配置中心、数据库、消息和下游 RPC 依赖。按运行方式选用本地配置与存储或保留必要的远程依赖。用 MyGo 的单实例、数据目录、日志目录和对话框能力。拿到单实例锁后再打开本地数据库，服务就绪后创建窗口，退出时先关闭 HTTP 再释放数据库。随包配置使用绝对数据路径，从 Finder 启动时也能定位数据库和其他持久化文件。
- 显式锁定 MyGo 时在 `package.yaml` 声明依赖，打包脚本与后端使用同一版本。调整契约或生成依赖时，对已有且受影响的目标执行 `fino build`，环境或生成目标变化前用相同参数运行 `fino doctor`。仅改图标、桌面启动实现或打包脚本可直接验证对应桌面构建。

## 构建 macOS 应用

在 `backend/mygo.json` 配置应用名、稳定的 Bundle ID、版本、Go 入口 `main`、输出目录 `out` 和 `macos.minimumSystemVersion`。桌面产物建议统一写到项目根目录的 `.fino/desktop/`。当配置位于 `backend/mygo.json` 时，设置 `"out": "../.fino/desktop"`。例如，`darwin/arm64` 的 `.app` 和 `.dmg` 会输出到 `.fino/desktop/darwin-arm64/`。已有项目使用自定义路径时沿用其配置。

优先复用项目的桌面构建脚本。以下示例使用 Fino 默认的 npm/Vite 布局。从仓库根目录运行，替换服务目录和构建标签，并设置 `mygo_version` 和 `desktop_arch`。项目如有原生功能开关，一并传给前端构建。

```sh
npm --prefix frontend ci
VITE_API_BASE='' npm --prefix frontend run build -- \
  --outDir ../backend/cmd/app-server/desktop-assets

CGO_ENABLED=1 GOFLAGS='-tags=app_desktop' \
  go -C backend run "github.com/egoist/mygo/cmd/mygo@${mygo_version}" build -platform "darwin/${desktop_arch}"
```

用目标 macOS 上的 Go、Node.js 和 Xcode Command Line Tools 构建。按目标机器选择 `arm64` 或 `amd64`，根据锁定版本和原生依赖确认系统最低版本。遇到打包工具兼容问题时先定位失败命令，再为受影响的环境做兼容处理。将嵌入目录和本地打包输出加入项目忽略规则。

## 配置应用图标

MyGo 默认可读取 `backend/resources/icon.png`。自定义路径可设 `mygo.json` 顶层 `icon`，路径相对于配置所在目录。建议使用带透明通道的 1024 × 1024 方形 PNG。已有 macOS 专用 `.icns` 时，可设 `macos.icon`。应用窗口内的字标由前端资源管理。图标换版后查看 16、32、64 px 预览，再确认生成的 `.app` 图标及实际 Dock 和 Finder 效果。

## 按改动验证

- 接入或修改启动装配时，验证普通服务和桌面版都能编译。在目标 macOS 上打包并从 Finder 启动 `.app`，按约定的联网条件验证窗口渲染、一次 API 读写和重启后数据可读。按改动补查单实例、关闭释放资源、文件选择与下载。同源路由或会话有变更时，运行已有测试并检查 SPA 深链接、无会话请求和跨源写请求。
- 修改构建或图标时，检查 `.app` 和 `.dmg` 产物，用 `file` 核对应用可执行文件架构、用 `codesign --verify --strict` 验证签名，有图标变更时核对 `Contents/Info.plist` 和 `Contents/Resources/`。对外分发时按项目发布要求处理 Developer ID 签名和公证。
- 共用代码有变更时，按 `skills/fino/references/build-and-verify.md` 检查受影响的 Go 模块和前端。若环境只允许静态检查，说明实际打包和 Finder 启动是否已验证。
