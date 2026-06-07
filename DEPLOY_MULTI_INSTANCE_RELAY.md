# Gardener 多用户多实例公网中转部署方案

## 结论

如果多个 Gardener 部署在不同用户电脑上，但希望共用同一套公网入口/VPS，中转方案必须具备：

1. 每台用户电脑一个独立本地 Gardener 实例。
2. 每个实例一个唯一公网入口，例如 `alice.gardener.example.com`、`bob.gardener.example.com`。
3. 每个实例一个独立隧道身份，例如 frp proxy name / token / subdomain。
4. Gardener 本体仍在用户电脑本地执行 Codex/Claude 和访问本地文件。
5. VPS 只做 HTTPS、路由、中转、升级包分发，不执行用户任务。

之前的单实例配置只适合“一台本地电脑对应一个公网域名”。多实例共用时，需要增加实例路由、认证和自动化安装/升级。

## 是否还需要钉钉

不一定。

### 只用网页 URL 就足够的场景

- 用户能打开浏览器。
- 用户需要查看文件、报告、任务状态。
- 用户不强依赖消息通知。
- 你希望减少配置复杂度。

推荐直接使用：

```text
https://alice.gardener.example.com
https://bob.gardener.example.com
```

### 仍建议保留钉钉的场景

- 用户习惯手机端发一句话创建/继续/停止任务。
- 希望任务完成/暂停时主动通知。
- 希望无需打开网页即可远程发指令。

钉钉可以作为“移动端命令入口/通知通道”，网页作为“完整控制台”。两者不是二选一。

## 推荐架构

```text
用户浏览器 / 钉钉
        ↓ HTTPS
公网 VPS：Caddy/Nginx + frps + 升级包下载
        ↓ frp HTTP 隧道
用户电脑 A：Gardener + Codex/Claude + frpc
用户电脑 B：Gardener + Codex/Claude + frpc
用户电脑 C：Gardener + Codex/Claude + frpc
```

## 域名规划

推荐使用子域名而不是路径前缀：

```text
alice.gardener.example.com -> 用户 A 本地 127.0.0.1:8080
bob.gardener.example.com   -> 用户 B 本地 127.0.0.1:8080
```

原因：当前前端和 API 使用 `/api/...`、`/assets/...` 等根路径，子域名路由最简单、最稳。

## VPS 上的 frps 示例

```toml
bindPort = 7000
vhostHTTPPort = 8081
# auth.token = "CHANGE_ME_LONG_RANDOM_TOKEN"
```

## 用户电脑上的 frpc 示例

```toml
serverAddr = "YOUR_VPS_IP"
serverPort = 27000
# auth.token = "CHANGE_ME_LONG_RANDOM_TOKEN"

[[proxies]]
name = "gardener-alice"
type = "http"
localIP = "127.0.0.1"
localPort = 8080
customDomains = ["alice.gardener.example.com"]
```

每个用户必须修改：

- `name`
- `customDomains`

## VPS 上 Caddy 示例

```caddyfile
*.gardener.example.com {
  reverse_proxy 127.0.0.1:8081
}
```

也可以用 Nginx，但 Caddy 自动 HTTPS 更省事。

## 安全建议

Gardener 可以调用 Codex/Claude 修改本地文件，公网访问必须谨慎：

1. Gardener 本地只监听 `127.0.0.1:8080`。
2. 只通过 frp 暴露到指定域名。
3. frps 配置强随机 token。
4. 公网域名使用 HTTPS。
5. 钉钉回调必须配置 `AUTO_GARDENER_DINGTALK_INCOMING_SECRET`。
6. 不要让陌生人访问用户的 Gardener 域名。
7. 后续建议增加 Web 登录/访问 Token 作为第二层保护。

## Windows 自动安装

把 `Gardener-Windows.zip` 放在 VPS 可下载地址，例如：

```text
https://download.gardener.example.com/Gardener-Windows.zip
```

用户在 PowerShell 运行：

