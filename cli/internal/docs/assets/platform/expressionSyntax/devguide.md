# 目标环境表达式字段后缀开发指南

## 1. 名称规则

对象和字段的 metadata `apiName` 默认不添加 `__c`、`__f` 或其他后缀：

```json
{"objectApiName":"Account","apiName":"annualRevenue"}
```

只有表达式中的字段令牌使用目标环境配置的字段后缀。平台通常默认为 `__c`，但目标环境可配置为 `__f` 等其他合法值；不要在项目或脚本中写死默认值。

关系路径中间段固定为 `__r`。假设目标字段后缀为 `__f`：

```text
amount__f
account__r.id__f
account__r.region__r.regionCode__f
```

## 2. CLI 何时识别和缓存

CLI 采用按需识别：只有首次执行公式字段、验证规则或工作流公式条件相关操作时才读取目标配置；普通对象、普通字段、工作流普通属性及其他 Domain 不会触发。

识别结果按目标环境隔离，并同时使用进程内缓存和用户级全局缓存。有效缓存会被后续 CLI 进程复用；缓存过期、损坏或目标身份变化时自动重新识别。缓存不写入项目配置，也不保存明文目标地址或认证信息。

MSAPI 模式由 MetadataService 统一解析目标配置并缓存，CLI 仍保留用户级缓存；UIAPI 模式由 CLI 的目标适配链路解析。两种模式对用户呈现相同结果。

目标识别失败、结果为空、格式非法或与当前目标不一致时，相关表达式操作失败关闭，CLI 不会猜测 `__c` 继续写入；不涉及表达式的操作不受影响。

## 3. 本次适用范围

| Domain | 受影响内容 |
|---|---|
| `fields` | 公式字段 `formulaText`，以及字段删除/影响分析中的公式引用保护 |
| `validationRule` | 验证规则公式表达式 |
| `workflow` | 公式模式的工作流条件 `functionCode` |

`workflowFieldUpdate`、批准过程、报表、OpenAPI、classes、timer、triggers 等不在本次自动后缀能力范围内，不能据此推断它们必须使用同一后缀规则。

## 4. 三类表达式示例

以下示例都假设目标后缀为 `__f`。

公式字段：

```text
amount__f - discount__f
account__r.annualRevenue__f
```

验证规则：

```text
endDate__f < startDate__f
ISCHANGED(stage__f)
account__r.annualRevenue__f > 0
```

工作流公式条件：

```text
status__f = 'ACTIVE'
```

## 5. 项目迁移与排错

- 项目 JSON 中对象和字段 metadata API 名始终保存无后缀值。
- 表达式文本迁移到另一目标环境前，先让 CLI 针对新目标重新识别后缀并审阅计划。
- 不要对任意文本全局替换 `__c`；只修改被表达式解析器识别为字段引用的令牌。
- 关系中间段始终保留 `__r`，只调整末端字段令牌。
- 遇到识别失败时，检查当前项目激活环境、登录状态、目标服务可达性和目标身份是否切换；不要手工伪造缓存。
