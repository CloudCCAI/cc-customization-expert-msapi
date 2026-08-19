# CloudCC 用户 CLI 命令说明

## 支持的命令

| 操作 | 说明 |
|------|------|
| `create` | 创建新用户 |
| `get` | 查询用户列表 |
| `view` | 查看单个用户详情 |
| `update` | 编辑/禁用用户 |
| `delete` | 删除用户 |

## CLI 命令详解

### 创建用户

```bash
cloudcc create user <path> <userName> <profileId> [email]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 是 | 项目路径，`.` 表示当前目录 |
| `userName` | 是 | 用户名称 |
| `profileId` | 是 | 关联简档 ID |
| `email` | 否 | 用户邮箱 |

**示例：**

```bash
# 创建用户
cloudcc create user . "张三" a0I9D000000XXXXUAI

# 创建带邮箱的用户
cloudcc create user . "李四" a0I9D000000XXXXUAI "lisi@example.com"
```

### 查询用户列表

```bash
cloudcc get user <projectPath> [encodedCondJson]
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
# 获取所有用户
cloudcc get user .

# 带查询条件（搜索关键词）
cloudcc get user . '%7B%22keyword%22%3A%22张三%22%2C%22limit%22%3A50%7D'
```

### 查看单个用户详情

```bash
cloudcc view user <projectPath> <userId>
```

**示例：**

```bash
cloudcc view user . 00520260C00C6FEfsMnT
```

### 编辑/禁用用户

```bash
cloudcc update user <projectPath> <userDataJson>
```

**userDataJson 关键字段：**

| 字段名 | 说明 |
|--------|------|
| `id` | 用户 ID（必填）|
| `loginName` | 登录名（必填）|
| `isusing` | 是否启用：`true`/`false` |
| `lastName` | 姓 |
| `email` | 邮箱 |
| `mobile` | 手机 |
| `profileId` | 简档 ID |
| `role` | 角色 ID |

**示例：**

```bash
# 禁用用户
cloudcc update user . '%7B%22id%22%3A%22xxx%22%2C%22loginName%22%3A%22test%40cloudcc.com%22%2C%22isusing%22%3A%22false%22%7D'

# 启用用户
cloudcc update user . '%7B%22id%22%3A%22xxx%22%2C%22loginName%22%3A%22test%40cloudcc.com%22%2C%22isusing%22%3A%22true%22%7D'
```

### 删除用户

```bash
cloudcc delete user <projectPath> <userId>
```

**示例：**

```bash
cloudcc delete user . 00520260C00C6FEfsMnT
```
