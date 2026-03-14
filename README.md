# cc-switch

`cc-switch` 是一个独立 CLI，用来管理 Claude 的多个 API 配置，并把当前选中的 profile 写入 `~/.claude/settings.json` 的 `env` 字段。

它替代了原来通过 `source *.env` 覆写当前 shell 环境变量的方式。现在 profile 持久化保存在 `~/.claude/cc-switch/profiles.json`，切换时只更新 Claude 实际读取的配置文件。

详细使用说明见 [docs/usage.md](/Users/liuzhiqiang/DevOps/cc-switch/docs/usage.md)。

## 功能概览

- 用 `~/.claude/cc-switch/profiles.json` 保存多个 profile
- 用 `cc-switch use <name>` 切换当前 profile
- 切换时只更新 `~/.claude/settings.json` 的 `env`
- 写入前自动备份 `settings.json`，同秒连续写入也不会覆盖旧备份
- 支持新增、编辑、删除、重命名 profile
- `add/edit` 支持交互式逐项提问

## 默认文件路径

- profiles 仓库：`~/.claude/cc-switch/profiles.json`
- Claude 运行配置：`~/.claude/settings.json`
- settings 备份目录：`~/.claude/cc-switch/backups/`

`cc-switch` 默认只会使用这 3 个位置，不会再向 `~/.claude` 根目录新增别的默认文件。

也可以通过环境变量覆盖默认路径：

- `CC_SWITCH_PROFILES_PATH`
- `CC_SWITCH_SETTINGS_PATH`

这两个变量主要用于测试或临时调试。

## 安装

### 方式 1：直接构建

```bash
go build -o cc-switch .
```

构建完成后会得到当前目录下的 `cc-switch` 可执行文件。

### 验证安装

```bash
./cc-switch
```

## 数据结构

`~/.claude/cc-switch/profiles.json` 的结构如下：

```json
{
  "version": 1,
  "current": "demo",
  "profiles": {
    "demo": {
      "description": "演示配置",
      "env": {
        "ANTHROPIC_AUTH_TOKEN": "token-demo",
        "ANTHROPIC_BASE_URL": "https://example.com",
        "ANTHROPIC_MODEL": "glm-5",
        "ANTHROPIC_DEFAULT_OPUS_MODEL": "glm-5",
        "ANTHROPIC_DEFAULT_SONNET_MODEL": "kimi-k2.5",
        "ANTHROPIC_DEFAULT_HAIKU_MODEL": "MiniMax-M2.5"
      }
    }
  }
}
```

### 支持的字段

必填：

- `ANTHROPIC_AUTH_TOKEN`
- `ANTHROPIC_BASE_URL`

可选：

- `ANTHROPIC_MODEL`
- `ANTHROPIC_DEFAULT_OPUS_MODEL`
- `ANTHROPIC_DEFAULT_SONNET_MODEL`
- `ANTHROPIC_DEFAULT_HAIKU_MODEL`

不在这个白名单中的字段不会被接受到 profile 中。

### `profiles.json` 读写规则

- profile 名称和 `current` 在读写时都会自动去掉首尾空白
- 如果规范化后发生重名，例如 `"demo"` 和 `" demo "`，读写都会直接失败
- 除了 `list` 的宽松读取路径外，`current` 指向不存在的 profile 会被视为配置错误

## 使用方法

### 1. 查看当前状态

```bash
cc-switch
```

示例输出：

```text
当前配置：demo
接口地址：https://example.com
模型：glm-5
可用配置：beta prod
```

当前实现里，只有在 macOS/Darwin 上运行，并且 stdin/stdout 都连接到交互终端、同时存在其他可切换的 profile 时，才会显示一个可用 `↑/↓` 选择、按 `Enter` 直接切换、按 `q` 或 `Ctrl+C` 退出的列表。其他平台或非 TTY 场景会继续输出上面的纯文本结果。

如果 `profiles.json` 不存在，或者当前 profile 不存在，输出会退化为：

```text
当前配置：未知
```

如果 `profiles.json` 本身损坏，或者规范化后出现重名冲突，则会输出类似：

```text
加载配置失败：...
```

### 2. 查看当前 profile 名称

```bash
cc-switch current
```

示例输出：

```text
demo
```

如果 `profiles.json` 不存在，或者 `current` 为空、或者指向一个不存在的 profile，则输出：

```text
未知
```

如果 `profiles.json` 本身损坏，或者规范化后出现重名冲突，则会输出类似：

```text
加载配置失败：...
```

### 3. 列出所有 profile

```bash
cc-switch list
```

非交互输出按名称排序，每行一个：

```text
beta
demo
prod
```

当前实现里，只有在 macOS/Darwin 上运行，并且 stdin/stdout 都连接到交互终端时，`cc-switch list` 才会显示一个可上下选择的列表。按 `Enter` 后会进入 `切换 / 修改 / 删除 / 返回` 菜单，按 `q` 或 `Ctrl+C` 退出。其他平台或非 TTY 场景会保持上面的纯文本输出。

