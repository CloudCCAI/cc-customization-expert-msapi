# CloudCC 权限集 CLI 命令说明

命令格式为 **`cloudcc <动作> permission …`**（动作即下表「操作」列）。

## 支持的命令

| 操作 | 说明 |
|------|------|
| `get` | 查询权限集列表 |
| `assign` | 查看权限集已分配的用户 |
| `add` | 为权限集添加分配用户 |
| `remove` | 从权限集移除分配用户 |

## MSAPI 实施方式

项目实施时，权限集应作为 `permissions` domain 纳入 MetadataService。该域可同时编排 profile 系统权限分配，以及 permission set 的定义、对象权限、字段权限和用户分配。

```bash
cloudcc plan msapi permissions @permission-set.json
cloudcc apply msapi <planId>
cloudcc changes msapi <operationId>
cloudcc rollback-plan msapi <operationId>
```

权限集规格示例：

```json
{
  "permissionSetId": "ps_sales_export",
  "name": "销售导出权限集",
  "description": "销售导出补充权限",
  "userIds": ["user_001", "user_002"],
  "objectPermissions": [
    {
      "category": "object",
      "objectId": "account",
      "objectOperateType": "1,1,0,0,0,0"
    }
  ],
  "fieldPermissions": [
    {
      "objectId": "account",
      "fieldId": "field_cost",
      "visible": true,
      "readonly": true
    }
  ]
}
```

执行前应按 profile、role、sharingRule 的整体权限矩阵复核，避免权限集替代基础简档或绕过数据权限设计。

## CLI 命令详解

### 查询权限集列表

```bash
cloudcc get permission <path> [viewId] [page] [pageSize] [searchKeyWord]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 否 | 项目路径，`.` 表示当前目录，默认为当前目录 |
| `viewId` | 否 | 视图ID，不传则自动获取 |
| `page` | 否 | 页码，默认为 1 |
| `pageSize` | 否 | 每页数量，默认为 100000 |
| `searchKeyWord` | 否 | 搜索关键词 |

**示例：**

```bash
# 获取所有权限集列表
cloudcc get permission .

# 使用指定视图获取权限集
cloudcc get permission . "00000000000000000000"

# 搜索权限集
cloudcc get permission . "" 1 100 "测试"
```

**输出示例：**

```
找到 17 个权限集:

  ID                    名称             许可证            系统管理员
  --------------------  ---------------  ----------------  ----------
  aabbudgeting2024ACgL  利润云标准权限集  Budgeting         是
  cac202530FE035CHCFGi  cloudcc222       CloudCC 用户      否
  cac2025F039C6A5DQGrt  TestMFA22        CloudCC 用户      否
  cac20258F0E4ABBnTxwi  古语云           CloudCC 用户      否
  ...
```

### 查看已分配用户

```bash
cloudcc assign permission <path> <permsetId>
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 否 | 项目路径，`.` 表示当前目录，默认为当前目录 |
| `permsetId` | 是 | 权限集ID |

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

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 否 | 项目路径，`.` 表示当前目录，默认为当前目录 |
| `permsetId` | 是 | 权限集ID |
| `userIds` | 否 | 要分配的用户ID列表，不传则进入交互式选择 |

**示例：**

```bash
# 交互式选择用户进行分配
cloudcc add permission . "cac20258F0E4ABBnTxwi"

# 直接指定单个用户ID进行分配
cloudcc add permission . "cac20258F0E4ABBnTxwi" "00520260C00C6FEfsMnT"

# 直接指定多个用户ID进行分配（用空格分隔）
cloudcc add permission . "cac20258F0E4ABBnTxwi" "00520260C00C6FEfsMnT" "0052026C0A60504jNcbV"
```

**交互式选择模式：**

```
正在获取用户列表...

? 请选择要分配的用户（使用空格键选择，回车确认）:
 ❯◯ 张三 (zhangsan@example.com) - 00520260C00C6FEfsMnT
  ◯ 李四 (lisi@example.com) - 0052026C0A60504jNcbV
  ◯ 王五 (wangwu@example.com) - 0052026D0A60504jNcbW
(Move up and down to reveal more choices)

正在为权限集 cac20258F0E4ABBnTxwi 分配 2 个用户...

✓ 用户分配成功
  权限集ID: cac20258F0E4ABBnTxwi
  分配用户数: 2
```

### 移除分配用户

```bash
cloudcc remove permission <path> <permsetId> [userIds...]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 否 | 项目路径，`.` 表示当前目录，默认为当前目录 |
| `permsetId` | 是 | 权限集ID |
| `userIds` | 否 | 要移除的用户ID列表，不传则进入交互式选择 |

**示例：**

```bash
# 交互式选择用户进行移除
cloudcc remove permission . "cac20258F0E4ABBnTxwi"

# 直接指定单个用户ID进行移除
cloudcc remove permission . "cac20258F0E4ABBnTxwi" "00520260C00C6FEfsMnT"

# 直接指定多个用户ID进行移除（用空格分隔）
cloudcc remove permission . "cac20258F0E4ABBnTxwi" "00520260C00C6FEfsMnT" "0052026C0A60504jNcbV"
```

**交互式选择模式：**

```
正在获取已分配用户列表...

? 请选择要删除的用户（使用空格键选择，回车确认）:
 ❯◯ 张三 (zhangsan@example.com) - 00520260C00C6FEfsMnT
  ◯ 李四 (lisi@example.com) - 0052026C0A60504jNcbV
(Move up and down to reveal more choices)

? 确定要从权限集中移除 2 个用户吗？ (y/N) y

正在从权限集 cac20258F0E4ABBnTxwi 移除 2 个用户...

✓ 用户移除成功
  权限集ID: cac20258F0E4ABBnTxwi
  移除用户数: 2
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

## 数据字段说明

### 权限集对象字段

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `id` | string | 权限集唯一标识 |
| `name` | string | 权限集名称 |
| `licence` | string | 许可证类型 |
| `sysadmin` | string | 是否为系统管理员权限（"1"表示是，"0"表示否） |

### 用户对象字段

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `id` | string | 用户唯一标识 |
| `name` | string | 用户姓名 |
| `loginname` | string | 登录名（邮箱） |
| `profilename` | string | 简档名称 |
| `profileid` | string | 简档ID |
| `rolename` | string | 角色名称 |
| `roleid` | string | 角色ID |
| `alias` | string | 别名 |

## 相关 API 接口

| 接口 | 说明 |
|------|------|
| `POST /api/permissionGroup/queryPermsetsList` | 获取权限集视图列表 |
| `POST /api/permissionGroup/listAJAX` | 查询权限集列表 |
| `POST /api/permissionGroup/queryUserlistBypermsetsid` | 查询权限集已分配用户 |
| `POST /api/permissionGroup/addUsersetup` | 为权限集添加用户 |
| `POST /api/permissionGroup/deleteUsersetup` | 从权限集移除用户 |
| `POST /api/usermange/queryUser` | 获取用户视图 |
| `POST /api/usermange/queryUserList` | 查询用户列表 |