```powershell
Set-ExecutionPolicy -Scope Process Bypass -Force
iwr https://download.gardener.example.com/install-gardener.ps1 -OutFile install-gardener.ps1
.\install-gardener.ps1 -PackageUrl "https://download.gardener.example.com/Gardener-Windows.zip" -DesktopShortcut -StartAfterInstall
```

安装目录默认：

```text
%LOCALAPPDATA%\Gardener
```

数据目录默认：

```text
桌面\forest_data
```

升级不会删除数据目录。

## Windows 升级

用户可运行：

```powershell
powershell -ExecutionPolicy Bypass -File "%LOCALAPPDATA%\Gardener\update-gardener.ps1" -PackageUrl "https://download.gardener.example.com/Gardener-Windows.zip" -Restart
```

升级流程：

1. 下载新 zip。
2. 停止正在运行的 `gardener.exe`。
3. 备份旧 exe/web/scripts 到 `backup-yyyyMMdd-HHmmss`。
4. 覆盖程序文件。
5. 保留 `gardener.config.ps1` 和 `Desktop\forest_data`。
6. 可选自动重启。

## 打包 Windows 升级包

在 macOS/Linux 开发机运行：

```bash
VERSION=0.1.0 ./scripts/build-windows-package.sh
```

产物：

```text
dist/Gardener-Windows.zip
```

上传该 zip 到 VPS 下载目录即可。

## 推荐交付流程

1. 你构建 `Gardener-Windows.zip`。
2. 上传到 VPS。
3. 每个用户运行一次安装命令。
4. 每个用户配置自己的 frpc 子域名。
5. 后续升级只需要替换 VPS 上的 zip，用户运行 update 脚本即可。

## 当前 VPS 安全加固状态（YOUR_VPS_IP）

当前中转服务器已按更安全的方式配置：

- frps control port：`27000`
- frps HTTP vhost：仅监听 VPS 本机 `127.0.0.1:18081`
- Nginx 公网入口：`80` 和 `28081`
- Gardener Web/API 入口：需要 HTTP Basic Auth
- `/health` 和 `/downloads/`：公开访问，便于健康检查和安装包下载

因此外部访问链路为：

```text
浏览器 -> YOUR_VPS_IP:28081 / 80 -> Nginx Basic Auth -> 127.0.0.1:18081 -> frps -> 用户本地 frpc -> 本地 Gardener
```

frpc 客户端配置应使用：

```toml
serverAddr = "YOUR_VPS_IP"
serverPort = 27000

auth.method = "token"
auth.token = "由部署管理员提供"
```

网页访问用户名：

```text
gardener
```

网页访问密码保存在部署管理员本机：

```text
.gardener_relay_web_password.local
```

端到端验证结果：

- 未带 Basic Auth 访问中转入口：`401`
- 带 Basic Auth 访问中转入口：成功穿透到本机测试服务，返回 `GARDENER_RELAY_E2E_OK`

## 用户分配脚本

VPS 上已安装：

```bash
/usr/local/bin/gardener-relay
```

新增用户：

```bash
gardener-relay add alice
```

输出会包含：

- 用户访问 URL
- 独立网页登录用户名
- 独立网页登录密码
- 独立 frpc 配置路径

查看用户：

```bash
gardener-relay list
```

查看某用户配置：

```bash
gardener-relay show alice --with-frpc
```

删除用户并释放端口：

```bash
gardener-relay remove alice
```

端口池：

- publicPort：`28081-28100`
- remotePort：`18081-18100`

脚本会检测：

- 用户名重复
- publicPort 重复
- remotePort 重复
- 端口已被系统占用

如果没有域名、只用 IP，则每个用户使用独立端口访问，例如：

```text
http://YOUR_RELAY_SERVER:28082
http://YOUR_RELAY_SERVER:28083
```

请在阿里云安全组放行实际分配给用户的公网端口，或放行端口段：

```text
28081-28100/tcp
```

## 一键启动 / SetupKey 交付方式

现在推荐不要让普通用户手工编辑 frpc 配置。管理员在 VPS 上为每位用户创建一个独立实例，系统会生成一个 `sk_...` SetupKey。

