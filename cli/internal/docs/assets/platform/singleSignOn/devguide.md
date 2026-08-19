# CloudCC 单点登录 CLI 命令说明

## 支持的命令

| 操作 | 说明 |
|------|------|
| `get` | 查询单点登录配置列表 |
| `delete` | 删除单点登录记录 |

## CLI 命令详解

### 查询单点登录配置列表

```bash
cloudcc get singleSignOn <projectPath> [encodedCondJson]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `encodedCondJson` | 否 | URI 编码后的查询条件 JSON |

**查询条件参数：**

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `start` | number | 起始位置，默认 0 |
| `limit` | number | 每页条数，默认 30 |
| `keyword` | string | 搜索关键词 |

**示例：**

```bash
# 获取所有单点登录配置
cloudcc get singleSignOn .

# 带查询条件（搜索关键词）
cloudcc get singleSignOn . '%7B%22keyword%22%3A%22test%22%2C%22limit%22%3A50%7D'
```

### 删除单点登录记录

```bash
cloudcc delete singleSignOn <projectPath> <id>
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `id` | 是 | 单点登录记录 id（从列表接口获取）|

**示例：**

```bash
# 删除单点登录记录
cloudcc delete singleSignOn . 20246DB16F11EDF4KHWd
```
