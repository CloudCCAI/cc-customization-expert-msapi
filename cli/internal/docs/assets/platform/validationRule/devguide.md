# 验证规则 CLI 开发指南

本文说明如何只使用 CloudCC CLI 管理验证规则，包括查询、校验、计划、执行、回读和回滚。

## 能力清单

| 目标 | 命令 |
|------|------|
| 查询对象上的规则 | `cloudcc get validationRule <projectPath> <objectSelector>` |
| 带名称或 API 名筛选列表 | `cloudcc getList validationRule <projectPath> <objectSelector> <filter>` |
| 查询规则详情 | `cloudcc detail validationRule <projectPath> <ruleId>` |
| 规范化 JSON | `cloudcc normalize msapi <projectPath> validation-rules @spec.json <operation>` |
| 单独校验 | `cloudcc validate msapi <projectPath> validation-rules @spec.json <operation>` |
| 创建计划 | `cloudcc create validationRule ...` 或 `cloudcc plan msapi ... create` |
| 更新计划 | `cloudcc update validationRule <projectPath> @update.json` |
| 删除计划 | `cloudcc delete validationRule <projectPath> <ruleId>` |
| 执行计划 | `cloudcc apply msapi <projectPath> <planId> @apply.json` |
| 查询执行状态 | `cloudcc operation msapi <projectPath> <operationId>` |
| 查询变更快照 | `cloudcc changes msapi <projectPath> <operationId>` |
| 生成回滚计划 | `cloudcc rollback-plan msapi <projectPath> <operationId>` |
| 执行回滚 | `cloudcc rollback msapi <projectPath> <operationId>` |

`objectSelector` 可以是对象 ID、对象 API 名或对象前缀。`normalize` 只检查并规范输入结构；`validate`、`create`、`update` 和 `plan` 在涉及公式时都会编译表达式。

## 推荐的创建流程

先查询字段，再准备规则文件：

```bash
cloudcc get fields . account
```

`validation-rules.json`：

```json
{
  "objectApiName": "account",
  "onExisting": "createOnly",
  "validationRules": [
    {
      "apiName": "employee_count_nonnegative",
      "name": "员工数不能为负数",
      "description": "保存客户前检查员工数",
      "formula": "zys < 0",
      "errorMessage": "员工数必须大于等于零",
      "msgLocation": "top",
      "active": true
    }
  ]
}
```

执行校验、计划、应用和回读：

```bash
cloudcc validate msapi . validation-rules @validation-rules.json create
cloudcc plan msapi . validation-rules @validation-rules.json create
cloudcc apply msapi . <planId> @apply.json
cloudcc operation msapi . <operationId>
cloudcc getList validationRule . account employee_count_nonnegative
cloudcc detail validationRule . <ruleId>
```

`apply.json`：

```json
{
  "async": false
}
```

`validate` 成功表示表达式能通过真实字段和函数语义校验；`plan` 成功表示写入内容已形成可检查计划；只有 `apply` 成功并回读到规则，才表示租户数据已经修改。

## JSON 字段

| 字段 | 是否必需 | 说明 |
|------|----------|------|
| `objectId` / `objectApiName` / `objectPrefix` | 创建必需 | 三选一指定目标对象。批量文件通常放在最外层。 |
| `id` | 更新、删除必需 | 规则 ID，必须来自查询结果。 |
| `apiName` | 创建建议必填 | 规则的稳定 API 名；同一租户中应保持唯一。 |
| `name` | 创建必需 | 管理界面显示名称。 |
| `description` | 可选 | 规则用途说明。 |
| `formula` | 创建必需 | 返回布尔值的验证表达式。也接受 `functionCode`、`expression`、`ruleContent` 或 `formulaText`。 |
| `errorMessage` | 创建必需 | 表达式为 `true` 时显示给当前操作者的消息。 |
| `active` | 可选 | 是否启用。快捷创建默认为 `false`。 |
| `msgLocation` | 可选 | `top` 表示页面顶部；也可传字段 ID，把消息定位到字段。默认 `top`。 |
| `locationField` | 可选 | 需要单独记录错误定位字段时传真实字段 ID。 |

字段 ID 通过 `cloudcc get fields` 查询，不要自行拼接。

## 批量创建与重复项策略

一个文件可在 `validationRules[]` 中放多条同对象规则。CLI 会逐条校验表达式；任意一条表达式无效时，整个请求在生成计划前终止。

`onExisting` 可选值：

| 值 | 行为 |
|----|------|
| `createOnly` | 同 ID、API 名或名称的规则已存在时，将该项标记为预检失败。 |
| `skipExisting` | 已存在的项跳过，其余项继续进入计划。 |

检查计划中的 `batchItemResults`、`batchExecutableCount` 和 `batchPrecheckFailedCount`，确认实际会创建哪些规则。

## 表达式校验

CLI 在创建计划前完成以下检查：

1. 唯一解析目标对象，并读取该对象的标准字段和自定义字段。
2. 检查字段、关联字段路径和 `$User` 字段是否真实存在。
3. 按字段类型检查文本、数值、布尔和日期表达式。
4. 编译完整表达式，确认运算符、括号、参数个数和函数调用可以执行。
5. 再检查规则 JSON 的必填项、对象归属和重复项策略。

校验失败会直接返回原因，不会生成计划。CLI 随包提供表达式编译依赖，但本机需要可用的 JDK 21；JDK 缺失时命令会明确报错。

显式校验结果包含：

- `valid`：所有校验是否通过。
- `formulaValidation.rules[]`：每条公式的对象、摘要和编译状态。
- `metadataValidation`：规则结构、对象归属和业务约束结果。

## 字段与关联字段

普通字段使用 `cloudcc get fields` 返回的字段 API 名：