补充说明：

- 如果 `profiles.json` 不存在，`list` 会输出空结果并返回成功
- 如果 `current` 指向一个不存在的 profile，`list` 仍会继续列出所有可用 profile
- 如果 `profiles.json` 本身损坏，或者规范化后出现重名冲突，则会输出 `加载配置失败：...`

### 4. 切换 profile

```bash
cc-switch use demo
```

成功输出：

```text
已切换到配置：demo
```

切换时会按下面的顺序执行：

1. 读取并校验 `~/.claude/cc-switch/profiles.json`
2. 找到目标 profile
3. 读取 `settings.json`
4. 备份原始 `settings.json`
5. 只替换 `settings.json` 中的 `env`
6. 原子写回 `settings.json`
7. 更新 `~/.claude/cc-switch/profiles.json` 中的 `current`

如果写入后的 `current` 指向不存在的 profile，或者 `profiles.json` 本身存在规范化冲突，切换会失败，不会把坏状态保存回去。

## Profile 管理

### 5. 新增 profile

无参数交互式：

```bash
cc-switch add
```

带参数：

```bash
cc-switch add demo \
  --description "演示环境" \
  --token "token-demo" \
  --base-url "https://example.com" \
  --model "glm-5" \
  --default-opus-model "glm-5" \
  --default-sonnet-model "kimi-k2.5" \
  --default-haiku-model "MiniMax-M2.5"
```

成功输出：

```text
已添加配置：demo
```

交互模式下会按顺序询问：

- `name`
- `description`
- `ANTHROPIC_AUTH_TOKEN`
- `ANTHROPIC_BASE_URL`
- `ANTHROPIC_MODEL`
- `ANTHROPIC_DEFAULT_OPUS_MODEL`
- `ANTHROPIC_DEFAULT_SONNET_MODEL`
- `ANTHROPIC_DEFAULT_HAIKU_MODEL`

规则：

- 已通过参数提供的值，不再询问
- `description` 和 4 个 model 字段可留空
- `token` 和 `base-url` 必填
- 命令行参数里的 profile 名称会自动去掉首尾空白
- 如果交互输入的 `name` 已存在，会立即报错退出，不继续后续提问
- 在非交互环境里缺少必填字段时会直接报错

例如：

```text
缺少必填字段：ANTHROPIC_AUTH_TOKEN
```

### 6. 编辑 profile

纯交互式：

```bash
cc-switch edit demo
```

带参数：

```bash
cc-switch edit demo \
  --description "新的描述" \
  --base-url "https://new.example.com" \
  --model "glm-5"
```

成功输出：

```text
已更新配置：demo
```

说明：

- `edit` 会按顺序逐项询问：`description`、`token`、`base-url`、4 个 model 字段
- 已通过参数提供的值，不再询问
- 每项都显示当前值，直接回车保留原值
- `token` 提示时显示掩码，不回显完整值
- 如果某个可选 model 字段原本不存在，回车保留后仍保持“不写这个 key”
- 命令行参数里的 profile 名称会自动去掉首尾空白
- `name` 不支持通过 `edit` 修改，改名继续走 `rename`
- 如果编辑后缺少必填字段，命令会失败

### 7. 删除 profile

```bash
cc-switch remove beta
```

成功输出：

```text
已删除配置：beta
```

限制：

- 不能删除当前激活的 profile

失败示例：

```text
不能删除当前正在使用的配置
```

### 8. 重命名 profile

```bash
cc-switch rename demo prod
```

成功输出：

```text
已将配置 demo 重命名为 prod
```

如果被重命名的是当前 profile，`current` 指针也会一起更新。命令行参数里的旧名称和新名称都会自动去掉首尾空白。

## `settings.json` 更新规则

这是本工具最重要的行为约束。

`cc-switch use <name>` 只会修改：

```json
{
  "env": {
    "...": "..."
  }
}
```

不会主动修改其他顶层字段，例如：

- `model`
- `statusLine`
- `enabledPlugins`
- `extraKnownMarketplaces`

但需要注意：

- 当前实现会重新格式化整个 `settings.json`
- 也就是说，其他字段的值会保留，但 JSON 的字段顺序和缩进格式可能变化

## 备份规则

每次成功开始写入 `settings.json` 前，都会先创建备份：

```text
~/.claude/cc-switch/backups/settings.json.20260313T150102Z.bak
```

如果同一秒里已经存在同名时间戳备份，后续备份会自动递增，例如：

```text
~/.claude/cc-switch/backups/settings.json.20260313T150102Z.1.bak
```

如果 `settings.json` 本身不存在，则不会先备份旧文件，而是直接初始化后写入新的 `env`。

## 常见错误

### profile 不存在

```text
未找到配置 "demo"
```

### 缺少必填字段

```text
配置 "demo" 缺少必填字段：ANTHROPIC_BASE_URL
```

或者：