管理员新增用户：

```bash
ssh <relay-host> 'gardener-relay add alice'
```

输出中会包含：

- `url`：该用户的公网访问地址，例如 `http://YOUR_RELAY_SERVER:28082`
- `basicAuthUser` / `password`：网页登录账号密码
- `setupKey`：一键安装用密钥
- `installCommand`：可直接发给用户执行的一键安装命令

用户只需要在 Windows PowerShell 运行管理员给出的命令，例如：

```powershell
powershell -ExecutionPolicy Bypass -Command "iwr http://YOUR_RELAY_SERVER/downloads/install-gardener.ps1 -OutFile install-gardener.ps1; .\install-gardener.ps1 -RelayBaseUrl http://YOUR_RELAY_SERVER -SetupKey sk_xxx -DesktopShortcut -StartMenuShortcut -StartAfterInstall"
```

安装脚本会自动完成：

1. 下载并安装 `Gardener-Windows.zip`
2. 写入本地 `gardener.config.ps1`
3. 根据 SetupKey 拉取并写入 `frpc.toml`
4. 写入 `gardener.relay.json`，保存公网 URL 与网页登录信息
5. 创建桌面/开始菜单快捷方式
6. 启动 Gardener 与 frpc 隧道

用户之后只需要双击 `Gardener` 快捷方式即可：

- 本地 Gardener 自动监听 `127.0.0.1:8080`
- frpc 自动连接 `YOUR_VPS_IP:27000`
- 浏览器自动打开该用户独立公网 URL

### 查看或补发用户安装命令

```bash
ssh <relay-host> 'gardener-relay show alice'
```

如需查看完整 provision 内容，包括网页登录密码和 frpc 配置：

```bash
ssh <relay-host> 'gardener-relay show alice --with-provision'
```

### 安全边界

- `setupKey` 等同于客户端配置密钥，应只发给对应用户。
- 当前没有 HTTPS，SetupKey 和网页登录密码通过 HTTP 下载；在没有域名/证书前，这只能防止随机访问，不能防止链路监听或中间人攻击。
- 每个用户拥有独立公网端口、独立 Basic Auth、独立 frp remotePort/proxyName，避免多个用户冲突。
- 云服务器安全组需要放行该用户分配到的公网端口，例如 `28082/tcp`；也可以一次性放行规划范围 `28081-28100/tcp`。

## 2026-06-07 当前已分配用户

已支持带点号的用户名，例如 `alice`、`bob`。点号用户名会保留为网页 Basic Auth 用户名；frp proxyName 会自动转为安全形式，例如 `gardener-alice`。

当前已创建：

| 用户 | 公网 URL | 公网端口 | frp remotePort | proxyName |
|---|---:|---:|---:|---|
| `alice` | `http://YOUR_RELAY_SERVER:28081` | 28081 | 18082 | `gardener-alice` |
| `bob` | `http://YOUR_RELAY_SERVER:28082` | 28082 | 18083 | `gardener-bob` |

补发安装命令：

```bash
ssh <relay-host> 'gardener-relay show alice'
ssh <relay-host> 'gardener-relay show bob'
```

补发完整配置和网页登录密码：

```bash
ssh <relay-host> 'gardener-relay show alice --with-provision'
ssh <relay-host> 'gardener-relay show bob --with-provision'
```

## 2026-06-07 手机端性能优化记录

本轮针对手机端长任务卡顿做了降载：

- `/api/tasks?compact=1`：任务列表默认返回精简数据，减少首页/手机端初次加载 JSON 体积。
- SSE 任务事件在服务端做 750ms 合并刷新，避免底层任务高频更新时手机反复解析大 JSON。
- 前端任务详情按区域做签名缓存：标题/状态、阶段、进度、消息、概览没有变化时不再重复重绘 DOM。
- SSE 渲染合并到 `requestAnimationFrame`，高频事件只取最后一次状态渲染。
- 页面隐藏时停止主动轮询，回到前台后再刷新一次。
- 手机端消息仅渲染最近 50 条，桌面端最近 140 条。
- 文件预览继续限制大文件/大代码行数，并增加 CSS `contain` / `content-visibility` 降低长列表和长文件布局成本。
- 手机端关闭重型 backdrop-filter 和非必要动画，降低低性能设备 GPU/CPU 压力。

