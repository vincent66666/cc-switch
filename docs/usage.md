# cc-switch 使用文档

## 1. 作用

`cc-switch` 用来管理 Claude 的多个 API 配置，并把当前选中的 profile 写入 `~/.claude/settings.json` 的 `env` 字段。

它解决的是这个问题：

- 旧方案通过 `source *.env` 覆写当前 shell 环境变量
- 这种方式只影响当前 shell，会话切换和 Claude 真实运行配置容易脱节
- `cc-switch` 改成直接维护 Claude 实际读取的配置文件

## 2. 默认文件位置

默认情况下，`cc-switch` 只会使用下面 3 个位置：

| 用途 | 路径 |
|---|---|
| profile 仓库 | `~/.claude/cc-switch/profiles.json` |
| Claude 运行配置 | `~/.claude/settings.json` |
| settings 备份目录 | `~/.claude/cc-switch/backups/` |

这意味着：

- 不再使用 `~/.claude/profiles.json`
- 不再使用 `~/.claude/backups/`
- 不会再往 `~/.claude` 根目录新增别的默认 `cc-switch` 文件

如果你需要临时覆盖默认路径，可以设置：

- `CC_SWITCH_PROFILES_PATH`
- `CC_SWITCH_SETTINGS_PATH`

## 3. 安装

### 3.1 本地构建

```bash
go build -o cc-switch .
```

### 3.2 基本验证

```bash
./cc-switch
```

如果当前还没有 profile，通常会输出：

```text
current: unknown
```

## 4. profile 数据结构

`~/.claude/cc-switch/profiles.json` 示例：

```json
{
  "version": 1,
  "current": "demo",
  "profiles": {
    "demo": {
      "description": "演示环境",
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

### 4.1 必填字段

- `ANTHROPIC_AUTH_TOKEN`
- `ANTHROPIC_BASE_URL`

### 4.2 可选字段

- `ANTHROPIC_MODEL`
- `ANTHROPIC_DEFAULT_OPUS_MODEL`
- `ANTHROPIC_DEFAULT_SONNET_MODEL`
- `ANTHROPIC_DEFAULT_HAIKU_MODEL`

### 4.3 不支持的字段

任何不在白名单中的字段都不会被保存在 profile 中。导入旧 `.env` 时也会被忽略。

## 5. 命令说明

### 5.1 `cc-switch`

显示当前状态。

```bash
cc-switch
```

示例输出：

```text
current: demo
base_url: https://example.com
model: glm-5
available: beta prod
```

### 5.2 `cc-switch current`

显示当前 profile 名称。

```bash
cc-switch current
```

示例输出：

```text
demo
```

### 5.3 `cc-switch list`

列出所有 profile，按名称排序。

```bash
cc-switch list
```

示例输出：

```text
beta
demo
prod
```

### 5.4 `cc-switch use <name>`

切换到指定 profile。

```bash
cc-switch use demo
```

成功输出：

```text
switched to demo
```

切换时的实际顺序：

1. 读取 `~/.claude/cc-switch/profiles.json`
2. 校验目标 profile 必填字段
3. 读取 `~/.claude/settings.json`
4. 备份原文件到 `~/.claude/cc-switch/backups/`
5. 只替换 `settings.json` 的 `env`
6. 原子写回 `settings.json`
7. 更新 `profiles.json.current`

### 5.5 `cc-switch add <name>`

新增 profile。

纯交互式：

```bash
cc-switch add
```

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
added demo
```

交互模式下会按这个顺序询问：

1. `name`
2. `description`
3. `ANTHROPIC_AUTH_TOKEN`
4. `ANTHROPIC_BASE_URL`
5. `ANTHROPIC_MODEL`
6. `ANTHROPIC_DEFAULT_OPUS_MODEL`
7. `ANTHROPIC_DEFAULT_SONNET_MODEL`
8. `ANTHROPIC_DEFAULT_HAIKU_MODEL`

规则：

- 已通过参数提供的字段不会再问
- `description` 和 4 个 model 字段可直接回车留空
- `token` 和 `base-url` 不能为空
- 如果交互输入的 `name` 已存在，命令会立即报错退出，不继续询问后续字段
- 非交互环境下缺少必填字段会直接报错，不进入提问流程

### 5.6 `cc-switch edit <name>`

编辑现有 profile，只覆盖显式传入的字段。

```bash
cc-switch edit demo \
  --description "新描述" \
  --base-url "https://new.example.com" \
  --model "glm-5"
```

成功输出：

```text
updated demo
```

如果在交互终端里执行 `cc-switch edit demo`，会按这个顺序逐项询问：

1. `description`
2. `ANTHROPIC_AUTH_TOKEN`
3. `ANTHROPIC_BASE_URL`
4. `ANTHROPIC_MODEL`
5. `ANTHROPIC_DEFAULT_OPUS_MODEL`
6. `ANTHROPIC_DEFAULT_SONNET_MODEL`
7. `ANTHROPIC_DEFAULT_HAIKU_MODEL`

交互规则：

- 每项都显示当前值
- 直接回车保留原值
- `token` 只显示掩码值，不回显完整 token
- 已通过参数提供的字段不会再问
- 如果某个可选 model 字段原本不存在，回车后仍保持“不写这个 key”
- `name` 不通过 `edit` 修改，改名继续用 `rename`

