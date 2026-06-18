# Gardener 正式需求文档

## 1. 项目定位

Gardener 是一个使用 Go 语言开发的本地多 Agent 任务调度系统，包含服务端和 Web 界面。本项目是正式上线需求，不是 MCP、实验 Demo 或 POC。

系统不需要真实多用户登录，不面向多用户 SaaS。系统运行在用户本机，服务当前使用者。

用户通过 Web 创建任务。每个任务对应一片 Forest。每片 Forest 有一个 Gardener。Gardener 背后真实调用 Codex CLI，负责和用户交互、理解需求、规划任务、明确每个 Tree 的工作范围、调度多个 Tree、读取 Tree 结果、派 Validation Tree 验证、必要时派新 Tree 修复冲突，并判断 Forest 何时 Finished。

每个 Tree 背后也真实调用 Codex CLI。Tree 可以在目标项目目录中实际修改代码或文件。每个 Tree 只负责 Gardener 分配的明确子任务，完成后必须生成 `fruit.md`。

## 2. 状态模型

Forest、Gardener、Tree 的状态均严格只有两种：

- `Running`
- `Finished`

不使用 queued、failed、cancelled、waiting、completed、result 等额外结构化状态字段。失败、停止、异常、部分完成等信息只写入：

- `gardener/log.md`
- `trees/{tree_id}/progress.log`
- `trees/{tree_id}/fruit.md`
- Web 文本日志/进展内容

Gardener 判断任务完成后，Forest 直接变为 `Finished`，不需要用户最终确认。

## 3. 核心对象

### 3.1 Forest / Task

每个用户任务对应一片 Forest。Forest 保存：

- Forest ID
- 任务标题
- 原始任务描述
- 目标项目目录 `workspacePath`
- 当前状态：`Running` 或 `Finished`
- Gardener 状态：`Running` 或 `Finished`
- Gardener 对话记录
- Tree 列表
- Tree 执行状态：`Running` 或 `Finished`
- Tree 报告路径
- 创建时间
- 更新时间

### 3.2 Gardener

Gardener 职责：

- 真实调用 Codex CLI
- 与用户交互
- 理解用户任务和追加需求
- 初始化目标目录 Git 仓库
- 生成和维护 `schedule.md`
- 生成和维护 `log.md`
- 拆分任务并明确每个 Tree 的工作范围
- 调度多个 Tree 并行工作
- 读取 Tree 的 `fruit.md` 和进展日志
- 派 Validation Tree 验证本批 Tree 结果
- 如果多个 Tree 并行修改产生冲突，由 Gardener 派新的 Tree 后续修复
- 判断 Forest 何时 Finished

Gardener 的 Codex CLI 工作目录等于 `workspacePath`。

Gardener 文件：

- `schedule.md`：当前 Forest 的任务规划、Tree 调度计划、批次安排、职责边界、冲突修复计划。
- `log.md`：Gardener 运行日志、用户对话摘要、Codex 输出、Tree 汇报摘要、停止/失败/异常记录、后续决策记录。

### 3.3 Tree

Tree 是具体子任务执行单元。每个 Tree：

- 真实调用 Codex CLI
- Codex CLI 工作目录等于 `workspacePath`
- 可以实际修改目标项目代码或文件
- 只负责一个明确子任务
- 必须遵守 Gardener 在 `schedule.md` 中划定的工作范围
- 多个 Tree 可以并行在同一个 `workspacePath` 中运行
- 系统不为 Tree 创建 worktree、副本或沙箱
- 完成后必须生成 `fruit.md`
- 完成后向 Gardener 汇报

Tree 状态只有：

- `Running`
- `Finished`

### 3.4 Validation Tree

每批普通 Tree 完成后，Gardener 需要派出 Validation Tree。Validation Tree 也是 Tree，必须生成自己的 `fruit.md`。

Validation Tree 职责：

- 检查本批 Tree 的修改
- 尝试运行合适的测试/构建/静态检查
- 总结冲突、失败、风险和后续修复建议
- 向 Gardener 汇报

## 4. 数据存储要求

不使用数据库。所有数据都保存到用户桌面的本地目录：

```text
~/Desktop/forest_data
```

需要兼容不同平台：

- 优先使用用户主目录下的 `Desktop/forest_data`
- 如果 `Desktop` 不存在，则使用用户主目录下的 `forest_data`
- 可通过环境变量 `AUTO_GARDENER_DATA` 覆盖

服务启动时，如果目录不存在，需要自动创建。

任务数据必须跨服务重启保留。服务关闭后再次启动，Web 仍然可以读取历史 Forest、Tree、消息记录、Gardener 文件和 `fruit.md`。

目录结构：