## 2026-06-07 Windows / macOS 自动安装更新

已补齐双端一键安装：

### Windows

- 下载入口：`http://YOUR_RELAY_SERVER/downloads/install-gardener.ps1`
- 安装包：`http://YOUR_RELAY_SERVER/downloads/Gardener-Windows.zip`
- 包内包含 `frpc.exe`。
- 安装脚本即使发现包内缺少 `frpc.exe`，也会自动从 `http://YOUR_RELAY_SERVER/downloads/frpc.exe` 兜底下载。
- 安装后写入 `frpc.toml`、`gardener.relay.json`、`gardener.config.ps1`，并可通过 `-StartAfterInstall` 自动启动。

### macOS

- 下载入口：`http://YOUR_RELAY_SERVER/downloads/install-gardener-macos.sh`
- Apple Silicon 包：`http://YOUR_RELAY_SERVER/downloads/Gardener-macOS-arm64.tar.gz`
- Intel Mac 包：`http://YOUR_RELAY_SERVER/downloads/Gardener-macOS-amd64.tar.gz`
- 两个 macOS 包都内置对应架构的 `frpc`。
- 安装脚本根据 `uname -m` 自动选择 Apple Silicon 或 Intel 包。
- 安装后写入 `frpc.toml`、`gardener.relay.json`、`gardener.config.sh`。
- 安装后注册并启动两个 LaunchAgent：
  - `com.gardener.local`：本地 Gardener，监听 `127.0.0.1:8080`
  - `com.gardener.relay`：公网 relay 隧道

macOS 安装命令样式：

```bash
curl -fsSL http://YOUR_RELAY_SERVER/downloads/install-gardener-macos.sh -o install-gardener-macos.sh && bash install-gardener-macos.sh --relay-base-url http://YOUR_RELAY_SERVER --setup-key sk_xxx
```

`gardener-relay show <user>` 现在同时输出 `installCommand` 和 `macInstallCommand`。

## 2026-06-07 macOS LaunchAgent CLI PATH 修复

问题：macOS 通过 LaunchAgent 常驻运行 Gardener 时，默认 `PATH` 只有 `/usr/bin:/bin:/usr/sbin:/sbin`，导致通过 Homebrew 安装的 `codex` / `claude` 不在 PATH 中。用户在已完成任务中点击“继续任务”后，会出现：

```text
本次请求没有完成：底层 CLI 或模型连接失败。
```

本地开发环境修复：LaunchAgent 已加入：

- `PATH`：包含 `/opt/homebrew/bin`、`/usr/local/bin`、`~/.local/bin` 等
- `HOME`
- `USER`
- `AUTO_GARDENER_CODEX_CMD=/opt/homebrew/bin/codex`
- `AUTO_GARDENER_CLAUDE_CMD=/opt/homebrew/bin/claude`

安装脚本修复：`install-gardener-macos.sh` 现在会在安装时捕获交互 shell 中的 PATH、`codex` 路径和 `claude` 路径，并写入：

- `~/Applications/Gardener/gardener.config.sh`
- `~/Library/LaunchAgents/com.gardener.local.plist`

已重新发布：

- `http://YOUR_RELAY_SERVER/downloads/install-gardener-macos.sh`
- `http://YOUR_RELAY_SERVER/downloads/Gardener-macOS-arm64.tar.gz`
- `http://YOUR_RELAY_SERVER/downloads/Gardener-macOS-amd64.tar.gz`

验证：对已完成任务再次调用 `/resume` 后，服务返回 `Running`，并进入 Git 初始化/规划流程，不再出现 `找不到 Codex CLI 命令 "codex"` 的即时失败。
