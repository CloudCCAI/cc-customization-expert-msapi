# CloudCC 角色 CLI 命令说明

## 支持的命令

| 操作 | 说明 |
|------|------|
| `create` | 创建新角色 |
| `get` | 查询角色列表 |
| `delete` | 删除角色 |

## CLI 命令详解

### 创建角色

```bash
cloudcc create role <path> <roleName> [parentRoleName] [description]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 是 | 项目路径，`.` 表示当前目录 |
| `roleName` | 是 | 角色名称 |
| `parentRoleName` | 否 | 直属上司角色名称（不传则交互式选择）|
| `description` | 否 | 角色描述 |

**示例：**

```bash
# 交互式选择直属上司
cloudcc create role . "销售经理"

# 指定直属上司（非交互式）
cloudcc create role . "销售经理" "CEO"

# 指定直属上司并添加描述
cloudcc create role . "市场专员" "销售总监" "负责市场推广工作"
```

### 查询角色列表

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

### 删除角色

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
cloudcc delete role . a0I9D000000XXXXUAI
```
