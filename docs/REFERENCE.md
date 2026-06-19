# Gardener

Gardener 是一个本地运行的 Go + Web 多代理任务执行工具。用户只需要描述想完成的事情，Gardener 会先规划，再把任务分给多个 Tree 并行处理，最后汇总成果。

## Open-source project status

Gardener is published as an open-source, local-first project under the MIT License.

Useful project documents:

- [Contributing guide](CONTRIBUTING.md)
- [Code of conduct](CODE_OF_CONDUCT.md)
- [Security policy](SECURITY.md)
- [Public release safety checklist](SECURITY_PUBLIC_RELEASE.md)
- [Changelog](CHANGELOG.md)
- [Support guide](SUPPORT.md)

Before contributing or publishing packages, run:

```bash
make check
```

## Public GitHub release / deployment config

This repository should not contain personal deployment information. Relay server addresses, setup keys, frp tokens, Basic Auth passwords, packaged binaries and runtime data are local-only.

Use environment variables or local files ignored by git:

```bash
cp config/gardener-relay.env.example config/gardener-relay.env.local
set -a; source config/gardener-relay.env.local; set +a
```

See `SECURITY_PUBLIC_RELEASE.md` before pushing to a public repository.

## 平台支持承诺

Windows 是 Gardener 的一等支持平台，后续任何改动都必须保持 Windows 可编译、可启动、可使用。

每次发布前必须至少通过：

```bash
go test ./...
go vet ./...
node --check web/static/app.js
GOOS=windows GOARCH=amd64 go test -c -o /tmp/gardener-compat-windows-test.exe ./internal/compat
GOOS=windows GOARCH=amd64 go build -o gardener.exe ./cmd/server
```

Windows 支持范围包括：

- Windows 下可直接编译 `gardener.exe`。
- Windows 下可通过 `start-gardener.bat` 一键启动。
- 自动使用 Windows 默认 npm 全局安装路径查找 Codex CLI。
- 默认数据目录使用 Windows 用户桌面下的 `forest_data`。
- 任务可选择任意本地目录作为 workspace，并在该目录执行 Codex 工作。
- 停止任务时会使用 Windows 原生 `taskkill /T /F /PID` 尽量终止 Codex 及其子进程。
- 前端静态资源可从 exe 同级目录下的 `web/static` 自动加载。

## 主要能力

- 一个任务对应一片 Forest。
- Gardener 可按 Forest 当前保存的 Codex CLI 或 Claude Code 设置进行规划、调度和后续判断。
- Tree 使用该 Forest 当前保存的底层 CLI，可在用户指定目录中实际修改文件；设置页切换模型/CLI 会同步到已有 Forest，供后续继续任务和新启动的子任务使用。
- 用户创建 Forest 时可以选择目标目录；Gardener 和 Tree 都会在该目录中执行。
- 每一轮执行称为 Forest；任务详情页通过 Forest / Tree / 文件选择器查看产物。
- 每个 Tree 完成后生成 `fruit.md`；前端以站内阅读器打开，不跳转到纯文本页面。
- Gardener 对用户的可见回复由模型输出驱动，不用工程话术冒充 Gardener。
- 服务重启后，未完成的 Running Forest 会继续执行。
- 任务详情页支持 Markdown、CSV、JSON、HTML、Python 可视化/格式化预览。
- 任务详情页拥有独立 URL：`/forests/{forest_id}`，支持刷新和直达。
- 支持简体中文和英文界面。
- 支持日志/工作记录详细程度配置：简洁、标准、详细。
- 支持在设置中切换底层 CLI：Codex CLI / Claude Code。切换设置会同步更新已有 Forest 的 `cliEngine` / `modelMode`，后续继续任务和新启动的子任务会使用新选择；两种 CLI 共享同一套 Forest/Tree/fruit 数据格式。
- 支持在设置中切换 CLI 默认模型、`MiniMax-M3`、`kimi-k2.7-code`（兼容旧值 `kimi-coding` / `kimik2.6`），并为外部模型保存本地 token。
- MiniMax / Kimi 通过 Gardener 内置兼容层接入，用户不需要手动修改 `~/.codex/config.toml`。
- 数据全部保存在本地文件中，不使用数据库。

## Windows 快速使用

推荐给普通 Windows 用户分发完整压缩包，而不是单独 exe：

```text
Gardener-Windows.zip
```

压缩包结构：

