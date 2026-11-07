# CloudCC 查重过滤器 CLI 开发指南

适用版本：技能/CLI `2.3.6+`，MetadataService `1.2.6+`。

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
  "conditions": [
    {"fieldId":"status","operator":"=","value":"active","sequence":1,"logic":"1"}
  ]
}
```

更新采用 patch 语义：省略条件表示保留。清空条件必须显式提交空集合，不能用缺失字段表达删除。

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
  `workFlowSetup/newWorkflow`；规则写入调用 `nsaverule/deleterule` 并回读详情。
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
| `dupe_catcher_horizontal_uiapi_unsupported` | 切换到 MSAPI 或 Lightning UIAPI |

旧位置式命令，例如 `create dupeCatcher . rule <json>`，不再作为规范合同。迁移到 `create-rule` 等
明确 action，避免 `rule` 被通用解析器当作 selector 或 JSON。
