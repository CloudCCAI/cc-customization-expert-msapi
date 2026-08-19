# Permission（权限集管理）

Permission 模块用于管理 CloudCC 权限集的分配功能，包括查看权限集列表、查看已分配用户、添加分配用户和删除分配用户。

在项目实施中，权限集属于 MSAPI `permissions` domain：权限集定义、对象权限、字段权限、系统权限和用户分配都应优先通过 MetadataService `plan/apply/changes/rollback` 编排；旧 `get/add/remove` 命令继续用于查询和兼容性操作。

## 功能介绍

- **查看权限集列表**: 获取系统中的所有权限集
- **查看已分配用户**: 查看指定权限集已分配的用户列表
- **添加分配用户**: 为指定权限集分配用户
- **删除分配用户**: 从指定权限集中移除用户

## 命令

命令统一为 **`cloudcc <动作> <资源> …`**；本模块资源名为 **`permission`**。

### MSAPI 计划

```bash
cloudcc plan msapi permissions @permission-set.json
cloudcc apply msapi <planId>
cloudcc changes msapi <operationId>
cloudcc rollback-plan msapi <operationId>
```

### 获取权限集列表

```bash
cloudcc get permission <path> [viewId] [page] [pageSize] [searchKeyWord]
```

**参数说明：**
- `path` - 项目路径（可选，默认为当前目录）
- `viewId` - 视图ID（可选，不传则自动获取）
- `page` - 页码（可选，默认为1）
- `pageSize` - 每页数量（可选，默认为100000）
- `searchKeyWord` - 搜索关键词（可选）

**示例：**

```bash
# 获取所有权限集列表
cloudcc get permission .

# 使用指定视图获取权限集
cloudcc get permission . "00000000000000000000"

# 搜索权限集
cloudcc get permission . "" 1 100 "测试"
```

### 查看已分配用户

```bash
cloudcc assign permission <path> <permsetId>
```

**参数说明：**
- `path` - 项目路径（可选，默认为当前目录）
- `permsetId` - 权限集ID（必需）

**示例：**

```bash
# 查看指定权限集的已分配用户
cloudcc assign permission . "cac20258F0E4ABBnTxwi"
```

**输出示例：**

```
权限集: 古语云
ID: cac20258F0E4ABBnTxwi

已分配 6 个用户:

  用户ID                  姓名      登录名               简档              角色
  --------------------  --------  --------------------  ----------------  ----------------
  00520263DB1540FTPmXK  我重新注册  421865903teet@qq.com  Cloudcc Partner 简档  青青草原艾艾合作伙伴用户
  005202652FF2FA8dltk1  auto行    autoTest@cloudcc.com  系统管理员        测试
  ...
```

### 添加分配用户

```bash
cloudcc add permission <path> <permsetId> [userIds...]
```

**参数说明：**
- `path` - 项目路径（可选，默认为当前目录）
- `permsetId` - 权限集ID（必需）
- `userIds` - 要分配的用户ID列表（可选，不传则进入交互式选择）

**示例：**

```bash
# 交互式选择用户进行分配
cloudcc add permission . "cac20258F0E4ABBnTxwi"

# 直接指定用户ID进行分配（多个用户用空格分隔）
cloudcc add permission . "cac20258F0E4ABBnTxwi" "00520260C00C6FEfsMnT" "0052026C0A60504jNcbV"
```

**交互式选择模式：**

如果不指定 userIds，命令会进入交互式选择模式，展示所有可用用户供选择：

```
? 请选择要分配的用户（使用空格键选择，回车确认）: (Press <space> to select, <a> to toggle all, <i> to invert selection)
 ❯◯ 张三 (zhangsan@example.com) - 00520260C00C6FEfsMnT
  ◯ 李四 (lisi@example.com) - 0052026C0A60504jNcbV
  ◯ 王五 (wangwu@example.com) - 0052026D0A60504jNcbW
```

### 删除分配用户

```bash
cloudcc remove permission <path> <permsetId> [userIds...]
```

**参数说明：**
- `path` - 项目路径（可选，默认为当前目录）
- `permsetId` - 权限集ID（必需）
- `userIds` - 要移除的用户ID列表（可选，不传则进入交互式选择）

**示例：**

```bash
# 交互式选择用户进行移除
cloudcc remove permission . "cac20258F0E4ABBnTxwi"

# 直接指定用户ID进行移除（多个用户用空格分隔）
cloudcc remove permission . "cac20258F0E4ABBnTxwi" "00520260C00C6FEfsMnT" "0052026C0A60504jNcbV"
```

**交互式选择模式：**

如果不指定 userIds，命令会进入交互式选择模式，展示当前已分配的用户供选择：

```
? 请选择要删除的用户（使用空格键选择，回车确认）: (Press <space> to select, <a> to toggle all, <i> to invert selection)
 ❯◯ 张三 (zhangsan@example.com) - 00520260C00C6FEfsMnT
  ◯ 李四 (lisi@example.com) - 0052026C0A60504jNcbV

? 确定要从权限集中移除 2 个用户吗？ (y/N)
```

## 完整使用流程示例

```bash
# 1. 查看所有权限集
cloudcc get permission .

# 2. 查看某个权限集的已分配用户
cloudcc assign permission . "cac20258F0E4ABBnTxwi"

# 3. 为权限集添加用户（交互式选择）
cloudcc add permission . "cac20258F0E4ABBnTxwi"

# 4. 再次查看已分配用户，确认添加成功
cloudcc assign permission . "cac20258F0E4ABBnTxwi"

# 5. 移除某些用户（交互式选择）
cloudcc remove permission . "cac20258F0E4ABBnTxwi"
```

## 数据结构

### 权限集对象

```json
{
    "name": "古语云",
    "licence": "CloudCC 用户",
    "id": "cac20258F0E4ABBnTxwi",
    "sysadmin": "0"
}
```

**字段说明：**
- `name` - 权限集名称
- `licence` - 许可证类型
- `id` - 权限集唯一标识
- `sysadmin` - 是否为系统管理员权限（"1"表示是，"0"表示否）

### 用户对象

```json
{
    "loginname": "421865903teet@qq.com",
    "profileid": "aaa000002",
    "rolename": "青青草原艾艾合作伙伴用户",
    "roleid": "2026A3D78466C42UokMO",
    "profilename": "Cloudcc Partner 简档",
    "name": "我重新注册",
    "alias": null,
    "id": "00520263DB1540FTPmXK"
}
```

**字段说明：**
- `id` - 用户唯一标识
- `name` - 用户姓名
- `loginname` - 登录名（邮箱）
- `profilename` - 简档名称
- `profileid` - 简档ID
- `rolename` - 角色名称
- `roleid` - 角色ID
- `alias` - 别名
