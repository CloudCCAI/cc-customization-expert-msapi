# CloudCC 验证规则使用总结

验证规则（Validation Rule）用于在数据保存前进行校验，确保数据的完整性和正确性。当数据不符合规则时，系统会显示错误信息阻止保存。

---

## 快速开始（CLI 命令）

### 支持的验证规则操作

| 操作 | 说明 |
|------|------|
| `create` | 创建新验证规则 |
| `get` | 查询验证规则列表 |
| `delete` | 删除验证规则 |

---

## CLI 命令详解

### 创建验证规则

创建一个新的 CloudCC 验证规则。

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

**全局变量：**

验证规则公式中已确认可执行的全局变量只有 `$User` 命名空间。setup-service 在 `validateFunction` 校验时会先把下表变量按当前登录用户替换为字符串值，空值替换为空字符串；替换后如果表达式仍包含 `$User`，校验会失败。因此 CLI 文档只承诺下表这些可执行全局变量，目标租户或版本差异仍必须以 `/api/validateRule/validateFunction` 校验结果为准。

| 全局变量 | 说明 |
|----------|------|
| `$User.id` | 当前登录用户 ID。 |
| `$User.name` | 当前登录用户名称。 |
| `$User.roleId` | 当前登录用户所属角色 ID。 |
| `$User.roleName` | 当前登录用户所属角色名称。 |
| `$User.profileId` | 当前登录用户所属简档 ID。 |
| `$User.profileName` | 当前登录用户所属简档名称。 |
| `$User.department` | 当前登录用户部门。 |
| `$User.title` | 当前登录用户职务或头衔。 |
| `$User.email` | 当前登录用户邮箱。 |
| `$User.phone` | 当前登录用户办公电话。 |
| `$User.mobilePhone` | 当前登录用户手机号。 |

setup-web / setup-service 的字段选择器和条件工具中还存在 `$User.<用户对象字段API>` 的动态展示逻辑：后端会把 `user` 对象字段包装为 `$User.<schemefieldName>`，并将 `role`、`profile`、`mobile` 分别映射为 `roleId`、`profileId`、`mobilePhone`，同时补充 `roleName`、`profileName`。这类动态选择项主要服务筛选条件、描述回显和字段选择器；验证规则公式最终能否作为全局变量执行，仍受上表运行时替换清单限制。

源码扫描未发现验证规则公式支持 `$Profile`、`$Organization`、`$Permission` 等其它独立全局变量命名空间；角色和简档信息通过 `$User.roleId`、`$User.roleName`、`$User.profileId`、`$User.profileName` 使用。

示例：

```text
$User.profileId == "aaa000001"
$User.mobilePhone != $User.phone
```

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

**示例：**

```bash
# 非交互式创建
cloudcc create validationRule . "b00" "规则1" "Batch_Size__c__f==5" "数量必须是5"

# 使用 JSON 请求体
cloudcc create validationRule . @validation-rule.json
```

---

### 查询验证规则列表

获取指定对象的所有验证规则列表。

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

---

### 删除验证规则

删除指定的验证规则。

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
# 删除指定规则
cloudcc delete validationRule . 202689E55795D38oAQlN
```

---

## 完整工作流示例

### 场景：为订单对象创建验证规则

```bash
# 1. 确认项目已初始化（有 cloudcc-cli.config.js）
cat cloudcc-cli.config.js

# 2. 查询对象现有的验证规则
cloudcc get validationRule . "b00"

# 3. 创建新验证规则
cloudcc create validationRule . "b00" "规则1" "Batch_Size__c__f==5" "数量必须是5"

# 4. 验证规则创建成功
cloudcc get validationRule . "b00"

# 5. 如需删除
# cloudcc delete validationRule . <ruleId>
```

---

*文档版本：1.0 | 最后更新：2026-03-27*
