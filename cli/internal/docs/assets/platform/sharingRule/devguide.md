# CloudCC 共享规则 CLI 与 MSAPI 命令说明

> 对应 one-setup-web 接口：`{setupSvc}/api/sharingSettings/queryRule`

## 1. 查询命令

```bash
# 按对象查询共享规则
cloudcc get sharingRule <objid> [projectPath]
```

## 2. 参数说明

### 2.1 查询：`cloudcc get sharingRule <objid> [projectPath]`

| 参数 | 必填 | 说明 |
| :--- | :--- | :--- |
| `objid` | 是 | 对象 id |
| `projectPath` | 否 | 项目根目录，默认当前目录（可传 `.`） |

## 3. 示例

### 3.1 按对象查询规则

```bash
cloudcc get sharingRule account .
```

## 4. MetadataService 写入域

共享规则低代码元数据应纳入 MSAPI domain：`sharing-rules`。

```bash
cloudcc plan msapi sharing-rules @sharing-rule.json
cloudcc apply msapi <planId>
cloudcc changes msapi <operationId>
cloudcc rollback-plan msapi <operationId>
cloudcc rollback msapi <operationId>
```

示例 spec：

```json
{
  "sharingRuleId": "share_sales_region",
  "name": "华北客户共享",
  "objectId": "account",
  "sourceType": "criteria",
  "targetType": "roleAndSubordinates",
  "targetId": "role_sales_north",
  "accessLevel": "read",
  "conditions": [
    {"fieldId": "field_region", "operator": "=", "value": "华北"}
  ]
}
```

## 5. 验收要求

- 共享规则只扩展访问，不替代 role 组织层级模型。
- 写入前应确认对象默认共享策略、角色层级、目标角色/公用小组/队列和字段条件。
- 上线前必须用目标用户账号验证：应该可见的记录可见，不应该可见的记录不可见。
- 共享规则主表和列在不同租户中可能存在差异，生产 apply 前应通过 scanner 或后台页面核验表结构。