```text
Gardener-Windows/
  gardener.exe
  start-gardener.bat
  start-gardener.ps1
  README-Windows.txt
  web/static/
```

使用方式：

1. 解压整个 `Gardener-Windows` 文件夹。
2. 双击 `start-gardener.bat`。
3. 浏览器会自动打开：

```text
http://localhost:8080
```

如果 Windows 拦截 bat，可右键 `start-gardener.ps1`，选择用 PowerShell 运行。

## Windows Codex CLI 路径

Gardener 会优先使用环境变量：

```powershell
$env:AUTO_GARDENER_CODEX_CMD
$env:AUTO_GARDENER_CLAUDE_CMD
```

如果没有设置，程序会自动查找 Windows npm 默认安装位置：

```text
%APPDATA%\npm\codex.cmd
%APPDATA%\npm\codex.exe
```

启动脚本还会把以下目录加入当前运行环境的 PATH：

```text
%APPDATA%\npm
%ProgramFiles%\nodejs
%ProgramFiles(x86)%\nodejs
```

因此，如果用户通过 npm 默认方式安装 Codex CLI，一般不需要手动配置。

如果 Codex 安装在自定义位置，可以在 PowerShell 中设置：

```powershell
$env:AUTO_GARDENER_CODEX_CMD = "C:\path\to\codex.cmd"
$env:AUTO_GARDENER_CLAUDE_CMD = "C:\path\to\claude.cmd"
.\gardener.exe
```

## Windows 编译

在 Windows 本机编译：

```powershell
go test ./...
go build -o gardener.exe ./cmd/server
.\gardener.exe
```

在 macOS / Linux 上交叉编译 Windows 版本：

```bash
GOOS=windows GOARCH=amd64 go test -c -o /tmp/gardener-compat-windows-test.exe ./internal/compat
GOOS=windows GOARCH=amd64 go build -o gardener.exe ./cmd/server
```

## macOS / Linux 运行

```bash
go run ./cmd/server
```

无需真实 Codex / Claude CLI 的本地 smoke test：

```bash
AUTO_GARDENER_RUNNER=mock go run ./cmd/server
```

打开：

```text
http://localhost:8080
```

## 数据目录

默认优先使用：

```text
~/Desktop/forest_data
```

如果 Desktop 不存在，则使用：

```text
~/forest_data
```

Windows 上通常是：

```text
C:\Users\<用户名>\Desktop\forest_data
```

也可以通过环境变量覆盖。

macOS / Linux：

```bash
AUTO_GARDENER_DATA=/path/to/forest_data go run ./cmd/server
```

Windows PowerShell：

```powershell
$env:AUTO_GARDENER_DATA = "D:\forest_data"
.\gardener.exe
```

## 静态前端目录

默认查找顺序：

```text
AUTO_GARDENER_STATIC
web/static
<exe目录>/web/static
<exe目录>/../web/static
```

如果把 exe 移到其他目录运行，请同时保留 `web/static`，或设置：

```powershell
$env:AUTO_GARDENER_STATIC = "C:\path\to\web\static"
.\gardener.exe
```

## Codex CLI

默认命令：

```text
codex
```

macOS / Linux 可覆盖：

```bash
AUTO_GARDENER_CODEX_CMD=/path/to/codex go run ./cmd/server
```

Windows 可覆盖：

```powershell
$env:AUTO_GARDENER_CODEX_CMD = "C:\path\to\codex.cmd"
.\gardener.exe
```

Gardener 会以自动执行模式调用 Codex CLI，允许其执行命令和修改指定 workspace 中的文件。

## 创建 Forest 与 workspace

创建 Forest 时，用户可以：

- 手动输入保存位置。
- 点击“选择”浏览本机目录。
- 留空，让 Gardener 自动在桌面上的 `Gardener成果` 下创建任务专属目录。

无论使用默认保存位置还是手动选择目录，Gardener 都会为每个任务创建独立子目录（目录名为任务 ID），避免不同任务的用户可见文件混在一起。底层 CLI 的过程文件仍在任务专属 scratch 目录中执行；保存位置只放最终交付物或必要最终修改。

## 设置

Web 顶部齿轮进入设置：