```text
forest_data/
  forests/
    {forest_id}/
      forest.json
      messages.json
      gardener/
        schedule.md
        log.md
      trees/
        {tree_id}/
          tree.json
          progress.log
          fruit.md
  workspaces/
    {forest_id}_{safe_task_title}/
      # 用户未选择目标目录时自动创建的默认 workspace
```

如果用户创建任务时选择了目标目录，则 `workspacePath` 使用用户选择的目录。若用户未选择，则自动创建：

```text
forest_data/workspaces/{forest_id}_{safe_task_title}
```

## 5. Git 初始化要求

目标目录不要求预先是 Git 仓库。由 Gardener 统一初始化。

规则：

1. 如果 `workspacePath` 不是 Git 仓库，Gardener 在派 Tree 前执行 `git init`。
2. 如果目录中已有文件，Gardener 尝试创建初始 commit。
3. 如果目录为空，只执行 `git init`。
4. 如果初始 commit 失败，不阻塞任务继续执行，但必须写入 `gardener/log.md`。

## 6. Codex CLI 权限要求

允许 Codex CLI 自主执行所有命令，即使是危险命令。

Gardener：

- 不做命令白名单
- 不做人为审批
- 调用 Codex CLI 时使用自动执行参数
- 允许 Codex 修改文件和执行命令
- 只负责记录日志、状态、输出和报告

默认 Codex 命令：

```text
codex
```

可通过环境变量覆盖：

```text
AUTO_GARDENER_CODEX_CMD=/path/to/codex
```

## 7. 并行与冲突处理

第一版必须体现多个 Tree 并行工作。

默认限制：

- 每个阶段最多 5 个普通子任务
- 同一时间最多 3 个子任务并行运行
- 自动阶段不设置上限；只要 Gardener 判断任务未完成，就持续进入下一阶段，直到完成、用户停止或底层 CLI/模型失败。

可通过环境变量覆盖：

- `AUTO_GARDENER_MAX_TREES_PER_FOREST`
- `AUTO_GARDENER_MAX_CONCURRENT_TREES`
- 兼容迁移别名：`AUTO_GARDENER_MAX_TREES_PER_WAVE`

系统不隔离 Tree，不创建 worktree，不创建 workspace 副本。多个 Tree 可以并行修改同一个 `workspacePath`。Gardener 必须明确各 Tree 的工作范围；如果产生冲突，由 Gardener 派新的 Tree 后续修复。

## 8. 停止任务

用户可以停止 Forest。停止时：

- 停止 Gardener
- 停止正在运行的 Tree
- Tree 的停止由用户通过 Gardener/Forest 层级触发
- Web 不提供单个 Tree 停止按钮
- 服务端负责实际中断/kill Codex CLI 进程，确保可靠停止
- Forest/Gardener/Tree 最终状态写为 `Finished`
- 停止原因写入 `log.md`、`progress.log`、`fruit.md`

## 9. fruit.md 要求

每个 Tree 完成后必须生成：

```text
fruit.md
```

位置：

```text
forest_data/forests/{forest_id}/trees/{tree_id}/fruit.md
```

内容结构：

```markdown
# Tree 工作报告

## 1. Tree 基本信息

- Tree ID:
- 所属 Forest ID:
- 所属任务:
- 子任务名称:
- 状态: Running 或 Finished
- 开始时间:
- 结束时间:
- 目标项目目录:
- 工作范围:

## 2. 子任务目标

## 3. 执行过程

## 4. 完成结果

## 5. 产出文件或关键修改

## 6. 遇到的问题

## 7. 对 Gardener 的汇报

## 8. 后续建议
```

## 10. Web 界面要求

Web 界面包含：

- 森林列表
- Forest 详情
- 创建任务表单
- 目标项目目录输入框，可为空
- Gardener 对话区
- 固定视角伪 3D 森林可视化
- Tree 进度列表
- `fruit.md` 查看入口
- `schedule.md` 查看入口
- `log.md` 查看入口
- 停止任务按钮
- 实时进度推送

## 11. 固定视角伪 3D 森林可视化

第一版采用轻量固定视角伪 3D，不引入 Three.js。

要求：

- 每个 Tree 对应一棵 3D 风格树
- Tree 数量越多，森林越大、越密集
- 通过透视地面、远近缩放、阴影、层次、轻微动画体现 3D 感
- 点击一棵树，定位到对应 Tree 详情卡片
- 不需要拖拽旋转视角

## 12. 实时推送

前端需要实时推送，使用 SSE：

```text
GET /api/tasks/{taskID}/events
```

用户发送消息、停止任务等仍使用 HTTP POST。

## 13. API