### 5.7 `cc-switch remove <name>`

删除非当前 profile。

```bash
cc-switch remove beta
```

成功输出：

```text
removed beta
```

如果目标是当前 profile，会失败：

```text
cannot remove the active profile
```

### 5.8 `cc-switch rename <old> <new>`

重命名 profile。

```bash
cc-switch rename demo prod
```

成功输出：

```text
renamed demo to prod
```

如果被重命名的是当前 profile，`current` 也会同步更新。

### 5.9 `cc-switch import --from <dir>`

从旧 `.env` 目录导入 profile。

```bash
cc-switch import --from ~/.claude/profiles
```

成功输出类似：

```text
imported=3 skipped=1
```

允许覆盖同名 profile：

```bash
cc-switch import --from ~/.claude/profiles --overwrite
```

## 6. 旧 `.env` 导入格式

支持的旧文件格式示例：

```bash
# 阿里 coding plan 个人
export ANTHROPIC_AUTH_TOKEN="token-demo"
export ANTHROPIC_BASE_URL="https://example.com"
export ANTHROPIC_MODEL="glm-5"
export ANTHROPIC_DEFAULT_OPUS_MODEL="glm-5"
export ANTHROPIC_DEFAULT_SONNET_MODEL="kimi-k2.5"
export ANTHROPIC_DEFAULT_HAIKU_MODEL="MiniMax-M2.5"
```

导入规则：

- 文件名去掉 `.env` 作为 profile 名称
- 第一行普通注释作为 `description`
- 只导入支持的 6 个字段
- 缺少必填字段的文件会被跳过
- 默认不覆盖同名 profile，除非加 `--overwrite`

## 7. `settings.json` 更新规则

`cc-switch` 最关键的约束是：只更新 `~/.claude/settings.json` 的 `env`。

它不会主动修改这些常见顶层字段的值：

- `model`
- `statusLine`
- `enabledPlugins`
- `extraKnownMarketplaces`

但要注意一个实现细节：

- 其他字段的值会保留
- 整个 `settings.json` 会被重新格式化
- 所以字段顺序和缩进可能发生变化

## 8. 备份规则

每次开始写入 `settings.json` 前，如果原文件存在，会先创建备份：

```text
~/.claude/cc-switch/backups/settings.json.20260313T150102Z.bak
```

如果 `settings.json` 不存在：

- 不会先生成旧文件备份
- 会直接创建最小 JSON 对象后写入新的 `env`

## 9. 常见错误

### 9.1 profile 不存在

```text
profile "demo" not found
```

### 9.2 缺少必填字段

```text
profile "demo" missing required field: ANTHROPIC_BASE_URL
```

或者：

```text
missing required field: ANTHROPIC_AUTH_TOKEN
```

### 9.3 `settings.json` 非法

```text
write settings env: invalid character ...
```

此时不会继续写入，也不会推进当前 profile。

### 9.4 未提供导入目录

```text
--from is required
```

### 9.5 命令不存在

```text
unknown command: foo
```

## 10. 推荐操作流程

1. 如果你有旧 `.env` 配置，执行 `cc-switch import --from <dir>`
2. 执行 `cc-switch list`
3. 如果需要手工新增配置，执行 `cc-switch add <name> --token ... --base-url ...`
4. 按需补模型字段
5. 执行 `cc-switch use <name>`
6. 执行 `cc-switch current`
7. 检查 `~/.claude/settings.json` 的 `env`
8. 检查 `~/.claude/cc-switch/backups/` 是否生成备份

## 11. 当前限制

- 只支持 profile 白名单中的 6 个环境变量
- `add/edit` 只支持逐项文本交互，不支持方向键选择或表单式 UI
- 目前没有 `--dry-run`
- 目前没有 shell 自动补全
- 目前没有单独的 `show <name>` 命令

## 12. 已覆盖的边缘用例

当前自动化测试已经覆盖这些交互和校验场景：

- 非交互环境下 `cc-switch add` 缺少 `name`，应直接失败
- 非交互环境下 `cc-switch add` 缺少 `ANTHROPIC_BASE_URL`，应直接失败
- `add` 遇到同名 profile，应拒绝写入
- 交互式 `add` 输入重复 `name`，应立即报错退出，不继续提问其他字段
- `edit` 中已通过参数提供的字段，应跳过交互提问
- `edit` 中短 token 的当前值应显示为掩码 `****`
- `edit` 中原本不存在的可选 model 字段，回车后不应写入空 key
- 备份目录不可写时，`use` 失败且 `current` 不推进
- 自定义 `CC_SWITCH_PROFILES_PATH` / `CC_SWITCH_SETTINGS_PATH` 下的 `add -> use -> current` 流程可正常工作

## 13. 建议继续补测

建议后续继续补下面这些自动化或人工回归场景：

- `cc-switch use <name>` 在备份目录不可写时，是否能稳定回滚
- `CC_SWITCH_PROFILES_PATH` / `CC_SWITCH_SETTINGS_PATH` 指向自定义位置时的完整流程
- 真实终端环境下长 token、短 token、空 token 的提示体验是否一致
- `settings.json` 很大、包含复杂插件配置时，格式化重写是否仍符合预期

## 14. 相关文档

- [README.md](/Users/liuzhiqiang/DevOps/cc-switch/README.md)