- 默认保存位置：创建 Forest 时如果不单独选择目录，则在这里创建任务专属子目录。
- 在任务中显示安排和记录：默认隐藏，避免非技术用户被内部信息打扰。
- 语言：简体中文 / English。
- 底层 CLI：
  - Codex CLI：使用 Codex CLI 执行 Gardener、Tree、Validation Tree。
  - Claude Code：使用 Claude Code 执行 Gardener、Tree、Validation Tree。
  - 设置页切换模型或底层 CLI 后，会同步更新已有 Forest；后续继续任务和新启动的 Gardener/Tree 会使用新的 `cliEngine` / `modelMode`。已经在运行的底层 CLI 进程不会被强制中断。
  - Forest 数据是 CLI 中立格式：`schedule.md`、`log.md`、Tree `fruit.md`、workspace 文件、token 记录和前端预览均不绑定 Codex 或 Claude。`cloud`、`claude-code` 等历史/误写值会自动归一为 `claude`。
- 模型：
  - CLI 默认模型：不注入外部模型参数，使用当前底层 CLI 的原生默认配置。
  - `MiniMax-M3`：Gardener/Tree 调用 Codex CLI 时自动接入 Gardener 内置兼容层，再由兼容层转发到 MiniMax OpenAI Compatible Chat Completions API。
  - `kimi-k2.7-code`（兼容旧值 `kimi-coding` / `kimik2.6`）：Codex CLI 会通过 Gardener 内置兼容层转发到 Kimi Coding API；Claude Code 会按 Kimi 官方方式注入 `ANTHROPIC_BASE_URL=https://api.moonshot.ai/anthropic`、`ANTHROPIC_AUTH_TOKEN` 和 `ANTHROPIC_MODEL=kimi-k2.7-code`（同时保留 `ANTHROPIC_API_KEY` 作为旧版兼容兜底）。
- Token：只在选择外部模型时显示。若 nginx/relay 安装 provision 已内置 MiniMax/Kimi key，设置页会提示“已内置/已配置”，用户无需再次填写；填写新 token 会覆盖本机设置。Token 会保存到本机 owner-only `forest_data/settings.json`，不会写入 Forest 报告或前端日志。
- 记录详细程度：
  - 简洁：默认，尽量少记录过程噪音。
  - 标准：记录关键进展。
  - 详细：记录更多执行输出，适合排查问题。

外部模型的默认 Codex provider 配置如下，可用环境变量覆盖：

```bash
AUTO_GARDENER_MINIMAX_MODEL=MiniMax-M3
AUTO_GARDENER_MINIMAX_TOKEN=sk-...   # 可选：本地设置没有 MiniMax key 时使用

AUTO_GARDENER_KIMI_MODEL=kimi-k2.7-code
AUTO_GARDENER_KIMI_TOKEN=sk-...      # 可选：本地设置没有 Kimi key 时使用
```

默认情况下不要覆盖 `AUTO_GARDENER_MINIMAX_BASE_URL` 或 `AUTO_GARDENER_KIMI_BASE_URL`。Gardener 会自动把 Codex CLI 指向本机兼容层，由兼容层负责把 Codex Responses API 转换为上游 Chat Completions API。

nginx/relay 安装若要让新用户自带 MiniMax/Kimi SK，请在中转服务器使用被 git 忽略的本地环境变量或文件：`GARDENER_RELAY_MINIMAX_TOKEN_FILE`、`GARDENER_RELAY_KIMI_TOKEN_FILE`（也支持 `GARDENER_RELAY_MINIMAX_TOKEN`、`GARDENER_RELAY_KIMI_TOKEN`）。新增用户时，这些 key 会进入该用户的一次性 provision，安装脚本会写入本机 `forest_data/settings.json` 并设置 owner-only 权限。

## 删除 Forest

- 可以在首页的已有任务卡片中删除 Forest；详情页不再放置删除按钮，避免误点。
- 删除后会清理该 Forest 在 `forest_data/forests/{forestID}` 下的全部数据。
- 如果该 Forest 使用的是 Gardener 自动创建的内部 workspace，也会一并清理对应 workspace。
- 如果用户手动选择了外部项目目录，Gardener 不会主动删除该外部目录，以避免误删用户自己的项目代码。

## 重命名 Forest

- 进入 Forest 详情页后，点击顶部标题旁的编辑按钮即可重命名。
- 重命名只改变 Forest 标题，不会移动 workspace，也不会改动 Tree、fruit 或已有文件。

## Token 消耗

普通任务详情页不展示 Token 消耗统计；用户只需要关注任务进度、对话和产出文件。

