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
      "ruleContent": "ISBLANK(Amount__c)",
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
