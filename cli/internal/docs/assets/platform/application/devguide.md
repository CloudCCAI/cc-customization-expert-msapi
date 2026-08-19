# CloudCC 应用开发指南

## 1. 模块定位

`application` 模块用于通过 CLI 管理 CloudCC 应用，当前提供：

- 创建应用：`cloudcc create application ...`
- 查询应用：`cloudcc get application ...`
- 删除应用：`cloudcc delete application ...`
- 更新应用：`cloudcc update application ...`
- 文档查看：`cloudcc doc platform/application introduction|devguide`

---

## 2. 开发前准备

执行命令前请确认：

- 已完成 `cloudcc doc platform/project devguide` 的环境初始化
- 项目根目录存在可用配置，且包含 `accessToken`
- 已准备应用名称、应用代码
- 若需自定义菜单挂载，已准备菜单 ID 列表

---

## 3. 命令总览（以代码实现为准）

```bash
cloudcc create application <path> <p1> <p2> [duel1]
cloudcc get application <projectPath> [encodedCondJson]
cloudcc delete application <projectPath> <appId>
cloudcc update application <path> <appId> <appLabel> <appName> [duel1] [encodedOptionsJson]
cloudcc doc platform/application <introduction|devguide>
```

参数约定：

- `path` / `projectPath`：项目路径
- `p1`：应用名称
- `p2`：应用代码
- `duel1`：菜单 ID 列表（逗号分隔，可选）
- `encodedCondJson`：URI 编码后的 JSON 查询条件
- `appId`：应用 ID
- `encodedOptionsJson`：URI 编码后的 JSON 覆盖参数（用于更新）

---

## 4. 创建应用

```bash
cloudcc create application <path> <p1> <p2> [duel1]
```

### 4.1 参数说明

- `path`：项目路径，推荐传 `.` 表示当前目录
- `p1`：应用名称（例如 `销售工作台`）
- `p2`：应用代码（例如 `sales_workbench`）
- `duel1`：菜单 ID 列表，多个 ID 以逗号分隔

### 4.2 默认行为与自动补全

- 若未传 `duel1`，CLI 会默认使用 `acf000001`
- 若传入 `duel1` 但不包含 `acf000001`，CLI 会自动补入该 ID
- 创建过程中会自动查询角色列表并注入权限参数

### 4.3 示例

```bash
# 使用默认菜单 ID
cloudcc create application . "销售工作台" sales_workbench

# 指定菜单 ID（可多个）
cloudcc create application . "销售工作台" sales_workbench "acf000001,a0I9D000000XXXXUAI"
```

### 4.4 常见报错

- 缺少参数：

```text
Error: 缺少必需参数
用法: cloudcc create application <path> <p1> <p2> [duel1]
```

- 角色列表获取失败（依赖 `brief/get`）：

```text
获取角色列表失败
```

---

## 5. 查询应用

```bash
cloudcc get application <projectPath> [encodedCondJson]
```

说明：

- 不传条件时，返回应用列表
- 传条件时需使用 `encodeURI(JSON.stringify(...))` 编码

示例：

```bash
# 查询全部应用
cloudcc get application .

# 按条件查询（示例）
cloudcc get application . '%7B%22appType%22%3A%22app%22%7D'
```

若条件 JSON 无法解析，会报错：

```text
Get Application List Failed: encodedCondJson 解析失败，请传 encodeURI(JSON.stringify(...))
```

---

## 6. 删除应用

```bash
cloudcc delete application <projectPath> <appId>
```

参数说明：

- `projectPath`：项目路径
- `appId`：应用 ID（必填）

示例：

```bash
cloudcc delete application . a0L9D000000XXXXUAI
```

缺少 `appId` 时会报错：

```text
Error: 缺少应用 ID
用法: cloudcc delete application <projectPath> <appId>
```

---

## 7. 更新应用

```bash
cloudcc update application <path> <appId> <appLabel> <appName> [duel1] [encodedOptionsJson]
```

参数说明：

- `path`：项目路径，推荐传 `.`
- `appId`：待更新应用 ID
- `appLabel`：应用显示名称
- `appName`：应用编码/内部名称
- `duel1`：已选菜单 ID 列表（可选，多个逗号分隔）
- `encodedOptionsJson`：可选覆盖参数，需使用 `encodeURI(JSON.stringify(...))`

示例：

```bash
# 仅更新名称并绑定菜单
cloudcc update application . ace2026E2412889rIjTH "贺文娟0415" hewenjuan0415 "acf000001,acf2026EF5726F9AP8RQ"

# 覆盖导航样式与说明
cloudcc update application . ace2026E2412889rIjTH "贺文娟0415" hewenjuan0415 "acf000001,acf2026EF5726F9AP8RQ" "%7B%22navigationStyle%22:%221%22,%22description%22:%22updated%20by%20cli%22%7D"
```

---

## 8. 文档命令

```bash
cloudcc doc platform/application introduction
cloudcc doc platform/application devguide
```

仅支持 `introduction` 与 `devguide`，其他子命令会抛错。

---

## 9. 推荐操作顺序

```bash
# 1) 先查现有应用，避免重名
cloudcc get application .

# 2) 创建应用
cloudcc create application . "销售工作台" sales_workbench

# 3) 再次查询确认已创建
cloudcc get application .

# 4) 如需回滚，按 appId 删除
cloudcc delete application . <appId>
```

---

## 10. 编辑保存接口参考（非 CLI 命令）

当前 `application` 模块支持 `create/get/delete/update`，其中更新命令为：`cloudcc update application ...`。  
若需要了解完整后端接口字段与前端组装规则，可参考下述接口整理文档：



---
