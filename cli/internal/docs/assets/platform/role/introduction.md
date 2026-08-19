# CloudCC 角色使用总结

角色（Role）用于定义 CloudCC 系统中的职能分工，可以对用户进行分组管理并分配不同的权限。

---

## 快速开始（CLI 命令）

### 支持的角色操作

| 操作 | 说明 |
|------|------|
| `create` | 创建新角色 |
| `get` | 查询角色列表 |
| `delete` | 删除角色 |

---

## CLI 命令详解

### 创建角色

创建一个新的 CloudCC 角色。

```bash
cloudcc create role <path> <roleName> [description]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 是 | 项目路径，`.` 表示当前目录 |
| `roleName` | 是 | 角色名称 |
| `description` | 否 | 角色描述 |

**示例：**

```bash
# 创建角色
cloudcc create role . "销售经理"

# 创建带描述的角色
cloudcc create role . "市场专员" "负责市场推广工作"
```

---

### 查询角色列表

获取当前环境中的所有角色列表。

```bash
cloudcc get role <projectPath> [encodedCondJson]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `encodedCondJson` | 否 | URI 编码后的查询条件 JSON |

**示例：**

```bash
# 获取所有角色
cloudcc get role .

# 带查询条件
cloudcc get role . '%7B%22name%22%3A%22%E9%94%80%E5%94%AE%22%7D'
```

---

### 删除角色

删除指定的角色。

```bash
cloudcc delete role <projectPath> <roleId>
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `roleId` | 是 | 角色 ID |

**示例：**

```bash
# 删除指定角色
cloudcc delete role . a0I9D000000XXXXUAI
```

---

## 完整工作流示例

### 场景：为新部门创建 CloudCC 角色

```bash
# 1. 确认项目已初始化（有 cloudcc-cli.config.js）
cat cloudcc-cli.config.js

# 2. 查询现有角色
cloudcc get role .

# 3. 创建新角色
cloudcc create role . "销售经理"
cloudcc create role . "销售代表"

# 4. 验证角色创建成功
cloudcc get role .

# 5. 如需删除
# cloudcc delete role . <roleId>
```

---

*文档版本：1.0 | 最后更新：2026-03-26*