```text
zys < 0
End_Date__c < Start_Date__c
ISCHANGED(Stage__c)
```

关联对象字段使用查找字段的 `__r.` 路径：

```text
Account__r.annualRevenue > 0
```

CLI 会逐段确认关系字段和末端字段都存在，并按末端字段类型编译；关系路径拼错或中间字段不是查找关系时，不会生成计划。

## 运算符

| 运算符 | 含义 |
|--------|------|
| `+` `-` `*` `/` | 数值运算。 |
| `==` `!=` `<` `>` `<=` `>=` | 等于、不等于和大小比较。 |
| `&&` `||` | 逻辑与、逻辑或。 |
| `(` `)` | 控制运算顺序。 |

表达式最终必须得到布尔值。规则表达式为 `true` 表示数据无效并阻止保存。

## 字段状态与文本函数

| 函数 | 说明 |
|------|------|
| `PRIORVALUE(field)` / `priorValue(field)` | 读取字段修改前的值。 |
| `ISCHANGED(field)` / `isChanged(field)` | 判断字段值是否变化。 |
| `ISNEW()` / `isNew()` | 判断当前保存是否为新建记录。 |
| `BEGINS(field, text)` | 判断字段文本是否以指定内容开始。 |
| `CONTAINS(field, text)` | 判断字段文本是否包含指定内容。 |
| `LEFT(field, count)` | 取左侧指定长度的文本。 |
| `RIGHT(field, count)` | 取右侧指定长度的文本。 |
| `SUBSTITUTE(field, old, new)` | 替换字段文本。 |
| `numberCompare(field, value, operator)` | 数值比较；`operator` 使用 `=`、`!=`、`>`、`>=`、`<` 或 `<=`。 |

`PRIORVALUE`、`ISCHANGED`、`BEGINS`、`CONTAINS`、`LEFT`、`RIGHT`、`SUBSTITUTE`、`numberCompare` 和 `ISNULL` 的字段参数不要加引号，CLI 会按字段引用处理。

## 日期、逻辑与精确数值函数

| 函数 | 说明 |
|------|------|
| `DATE(year, month, day)` | 创建日期值。 |
| `NOW()` / `TODAY()` | 当前日期时间 / 当前日期。 |
| `DAY(date)` / `MONTH(date)` / `YEAR(date)` | 读取日期的日、月、年。 |
| `AND(...)` / `OR(...)` / `NOT(value)` | 逻辑与、逻辑或、逻辑非。 |
| `IF(condition, whenTrue, whenFalse)` | 根据条件返回两个值之一。 |
| `ISNULL(field)` | 判断字段是否为空。 |
| `PRECISEADD(a, b[, scale])` | 精确加法，默认 `scale=2`。 |
| `PRECISESUBTRACT(a, b[, scale])` | 精确减法，默认 `scale=2`。 |
| `PRECISEMULTIPLY(a, b[, scale])` | 精确乘法，默认 `scale=2`。 |
| `PRECISEDIVIDE(a, b[, scale])` | 精确除法，默认 `scale=2`。 |

函数名大小写按表中写法使用；只有表中同时列出的大小写别名可以互换。

## `$User` 当前用户变量

`$User` 表示规则执行时发起记录保存的当前登录用户。它不会自动指向记录的 `ownerid`、`createbyid` 或 `lastmodifybyid`。

固定属性：

| 变量 | 含义 |
|------|------|
| `$User.id` / `$User.name` | 当前用户 ID / 名称。 |
| `$User.roleId` / `$User.roleName` | 当前用户角色 ID / 名称。 |
| `$User.profileId` / `$User.profileName` | 当前用户简档 ID / 名称。 |
| `$User.department` / `$User.title` | 当前用户部门 / 职务。 |
| `$User.email` / `$User.phone` / `$User.mobilePhone` | 当前用户联系信息。 |

动态用户字段：

```text
$User.<schemefieldName>
```

先查询用户字段：

```bash
cloudcc get fields <projectPath> ccuser
```

从输出中复制精确的 `schemefieldName`，不要使用字段标签或字段 ID，并保持大小写一致。例如：

```text
$User.userid__c >= 0 && zys < 0
$User.center__c == "North"
```

动态字段按真实字段类型参与编译：数值字段可做大小比较，布尔字段可直接作为条件，日期字段可与日期值比较，文本或选项字段可与字符串比较。

## 更新

更新文件只需要规则 `id` 和要修改的字段：

```json
{
  "id": "val000000000000000001",
  "formula": "$User.center__c == \"North\" || zys < 0",
  "errorMessage": "员工数必须大于等于零",
  "active": true
}
```

```bash
cloudcc update validationRule . @validation-rule-update.json
cloudcc apply msapi . <planId> @apply.json
cloudcc detail validationRule . val000000000000000001
```

包含公式的更新会先读取规则所属对象并重新编译。只改名称、说明、错误消息、错误位置或启用状态时，不重复编译未变化的公式。更新使用局部修改语义：未传字段保持原值，规则 API 名和创建审计信息不会被重新生成。

## 删除与回滚

删除先生成高风险计划，检查目标 ID 后再执行：

```bash
cloudcc delete validationRule . <ruleId>
cloudcc apply msapi . <planId> @apply.json
cloudcc getList validationRule . <objectSelector> <ruleApiName>
```

应用后列表中应查不到该规则。需要恢复已应用的创建、更新或删除时，先用原 `operationId` 查看变更并生成回滚计划：

```bash
cloudcc changes msapi . <operationId>
cloudcc rollback-plan msapi . <operationId>
cloudcc rollback msapi . <operationId>
```

回滚完成后再次运行 `get` 或 `detail` 核对最终状态。
