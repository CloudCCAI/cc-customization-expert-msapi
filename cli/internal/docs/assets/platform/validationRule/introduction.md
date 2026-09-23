# 验证规则

验证规则用于在记录保存前检查数据。表达式结果为 `true` 时，CloudCC 阻止本次保存，并向正在操作记录的用户显示规则配置的错误消息。

CLI 支持查询、详情、表达式校验、创建、更新、删除、执行状态查询和回滚。所有写操作都先生成计划，检查计划后再执行，不会因为运行 `create`、`update` 或 `delete` 立即改动租户。

## 常用流程

```bash
# 1. 查询目标对象字段，确认表达式中的字段名
cloudcc get fields <projectPath> <objectId|objectApiName|objectPrefix>

# 2. 单独校验规则文件
cloudcc validate msapi <projectPath> validation-rules @validation-rules.json create

# 3. 生成创建计划；plan 也会再次校验表达式
cloudcc plan msapi <projectPath> validation-rules @validation-rules.json create

# 4. 执行计划并查询结果
cloudcc apply msapi <projectPath> <planId> @apply.json
cloudcc operation msapi <projectPath> <operationId>

# 5. 回读规则
cloudcc get validationRule <projectPath> <objectId|objectApiName|objectPrefix>
cloudcc detail validationRule <projectPath> <ruleId>
```

`apply.json` 示例：

```json
{
  "async": false
}
```

Windows PowerShell 或 CMD 中建议用 `@file.json` 传 JSON，避免命令行转义改变引号、`&&` 等表达式字符。

## 快捷创建

单条简单规则可使用快捷命令：

```bash
cloudcc create validationRule <projectPath> <objectPrefix> <ruleApiName> <formula> <errorMessage>
```

快捷命令会先校验表达式，再返回 `planId`；新规则默认停用。需要名称、说明、启用状态、错误位置或批量创建时，使用 JSON 文件。

## `$User` 是谁

`$User` 表示规则在记录保存时的当前登录用户，也就是发起本次保存操作的用户。它不是记录所有者、记录创建人或记录最后修改人；只有这些人恰好就是当前操作者时，值才会相同。

常用属性包括 `$User.id`、`$User.name`、`$User.roleId`、`$User.roleName`、`$User.profileId`、`$User.profileName`、`$User.department`、`$User.title`、`$User.email`、`$User.phone` 和 `$User.mobilePhone`。

用户对象上的租户自定义字段也可以动态引用：

```text
$User.<schemefieldName>
```

先运行以下命令查看当前租户真实可用的用户字段：

```bash
cloudcc get fields <projectPath> ccuser
```

从结果中复制字段的 `schemefieldName`，保持大小写完全一致；不要填写中文标签或字段 ID。例如查询结果中存在 `schemefieldName=userid__c` 时，可以写：

```text
$User.userid__c >= 0
```

CLI 会按该用户字段的真实类型校验比较、日期或布尔表达式。字段不存在、名称大小写不一致或类型运算不成立时，校验失败且不会生成计划。

完整 JSON 字段、批量策略、表达式函数、更新、删除和回滚参见开发指南。
