# SkillHub macOS 桌面版实施计划

审查通过，可直接用 **MyGo v0.10.0 和 Go 1.27.4** 实施。首版只验收当前 Intel Mac（macOS `13.7.8`），产物为从 Finder 打开的 ad hoc 签名 `SkillHub.app`。一次性更新后端模块的 Go 基线，桌面构建继续使用仓库现有的 npm 锁文件。 [MyGo v0.10.0](https://github.com/egoist/mygo/releases/tag/v0.10.0) · [Go 1.27.4](https://go.dev/doc/devel/release#go1.27.0)

## 架构

保留单进程方案。MyGo 提供窗口，WKWebView 通过同源 HTTP 使用内嵌的 React 页面和 Fino API，数据继续由现有业务层写入 SQLite 和快照目录。改动集中在桌面启动、资源打包和原生交互。原方案中的单实例锁和目录选择直接复用 MyGo，ZIP 下载分别覆盖直链和 Blob 两条现有路径。 [MyGo 应用生命周期](https://mygo.run/docs/app#lifecycle) · [原生对话框](https://mygo.run/docs/native#dialogs) · [下载行为](https://mygo.run/docs/windows#downloads)

- **启动与退出**。保持 Fino 生成的 `main.go`、数据库装配和迁移函数。用 `skillhub_desktop` 构建标签区分手写的 `loadConfig` 和 `serve`，共享 `buildServer`。桌面模式使用随包的本地配置和绝对路径，初始化 SQLite、分页和日志。先获取 MyGo 单实例锁，再打开数据库。HTTP 就绪后，在 MyGo 主线程创建窗口。关闭最后一个窗口或按 Cmd+Q 时，停止接收请求，限时等待进行中的操作，再关闭数据库。启动失败时记录原因并在桌面提示。
- **页面与本地接口**。将 Vite 产物通过 `go:embed` 嵌入，在 `127.0.0.1:0` 上与 `/skillhub/v1` 同源提供。复用生成的 HTTP Handler 和 endpoint 中间件，仅对页面导航回退到 `index.html`。窗口用一次性凭证换取 `HttpOnly`、`SameSite=Strict` 会话后重定向。API 校验会话与 `Host`，写请求校验同源。外链交给系统浏览器。
- **桌面交互**。目录选择和导出共享 `mygo.Dialog.Open` 适配。首版导航聚焦技能库，用户通过目录导出或 ZIP 下载获取 Skill，主题跟随系统。GitHub 收录和更新调用本机 Git 并需要联网。
- **数据**。数据库、快照、上传暂存和默认导出目录放在 `~/Library/Application Support/SkillHub/`，日志放在 `~/Library/Logs/SkillHub/`。保留 SQLite 的 WAL、忙等待和外键设置。首版从空库启动，用户可通过已有的备份 ZIP 导入功能恢复数据。

## 实施顺序

1. **打通技术链路。** 用新版工具链、当前的 Xcode Command Line Tools 和项目现有 SQLite 驱动打出最小 `.app`。从 Finder 启动，用现有前端生产产物验证 WKWebView 渲染，跑通一次同源请求和 SQLite 写入读回。 [Go 1.27 系统要求](https://go.dev/doc/go1.27#darwin) · [MyGo CGO 构建说明](https://mygo.run/docs/distribution#platforms)
2. **接入 SkillHub。** 更新 `backend/go.mod` 的 Go 基线，在 `package.yaml` 固定 MyGo v0.10.0，接好桌面装配。运行 `fino doctor --output=. --targets=service` 和 `fino build --output=. --targets=service`，检查生成差异、模块依赖和手写扩展点。
3. **固定本机打包入口。** 在 `backend/mygo.json` 设置应用 ID、Go 入口 `./cmd/skillhub-server`、输出目录 `../.fino/desktop` 和系统最低版本 `13.0`，与 Go 的系统要求一致。一个脚本负责 `npm ci`、构建并写入桌面专用的嵌入目录，以及调用 MyGo 打包。桌面前端使用同源 API 并开启目录选择。脚本从仓库根目录运行下面的打包命令，生成 `SkillHub.app` 和拖拽安装用的 `skillhub-0.1.0-amd64.dmg`，输出到 `.fino/desktop/darwin-amd64/`。 [MyGo 配置](https://mygo.run/docs/configuration) · [构建标签实现](https://github.com/egoist/mygo/blob/v0.10.0/cmd/mygo/build.go#L514)

```sh
GOTOOLCHAIN=go1.27.4 CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=13.0 \
  GOFLAGS='-tags=skillhub_desktop' go -C backend run github.com/egoist/mygo/cmd/mygo@v0.10.0 \
  build -platform darwin/amd64
```

## 本机验收

| 检查 | 通过标准 |
| --- | --- |
| 产物与启动 | 确认 Mach-O 为 `x86_64`，签名验证通过。用空库从 Finder 打开应用，列表、详情和刷新可用。 |
| 核心流程 | 用包含空格和中文的路径完成目录收录，验证 ZIP 收录、版本比对、目录导出、直链 ZIP 下载和备份 ZIP 下载及恢复。验证选择取消、外链和剪贴板。 |
| 生命周期 | 重复启动聚焦已有窗口，退出后释放端口。重启或替换 `.app` 后数据可读，断网仍能浏览和导出已保存版本。验证无会话请求和跨源写请求会被拒绝。 |
| 项目检查 | 在新版工具链下运行 API 和后端测试、前端生产构建。 |
