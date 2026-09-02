# CloudCC 验证规则 CLI 命令说明

## 支持的命令

| 操作 | 说明 |
|------|------|
| `create` | 创建新验证规则 |
| `get` | 查询验证规则列表 |
| `delete` | 删除验证规则 |

## CLI 命令详解

### 创建验证规则

```bash
cloudcc create validationRule <path> <objectPrefix> <ruleName> <ruleContent> <errorMessage>
cloudcc create validationRule <path> @body.json
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 是 | 项目路径，`.` 表示当前目录 |
| `objectPrefix` | 是 | 对象前缀，如 `b00` |
| `ruleName` | 是 | 规则名称 |
| `ruleContent` | 是 | 规则内容，如 `Batch_Size__c__f==5` |
| `errorMessage` | 是 | 错误提示信息 |

Go 版 CLI 不实现交互式录入。若需要更完整的请求体，使用 raw JSON 或 `@body.json`。

> 后端接口以 setup 前端包确认为准：查询使用 `{setupSvc}/api/validateRule/queryByPrefix`，公式校验使用 `{setupSvc}/api/validateRule/validateFunction`，创建/更新使用 `{setupSvc}/api/validateRule/save`，删除使用 `{setupSvc}/api/validateRule/delete`。若租户或版本接口有差异，先用只读查询和公式校验确认端点，再进入受控写入窗口。

### 公式表达式说明

`ruleContent` / `functionCode` 是验证规则公式。CLI 写入前必须使用 setup-service 的 `{setupSvc}/api/validateRule/validateFunction` 或等价平台校验确认公式可编译通过；`save` 接口本身不等价于公式校验。下面清单按 setup-service 当前 `validateFunction` 注入的实际函数整理，不能把未通过 `validateFunction` 的表达式当作可用能力。

字段引用通常使用字段 API 名，如 `Amount__c`、`End_Date__c`。部分函数会由 setup-service 在校验时把字段参数改写为字段 API 字符串，例如 `ISCHANGED(Stage__c)`、`PRIORVALUE(Stage__c)`、`BEGINS(Name__c, "A")`。

**运算符：**

| 运算符 | 说明 |
|--------|------|
| `+` | 加法；在服务端表达式编译语义下也可用于字符串拼接。 |
| `-` | 减法。 |
| `*` | 乘法。 |
| `/` | 除法。 |
| `(` `)` | 分组，控制表达式优先级。 |
| `==` | 等于比较。 |
| `!=` | 不等于比较。 |
| `<` | 小于比较。 |
| `>` | 大于比较。 |
| `<=` | 小于等于比较。 |
| `>=` | 大于等于比较。 |
| `&&` | 逻辑与。 |
| `||` | 逻辑或。 |
| `&` | setup-web 面板展示为连接符；服务端最终按 `validateFunction` 编译结果为准，建议优先使用 `+` 做字符串拼接。 |

**函数：**

| 函数 | 说明 |
|------|------|
| `PRIORVALUE(field)` / `priorValue(field)` | 返回更新前记录中指定字段的旧值；创建场景旧记录为空时返回 `null`。 |
| `ISCHANGED(field)` / `isChanged(field)` | 判断指定字段新旧值是否不同；新旧记录缺失时返回 `false`。 |
| `ISNEW()` / `isNew()` | 判断当前记录是否为新建；旧记录为空时返回 `true`。 |
| `BEGINS(field, compare_text)` | 判断指定字段当前值是否以 `compare_text` 开头；字段值或比较值为空时当前实现返回 `true`。 |
| `CONTAINS(field, compare_text)` | 判断指定字段当前值是否包含 `compare_text`；字段值或比较值为空时当前实现返回 `true`。 |
| `LEFT(field, num_chars)` | 返回指定字段当前值左侧 `num_chars` 个字符；长度小于 0 返回空字符串，长度超过原值时按原值长度截取。 |
| `RIGHT(field, num_chars)` | 返回指定字段当前值右侧 `num_chars` 个字符；长度小于 0 返回空字符串，长度超过原值时按原值长度截取。 |
| `SUBSTITUTE(field, old_text, new_text)` | 将指定字段当前值中的 `old_text` 替换为 `new_text`。 |
| `DATE(year, month, day)` | 返回日期值；参数直接传给服务端 `Calendar.set(year, month, day)`。 |
| `NOW()` / `now()` | 返回当前日期时间。 |
| `TODAY()` / `today()` | 返回当前日期零点。 |
| `DAY(date)` | 返回日期中的日。 |
| `MONTH(date)` | 返回日期中的月份，范围为 1 到 12。 |
| `YEAR(date)` | 返回日期中的年份。 |
| `HOUR(date)` | 返回日期中的小时；当前服务端实现使用 `Calendar.HOUR`。 |
| `MINUTE(date)` | 返回日期中的分钟。 |
| `SECOND(date)` | 返回日期中的秒。 |
| `MILLISECOND(date)` | 返回日期中的毫秒。 |
| `ADDMONTHS(date, months)` | 在指定日期上增加 `months` 个月并返回日期。 |
| `DATETIMEVALUE(date)` | 将日期格式化为 `yyyy-MM-dd HH:mm:ss` 字符串。 |
| `DATEVALUE(date)` | 将日期格式化为 `yyyy-MM-dd` 字符串。 |
| `TIMENOW()` | 返回当前时间字符串，格式为 `HH:mm:ss`。 |
| `TIMEVALUE(date)` | 将日期格式化为 `HH:mm:ss` 字符串。 |
| `WEEKDAY(date)` | 当前服务端实现返回 `Calendar.HOUR`，不是单独的星期值；使用前必须以目标环境 `validateFunction` 结果为准。 |
| `AND(logical1, logical2, ...)` | 所有布尔参数均为 `true` 时返回 `true`，否则返回 `false`。 |
| `OR(logical1, logical2, ...)` | 任一布尔参数为 `true` 时返回 `true`，否则返回 `false`。 |
| `IF(logical_test, value_if_true, value_if_false)` | 条件为 `true` 时返回第二个参数，否则返回第三个参数。 |
| `ISNULL(expression)` | 判断字符串表达式是否为 `null` 或空字符串。 |
| `NOT(logical)` | 当前服务端实现返回传入布尔值本身；使用前必须以目标环境 `validateFunction` 结果为准。 |

> 注意：当前 setup-service 验证规则函数中未看到 `ISBLANK`、`PRECISEADD`、`PRECISESUBTRACT`、`PRECISEMULTIPLY`、`PRECISEDIVIDE` 的实现，不应在 CLI 文档或示例中作为验证规则可用函数承诺。setup-web 面板中 `BEGIN` 的函数名应按服务端实际能力使用 `BEGINS`。

### 受控写入要求

- 写入前必须先确认目标对象前缀、候选公式、错误提示和是否启用。
- 公式必须先通过平台公式校验或等价只读预检。
- 执行保存前必须保留可回滚信息；写入后用只读查询确认规则存在、状态符合预期，再做 MetadataService compare 或后台页面验收。

**示例：**

```bash
# 非交互式创建
cloudcc create validationRule . "b00" "规则1" "Batch_Size__c__f==5" "数量必须是5"