```text
POST /api/tasks
GET /api/tasks
GET /api/tasks/{taskID}
POST /api/tasks/{taskID}/messages
POST /api/tasks/{taskID}/stop
GET /api/tasks/{taskID}/events
GET /api/tasks/{taskID}/trees/{treeID}
GET /api/tasks/{taskID}/trees/{treeID}/fruit.md
GET /api/tasks/{taskID}/gardener/schedule.md
GET /api/tasks/{taskID}/gardener/log.md
```

## 14. 启动方式

第一版支持命令行和二进制启动：

```bash
go run ./cmd/server
```

```bash
go build -o auto_gardener ./cmd/server
./auto_gardener
```

## 15. 开发要求

保留当前已有代码，但允许较大重构。


## 15. 品牌、用户体验与国际化补充

- 项目对外名称为 Gardener，不再使用 Auto Gardener 作为产品名。
- 左上角必须展示园丁主题 logo。当前使用 Google Noto Emoji 的 person farmer/gardener SVG 素材 `web/static/assets/gardener-logo.svg`，授权为 Apache License 2.0。
- 普通用户界面避免展示工程术语。默认不展示保存位置、任务安排、工作记录等高级信息；这些入口由设置控制。
- Gardener 在聊天中的回复必须面向普通用户，不应出现 workspace、Codex CLI、Tree、log.md、fruit.md 等工程化表达。前端也需要对历史技术词做用户化转换。
- Gardener 推理或任务运行期间，聊天框需要展示“输入中”动画，让用户知道系统仍在工作。
- 首页不应是空洞提示页，应聚合所有任务形成 Garden 总览；每个任务以园圃/树林形式展示，可点击进入。
- Forest 详情页中右上角 Forest/Forest 概览应为小型悬浮窗。概览只粗略展示 Forest，不直接铺开全部 Tree；点击某个 Forest 后再展示其下具体 Tree。
- 点击成果报告时不能跳转到纯文本新页面，应在站内阅读器/浮层中展示，并提供复制能力。
- 支持 i18n：简体中文和英文。语言可在设置中切换，设置保存在本地浏览器。
- 日志/工作记录等级必须可配置：简洁、标准、详细。简洁为默认，减少重复、低信息量输出。

## 16. 外部模型兼容补充

- 设置页必须允许用户在 Codex CLI 和 Claude Code 两种底层 CLI 之间切换。
- 一个 Forest 创建后必须固定底层 CLI；该 Forest 内的 Gardener、Tree、Validation Tree 要么全部使用 Claude Code，要么全部使用 Codex CLI，不能混用。
- 设置页必须允许用户在 CLI 默认模型、MiniMax、Kimi 之间切换。
- CLI 默认模型使用用户本机所选底层 CLI 的原生配置，不注入外部 provider。
- MiniMax / Kimi 对普通用户应尽量无感：用户只选择模型并填写 token，不需要手动编辑 `~/.codex/config.toml`。
- MiniMax / Kimi 通过 Gardener 内置兼容层接入。Codex CLI 仍然面向本机 Responses API endpoint；兼容层负责转发到上游 OpenAI Compatible Chat Completions API。
- MiniMax 默认上游为 `https://api.minimaxi.com/v1`，默认模型为 `MiniMax-M3`。
- Kimi 默认上游为 `https://api.kimi.com/coding/v1`，默认模型为 `kimi-k2.7-code`，可通过 `AUTO_GARDENER_KIMI_MODEL` 覆盖。
- 当底层 CLI 为 Claude Code 且模型选择 Kimi 时，Gardener 按 Kimi 官方 Claude Code 接入方式注入 `ANTHROPIC_BASE_URL=https://api.kimi.com/coding/` 和 `ANTHROPIC_API_KEY`。
- token 只保存在本地 `forest_data/settings.json`，不得写入日志、报告或需求文档。

## 25. Codex / Claude 数据兼容要求

- Forest/Tree 数据必须采用 CLI 中立格式保存，不能把 Codex 或 Claude 的私有实现细节写成必须字段。
- 每个 Forest 仍然固定一个底层 CLI：Codex CLI 或 Claude Code；同一 Forest 内不得混用。
- 切换全局底层 CLI 后，历史 Forest 仍按自身 `cliEngine` 字段继续运行，但其 `schedule.md`、`log.md`、Tree `fruit.md`、workspace 文件、usage 记录和前端文件预览必须继续可读。
- `cliEngine` 允许兼容历史/误写别名：`codex`、`codex-cli`、`openai` 归一为 `codex`；`claude`、`claude-code`、`claude-cli`、`anthropic`、`cloud` 归一为 `claude`。
- 新生成的内部执行输出文件应使用中立命名，例如 `agent_last_message.md`，不要再使用 `codex_last_message.md` 这类绑定某一 CLI 的新文件名；旧文件仍允许保留并可由历史数据继续引用。