```text
缺少必填字段：ANTHROPIC_AUTH_TOKEN
```

### `settings.json` 非法

```text
写入 settings.json 的 env 失败：invalid character ...
```

遇到这种情况时，工具不会继续写入，也不会推进 `~/.claude/cc-switch/profiles.json` 里的 `current`。如果 `settings.json` 已经写入成功、但更新 `current` 失败，工具会回滚刚刚写入的 `settings.json`。

### `profiles.json` 非法或不一致

例如：

```text
加载配置失败：invalid character ...
```

或者：

```text
加载配置失败：配置 "demo" 已存在
```

或者：

```text
加载配置失败：当前配置 "ghost" 不存在
```

这类错误通常表示 `profiles.json` 本身损坏、规范化后重名，或者 `current` 指针指向了不存在的 profile。`current/status` 会只在“缺文件”或“悬空 current”这两类可降级场景下输出 `未知`；其余读取错误会直接报错。

### 命令不存在

```text
未知命令：foo
```

## 推荐使用流程

### 首次配置

1. 执行 `cc-switch add <name> --token ... --base-url ...`
2. 按需补充模型字段
3. 执行 `cc-switch use <name>`
4. 执行 `cc-switch current`
5. 打开 `~/.claude/settings.json` 检查 `env` 是否已更新
6. 如果原本已存在 `settings.json`，确认备份出现在 `~/.claude/cc-switch/backups/`

### 日常切换

1. 执行 `cc-switch list`
2. 执行 `cc-switch use <name>`
3. 执行 `cc-switch current`

## 当前限制

- 只支持 profile 白名单中的 6 个环境变量
- `add/edit` 的交互输入目前只覆盖 `name`、`description`、`token`、`base-url` 和 4 个 model 字段
- 目前没有 `--dry-run`
- 目前没有 shell 自动补全
- 目前没有直接显示某个 profile 全量详情的子命令

## 已覆盖的边缘用例

- 非交互环境下 `cc-switch add` 缺少 `name` 会直接失败
- 非交互环境下 `cc-switch add` 缺少 `base-url` 会直接失败
- `add` 遇到重复 profile 名称会拒绝写入
- 交互式 `add` 输入重复 `name` 会立即报错退出，不继续询问其他字段
- `edit` 里已通过参数提供的字段不会重复提问
- `edit` 里短 token 也会以掩码形式显示，不会明文回显
- `edit` 里原本不存在的可选 model 字段，回车保留后不会写出空 key
- `add/edit/use/remove/rename` 的命令行 profile 名称参数会自动修剪首尾空白
- 备份目录不可写时，`use` 会失败且不会推进 `current`
- 更新 `current` 失败时，`use` 会回滚刚写入的 `settings.json`
- 同一秒内连续备份不会覆盖之前的备份文件
- `current` 指向不存在的 profile 时，`cc-switch current` / `cc-switch status` 会输出 `未知`
- `profiles.json` 不存在时，`cc-switch list` 会输出空结果而不是报错
- `profiles.json` 规范化后重名或 JSON 损坏时，`current/status/list` 会输出 `加载配置失败：...`
- 自定义 `CC_SWITCH_PROFILES_PATH` / `CC_SWITCH_SETTINGS_PATH` 的 `add -> use -> current` 流程已有回归测试

## 建议继续补测

- `settings.json` 原本不存在、随后 `current` 更新失败时，是否能稳定删除刚创建的临时配置文件
- 真实终端环境下，带空白名称参数的 `use/remove/rename` 交互提示是否符合预期
- 交互式 `list` 在 `current` 悬空时是否完全不高亮任何项
- `settings.json` 非法 JSON、目录不存在、跨自定义 `CC_SWITCH_SETTINGS_PATH` 时的完整人工回归

## 文档

- [README.md](/Users/liuzhiqiang/DevOps/cc-switch/README.md)：项目概览与快速上手
- [docs/usage.md](/Users/liuzhiqiang/DevOps/cc-switch/docs/usage.md)：详细使用文档

## 开发与验证

运行测试：

```bash
go test ./... -count=1
```

构建：

```bash
go build ./...
```

## 文件说明

- [main.go](/Users/liuzhiqiang/DevOps/cc-switch/main.go)：CLI 入口
- [internal/cli/app.go](/Users/liuzhiqiang/DevOps/cc-switch/internal/cli/app.go)：命令分发与参数处理
- [internal/profile/store.go](/Users/liuzhiqiang/DevOps/cc-switch/internal/profile/store.go)：`~/.claude/cc-switch/profiles.json` 读写
- [internal/profile/validate.go](/Users/liuzhiqiang/DevOps/cc-switch/internal/profile/validate.go)：profile 校验规则
- [internal/settings/store.go](/Users/liuzhiqiang/DevOps/cc-switch/internal/settings/store.go)：只更新 `settings.json.env`
- [internal/settings/backup.go](/Users/liuzhiqiang/DevOps/cc-switch/internal/settings/backup.go)：备份逻辑
