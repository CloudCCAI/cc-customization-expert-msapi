# CloudCC 查重过滤器 CLI 开发指南

适用版本：技能/CLI `2.3.8+`，MetadataService `1.2.8+`。

## 1. 概念与边界

条件和规则不能互换：`conditions`/`conditionVals` 写入 `tp_sys_condition`，限定参与查重的记录范围；
`rules` 写入 `tp_sys_dupecatcherule`，定义怎样算重复。多条规则按 AND 组合。过滤器主体控制新增和更新
命中后的提示、允许保存或阻止保存策略。

`dupeCatcher` 是唯一一级 Domain。规则是过滤器拥有的子资源，所有规则命令都必须同时提供 `filterId`
和需要时的 `ruleId`，以验证父子归属。

## 2. 发现与读取

```bash
cloudcc capabilities dupeCatcher <projectPath>
cloudcc get dupeCatcher <projectPath> [filter]
cloudcc detail dupeCatcher <projectPath> <filterId>
cloudcc detail-rule dupeCatcher <projectPath> <filterId> <ruleId>

cloudcc candidates dupeCatcher <projectPath> objects [keyword]
cloudcc candidates dupeCatcher <projectPath> rule-fields <filterId> [keyword]
cloudcc candidates dupeCatcher <projectPath> condition-fields <objectId|apiName> [keyword]
```

Contract 2 详情返回 `filter`、`conditions`、`rules` 和 `fingerprint`。兼容键 `dupeCatcher`、`relations`
仍然保留，但新客户端不应依赖数据库列名。

## 3. 过滤器生命周期

推荐使用 `@file`，不需要 `encodeURI(JSON.stringify(...))`。

```bash
cloudcc create dupeCatcher <projectPath> @filter.json
cloudcc update dupeCatcher <projectPath> <filterId> @patch.json
cloudcc delete dupeCatcher <projectPath> <filterId>
```

示例 `filter.json`：

```json
{
  "name": "客户查重",
  "objectId": "Account",
  "active": false,
  "profileScoped": false,
  "errorMessage": "发现重复客户",
  "insertOperation": "check",
  "insertIsSave": false,
  "insertIsTips": true,
  "updateOperation": "check",
  "updateIsSave": false,
  "updateIsTips": true,
  "conditions": {
    "logic": "1 AND 2",
    "items": [
      {"fieldId":"status","operator":"equals","value":"active","sequence":1},
      {"fieldId":"createdate","operator":"greaterOrEqual","value":"2026-01-01","sequence":2}
    ]
  }
}
```

更新采用 patch 语义：省略条件表示保留。清空条件必须显式提交空集合，不能用缺失字段表达删除。

### 3.1 条件输入形状

新脚本应使用上面的 `conditions.logic` + `conditions.items[]`。也支持 setup-service 原生形状：

```json
{
  "conditionVals": {
    "data": [
      {"fieldId":"field_status","op":"e","val":"active"},
      {"fieldId":"field_createdate","op":"bt","val":"2026-01-01","endval":"2026-12-31"}
    ],
    "filter": "1 AND 2"
  }
}
```

`conditionVals` 可以是 JSON 对象或该对象的 JSON 字符串；旧数组形式仍可读。别名为
`operator`/`op`、`value`/`val`、`endValue`/`endvalue`、`sequence`/`seq`、
`logic`/`filter`/`boolFilter`。计划会将字段解析为真实 ID，将操作符转成存储码，并把顺序压缩为
`1..N`。

- 省略 `conditions` 和 `conditionVals`：保留已有条件。
- `"conditions":{"items":[],"logic":""}` 或 `"conditionVals":{"data":[],"filter":""}`：清空条件。
- 非空条件省略 logic 时默认为 `1 AND 2 ...`。显式 logic 只能使用条件序号、`AND`、`OR`和括号，
  且必须覆盖所有条件。

### 3.2 条件字段与操作符

先运行 `condition-fields`。每个候选项返回 `type`、`effectiveType`、`eligible`、
`supportedOperators`、`valueKind`、`options` 和 `exclusionReason`。只能使用 `eligible=true`的字段；
公式/汇总字段只有在能确定结果类型时才合格。

| 字段族 / valueKind | 可用操作符 |
|---|---|
| 文本、单选、查找 / `text`、`select` | `equals`, `notEquals`, `lessThan`, `greaterThan`, `lessOrEqual`, `greaterOrEqual`, `contains`, `notContains`, `startsWith` |
| 数字、百分比、货币、评分 / `number` | `equals`, `notEquals`, `lessThan`, `greaterThan`, `lessOrEqual`, `greaterOrEqual` |
| 日期、日期时间、时间 / `date`、`dateTime`、`time` | 上述六个比较操作符，加 `between` |
| 复选框 / `boolean` | `equals`, `notEquals` |
| 多选列表 / `multiSelect` | `equals`, `notEquals`, `includes`, `excludes` |
| 地理定位 / `location` | `within` |

友好名与存储码对照：

| 友好名 | 码 | 兼容输入 |
|---|---|---|
| `equals` | `e` | `=`, `e` |
| `notEquals` | `n` | `!=`, `<>`, `n` |
| `startsWith` / `contains` / `notContains` | `s` / `c` / `k` | 对应单字母码 |
| `lessThan` / `greaterThan` | `l` / `g` | `<` / `>` |
| `lessOrEqual` / `greaterOrEqual` | `m` / `h` | `<=` / `>=` |
| `includes` / `excludes` | `u` / `x` | `in` / `notIn` |
| `within` | `o` | `location` |
| `between` | `bt` | `bt` |