# 使用 JSON 请求体
cloudcc create validationRule . @validation-rule.json
```

### 批量创建验证规则

一次要在同一个对象下创建多个验证规则时，使用 MetadataService plan/apply。文件顶层写 `objectId`、`objectApiName` 或 `objectPrefix` 指定目标对象，并在 `validationRules[]` 中写每条规则。批量创建是对象级能力：同一个文件只能作用于一个对象，数组项不能覆盖到其它对象，也不能和其它 domain 混在同一次计划里提交。

示例 `validation-rules-batch.json`：

```json
{
  "objectPrefix": "b00",
  "onExisting": "createOnly",
  "validationRules": [
    {
      "id": "val_contract_amount_required",
      "name": "合同金额必填",
      "ruleContent": "ISNULL(Amount__c)",
      "errorMessage": "合同金额不能为空",
      "isActive": "true"
    },
    {
      "id": "val_contract_end_after_start",
      "name": "结束日期晚于开始日期",
      "ruleContent": "End_Date__c < Start_Date__c",
      "errorMessage": "结束日期必须晚于开始日期"
    }
  ]
}
```

执行命令：

```bash
cloudcc plan msapi <projectPath> validation-rules @validation-rules-batch.json create
cloudcc apply msapi <projectPath> <planId> '{"async":true}'
cloudcc operation msapi <projectPath> <applyId>
cloudcc get validationRule <projectPath> <objectPrefix>
```

`validationRule` / `validation-rules` 都可作为 `plan msapi` 的 domain 参数。批量计划会逐项检查同批重复、目标对象已有同 ID / API 名 / 名称验证规则、以及数组项是否声明了其它对象。公式、错误提示、启用状态等仍按单条验证规则的字段填写；批量不会降低公式校验要求，提交前仍建议先使用平台公式校验或等价只读预检。

`onExisting` 支持：

| 策略 | 行为 |
|------|------|
| `createOnly` | 默认策略；目标已存在时该项标记为 `FAILED_PRECHECK`，其它无关项继续生成步骤。 |
| `skipExisting` | 目标已存在时跳过该项，plan metadata 记录为 `SKIPPED`。 |

调用方应读取 plan metadata 中的 `batchItemResults`、`batchExecutableCount`、`batchPrecheckFailedCount`。`batchItemResults[].status` 可能是 `PLANNED`、`SKIPPED` 或 `FAILED_PRECHECK`；预检失败项不会生成 SQL 步骤。如果整批都没有可执行项，`apply` 会失败，避免提交空计划。

### 查询验证规则列表

```bash
cloudcc get validationRule <projectPath> <objectPrefix>
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `objectPrefix` | 是 | 对象前缀，如 `b00` |

**示例：**

```bash
# 获取对象 b00 的所有验证规则
cloudcc get validationRule . "b00"
```

### 删除验证规则

```bash
cloudcc delete validationRule <projectPath> <ruleId>
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `ruleId` | 是 | 规则 ID |

**示例：**

```bash
cloudcc delete validationRule . 202689E55795D38oAQlN
```
