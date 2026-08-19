# CloudCC 用户使用总结

用户（User）用于定义 CloudCC 系统中的登录账号，每个用户都关联一个简档以确定其权限。

---

## 快速开始（CLI 命令）

### 支持的用户操作

| 操作 | 说明 |
|------|------|
| `create` | 创建新用户 |
| `get` | 查询用户列表 |
| `delete` | 删除用户 |

---

## CLI 命令详解

### 创建用户

创建一个新的 CloudCC 用户。

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

---

### 查询用户列表

获取当前环境中的所有用户列表。

```bash
cloudcc get user <projectPath> [encodedCondJson]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `encodedCondJson` | 否 | URI 编码后的查询条件 JSON |

**示例：**

```bash
# 获取所有用户
cloudcc get user .

# 带查询条件
cloudcc get user . '%7B%22status%22%3A%22active%22%7D'
```

---

### 删除用户

删除指定的用户。

```bash
cloudcc delete user <projectPath> <userId>
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `userId` | 是 | 用户 ID |

**示例：**

```bash
# 删除指定用户
cloudcc delete user . a0I9D000000XXXXUAI
```

---

## 完整工作流示例

### 场景：为新员工创建 CloudCC 账号

```bash
# 1. 确认项目已初始化（有 cloudcc-cli.config.js）
cat cloudcc-cli.config.js

# 2. 查询可用简档，确定要分配的权限
cloudcc get profile .

# 3. 创建新用户
cloudcc create user . "王五" a0I9D000000XXXXUAI "wangwu@example.com"

# 4. 验证用户创建成功
cloudcc get user .

# 5. 如需删除
# cloudcc delete user . <userId>
```

---

*文档版本：1.0 | 最后更新：2026-03-26*