setup-web 源码中存在没有公共标签和服务端语义的孤立 `i` token；CLI 不将它宣告为可用操作符。

### 3.3 值格式

| valueKind | `value` / `endValue` |
|---|---|
| `text` | 字符串；空字符串沿用平台空值比较语义 |
| `select` | 优先用 `options[].value`；候选已返回选项时也可输入 label/ID，计划会归一化为存储值 |
| `multiSelect` | JSON 数组或逗号分隔值；`includes`/`excludes` 不拆成多条条件 |
| `number` | 十进制数字字符串或 JSON number |
| `boolean` | `true` / `false`（兼容 `1` / `0` / `真` / `假`） |
| `date` / `dateTime` / `time` | 平台可解析时间值；日期同时兼容 today/yesterday/tomorrow 等平台相对值 |
| `location` | `半径,纬度,经度`，三项均为数字 |

`between` 必须同时提供 `value` 和 `endValue`。详情回读中 `operator` 是友好名，
`operatorCode` 是库内原始码；`relations.tp_sys_condition` 仍保留 Contract 1 原始列形状。

## 4. 详情页规则生命周期

```bash
cloudcc create-rule dupeCatcher <projectPath> <filterId> @rule.json
cloudcc update-rule dupeCatcher <projectPath> <filterId> <ruleId> @patch.json
cloudcc delete-rule dupeCatcher <projectPath> <filterId> <ruleId>
```

完全匹配：

```json
{
  "fieldId": "field_account_name",
  "matchOption": "exact",
  "blankMatches": false
}
```

后 N 位匹配：

```json
{
  "fieldId": "field_account_phone",
  "matchOption": "lastNletters",
  "matchLength": 8,
  "blankMatches": false
}
```

匹配方式：

| matchOption | matchLength | 含义 |
|---|---:|---|
| `exact` | 禁止 | 完全相等 |
| `contains` | 禁止 | 包含匹配 |
| `firstNletters` | 正整数、必填 | 前 N 位匹配 |
| `lastNletters` | 正整数、必填 | 后 N 位匹配 |

兼容输入 `firstletters`/`firstLetters` 会归一化到 `matchLength`。字段必须属于过滤器目标对象；汇总、
公式、附件、长文本、富文本、记录类型、币种、文件、地址及系统身份字段不能作为重复规则字段。

## 5. MSAPI 计划、回读与回滚

友好写命令只创建计划。检查输出中的 `planId` 后显式执行：

```bash
cloudcc apply msapi <projectPath> <planId>
cloudcc changes msapi <projectPath> <operationId>
cloudcc rollback msapi <projectPath> <operationId>
```

规则计划保证：

- `create-rule` 只插入规则行，不更新过滤器主体；
- `update-rule` 和 `delete-rule` 同时绑定 `ruleId` 与 `filterId`；
- 父子不匹配、字段不合格、匹配方式或长度非法时在规划阶段失败；
- apply 后必须通过详情/规则详情做语义回读，不能仅凭 HTTP 200 判定成功。

## 6. Provider 行为

- MSAPI：MetadataService 直接执行租户关系型元数据计划，支持审计和回滚。
- Lightning UIAPI：对象、规则字段、条件字段分别调用 `insetNewfilter`、`insetNewrule`、
  `workFlowSetup/newWorkflow`；条件字段输出会归一化为 Contract 2，同时保留 `providerResponse`用于审计。
  规则写入调用 `nsaverule/deleterule` 并回读详情。
- Horizontal UIAPI：无已验证持久化合同，返回 `dupe_catcher_horizontal_uiapi_unsupported`。
- Universal：根据项目 provider 选择上述链路，不允许跨 provider 静默回退。

## 7. 常见错误

| 错误码 | 处理 |
|---|---|
| `dupe_catcher_not_found` | 重新查询过滤器 ID |
| `dupe_rule_not_found` | 重新读取详情中的规则 ID |
| `dupe_rule_parent_mismatch` | 使用规则真实所属的 filterId |
| `dupe_rule_field_ineligible` | 从 `rule-fields` 候选项重新选择 |
| `dupe_rule_match_option_invalid` | 使用四种受支持匹配方式 |
| `dupe_rule_match_length_invalid` | 为前/后 N 位提供正整数 matchLength |
| `dupe_condition_field_invalid` | 从 `condition-fields` 选择同一对象的合格字段 |
| `dupe_condition_operator_invalid` | 使用候选项 `supportedOperators` 中的操作符 |
| `dupe_condition_value_invalid` / `dupe_condition_range_invalid` | 按 `valueKind` 修正值；`between` 同时传起止值 |
| `dupe_condition_logic_invalid` | 只使用 1..N、AND/OR 和括号，并覆盖全部条件 |
| `dupe_condition_candidate_response_invalid` | Lightning setup-service 返回不可归一化；不要把原始响应当作合同 |
| `dupe_catcher_horizontal_uiapi_unsupported` | 切换到 MSAPI 或 Lightning UIAPI |

旧位置式命令，例如 `create dupeCatcher . rule <json>`，不再作为规范合同。迁移到 `create-rule` 等
明确 action，避免 `rule` 被通用解析器当作 selector 或 JSON。
