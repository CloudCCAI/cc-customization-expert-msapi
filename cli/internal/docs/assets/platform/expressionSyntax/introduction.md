# 目标环境表达式字段后缀

CloudCC 的 metadata API 名与表达式字段令牌是两套概念：

- 对象和字段的 metadata `apiName` 默认不加任何后缀，例如 `Account`、`annualRevenue`。
- 公式字段、验证规则和工作流公式条件中的字段令牌使用目标环境配置的后缀。平台通常默认为 `__c`，也可配置为 `__f` 等值。
- 关系路径中间段固定使用 `__r`，不随字段后缀变化，例如 `account__r.id__f`。

CLI 仅在本次操作确实涉及上述表达式时按需识别目标后缀，并按目标环境缓存；普通 metadata 操作不会触发识别。MSAPI、UIAPI、SaaS 与私有云遵循同一用户规则。

完整范围、缓存行为、示例与失败处理见：

```bash
cloudcc doc expressionSyntax devguide
```