- 后端仍可保留底层 CLI 的 usage 解析能力，供排障或后续内部扩展使用。
- 前端默认不主动展示或强调 Token 消耗，避免把普通用户引导到成本/计量细节上。

## 并发配置

```bash
AUTO_GARDENER_MAX_TREES_PER_FOREST=5
AUTO_GARDENER_MAX_CONCURRENT_TREES=3
```

Windows PowerShell 示例：

```powershell
$env:AUTO_GARDENER_MAX_TREES_PER_FOREST = "5"
$env:AUTO_GARDENER_MAX_CONCURRENT_TREES = "3"
.\gardener.exe
```

当前推荐使用 `AUTO_GARDENER_MAX_TREES_PER_FOREST` 控制单阶段普通子任务数量；Gardener 不再限制自动阶段数量，会持续进入下一阶段直到完成、用户停止或底层 CLI/模型失败。旧环境变量 `AUTO_GARDENER_MAX_TREES_PER_WAVE` 仅作为迁移兼容别名读取。

## API

- `GET /api/settings`
- `PUT /api/settings`
- `GET /api/fs/dirs`
- `POST /api/tasks`
- `GET /api/tasks`
- `GET /api/tasks/{taskID}`
- `PATCH /api/tasks/{taskID}`
- `DELETE /api/tasks/{taskID}`
- `GET /api/usage`
- `GET /api/tasks/{taskID}/usage`
- `POST /api/tasks/{taskID}/messages`
- `POST /api/tasks/{taskID}/stop`
- `GET /api/tasks/{taskID}/events`
- `GET /api/tasks/{taskID}/files`
- `GET /api/tasks/{taskID}/trees/{treeID}`
- `GET /api/tasks/{taskID}/trees/{treeID}/fruit.md`
- `GET /api/tasks/{taskID}/gardener/schedule.md`
- `GET /api/tasks/{taskID}/gardener/log.md`

## 素材说明

- 左上角 Gardener logo 使用 Google Noto Emoji 的 person farmer/gardener SVG，语义更接近 Gardener。
- 本地文件：`web/static/assets/gardener-logo.svg`
- 来源：`https://raw.githubusercontent.com/googlefonts/noto-emoji/main/svg/emoji_u1f9d1_200d_1f33e.svg`
- 授权：Noto Emoji 项目主要图像资源采用 Apache License 2.0。

## 网页端远程控制

Gardener 后续统一使用网页端作为远程创建、查看、继续和停止任务的入口。通过 relay/frp 暴露公网 URL 后，用户直接在浏览器中打开对应地址即可操作。

当新任务或执行过程中的下一步缺少必要信息、存在多个高风险解释、需要用户选择方向/范围/风格，或必须补充凭据时，Gardener 会暂停并反问用户。用户直接在网页对话框里补充后，Gardener 会把“原始任务 + 上一次澄清问题 + 用户补充”一起交给规划器继续执行，而不是把补充误当成全新任务。

## 多用户多实例公网中转与自动升级

如果多个用户电脑各自运行本地 Gardener，但共用一台 1C2G 公网 VPS 作为入口，推荐采用：

```text
用户浏览器 -> VPS HTTPS/Caddy/frps -> 用户电脑 frpc -> 本地 Gardener
```

每台用户电脑应使用独立子域名，例如：

```text
alice.gardener.example.com
bob.gardener.example.com
```

详细部署、frp 配置、Windows 自动安装与升级流程见：

```text
DEPLOY_MULTI_INSTANCE_RELAY.md
```

构建 Windows 升级包：

```bash
VERSION=0.1.0 ./scripts/build-windows-package.sh
```

生成：

```text
dist/Gardener-Windows.zip
```

## Linux / Ubuntu packaging

Build Linux packages:

```bash
VERSION=0.1.0 ./scripts/build-linux-package.sh
```

The Linux package installs to `~/.local/share/Gardener` by default and uses systemd user services:

```bash
systemctl --user status gardener.local.service
systemctl --user status gardener.relay.service
```

Relay install command shape:

```bash
curl -fsSL http://YOUR_RELAY_SERVER/downloads/install-gardener-linux.sh -o install-gardener-linux.sh \
  && bash install-gardener-linux.sh --relay-base-url http://YOUR_RELAY_SERVER --setup-key YOUR_SETUP_KEY
```
