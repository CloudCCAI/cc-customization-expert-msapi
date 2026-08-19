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
