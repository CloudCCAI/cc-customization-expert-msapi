# 字段依赖 CLI 使用指南

本文说明如何使用 `fieldDependency` 查询候选字段和现有配置，创建或更新自定义映射，切换全局依赖，删除配置，并在 MSAPI 模式下审计和回滚变更。

## 使用前检查

```text
cloudcc --version
cloudcc doctor provider <projectPath>
cloudcc domain fieldDependency
```

- MSAPI 要求 MetadataService `1.1.81` 或更高版本。写命令只生成计划，必须显式 apply 才会生效。
- Lightning UIAPI 直接提交到平台配置接口，不生成 MetadataService ledger 或 rollback。
- horizontal UIAPI 当前不可用，会在发网前明确拒绝；horizontal 项目请使用 MSAPI。

## 命令总览

```text
cloudcc get fieldDependency <projectPath> <object>
cloudcc candidates fieldDependency <projectPath> <object>
cloudcc detail fieldDependency <projectPath> <object> <controlField> <dependentField>
cloudcc create fieldDependency <projectPath> <@dependency.json>
cloudcc update fieldDependency <projectPath> <@dependency.json>
cloudcc applyGlobalDependency fieldDependency <projectPath> <object> <controlField> <dependentField>
cloudcc delete fieldDependency <projectPath> <object> <controlField> <dependentField>
```

`object` 可传对象 ID、API 名或对象前缀。MSAPI 中的字段和值选择器可使用候选结果返回的稳定 ID 或语言无关 code。Lightning UIAPI 应使用 candidates/detail 返回的字段 ID 和默认值；不要使用随语言变化的显示标签。

文档和命令也接受 `field-dependency`、`field-dependencies` 别名，示例统一使用 `fieldDependency`。

## 完整操作流程

### 1. 查询候选字段

```text
cloudcc candidates fieldDependency . contact
```

控制字段只会包含支持的复选框或单选列表；依赖字段只会包含支持的单选列表或多选列表。候选结果还会排除不属于当前对象、选项数量不符合限制或无法安全配置的字段。

保存候选结果中的字段 ID/API 名和选项 ID/code。后续配置不要自行构造 ID，也不要把中文、英文等本地化显示标签作为稳定值身份。

### 2. 查询当前配置和详情

```text
cloudcc get fieldDependency . contact
cloudcc detail fieldDependency . contact customer_type contact_level
```

重点确认：

- 控制字段和依赖字段是否为预期字段。
- 当前模式是 `custom` 还是 `global`。
- 自定义模式下每个控制值对应的完整允许值集合。
- 全局模式下两个字段是否仍绑定到具有依赖关系的全局选项列表。

### 3. 准备自定义映射文件

新建 `field-dependency.json`：

```json
{
  "object": "contact",
  "controlField": "customer_type",
  "dependentField": "contact_level",
  "mode": "custom",
  "mappings": [
    {
      "controlValue": "enterprise",
      "allowedValues": ["key", "normal"]
    },
    {
      "controlValue": "individual",
      "allowedValues": ["normal"]
    }
  ]
}
```

`mappings` 表示该字段对变更后的完整映射，不是增量追加。一个控制值只能出现一次，同一行的 `allowedValues` 也不能重复。

### 4. 创建、审阅并执行 MSAPI 计划

```text
cloudcc create fieldDependency . @field-dependency.json
cloudcc apply msapi . <planId>
cloudcc operation msapi . <operationId>
cloudcc changes msapi . <operationId>
cloudcc detail fieldDependency . contact customer_type contact_level
```

MSAPI 模式下，`create` 返回待执行计划。先审阅对象、字段对、模式、完整替换提示和 `planId`，确认无误后再 apply。apply 返回 `operationId`；随后用 operation 查看状态、用 changes 查看实际变更，并通过 detail 回读。

Lightning UIAPI 模式下，`create` 会直接提交，不返回可 apply 的 MetadataService `planId`。

### 5. 更新自定义映射

修改同一 JSON 文件后执行：

```text
cloudcc update fieldDependency . @field-dependency.json
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc detail fieldDependency . contact customer_type contact_level
```

update 与 create 都按对象、控制字段和依赖字段这一自然键执行幂等替换。数组中遗漏的旧映射会被移除。如果计划生成后字段、选项、已有依赖或全局绑定发生变化，apply 会拒绝陈旧计划；重新查询并生成新计划，不要反复执行旧 `planId`。

### 6. 清空自定义映射

提交显式空数组：

```json
{
  "object": "contact",
  "controlField": "customer_type",
  "dependentField": "contact_level",
  "mode": "custom",
  "mappings": []
}
```

然后运行 update，并在 MSAPI 模式下 apply。空 `mappings` 表示清空该字段对的自定义值映射，但仍保留字段对配置；它与 delete 不同。

### 7. 使用已有全局依赖

先确认两个字段各自绑定了全局选项列表，并且两个全局选项列表之间已经配置依赖关系。可以使用 JSON：

```json
{
  "object": "contact",
  "controlField": "country",
  "dependentField": "province",
  "mode": "global"
}
```

执行：

```text
cloudcc create fieldDependency . @global-dependency.json
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc detail fieldDependency . contact country province
```

也可以使用字段对快捷命令：

```text
cloudcc applyGlobalDependency fieldDependency . contact country province
```

MSAPI 下快捷命令仍然只生成计划；Lightning UIAPI 下会直接提交。全局模式不接受 `mappings`。切换到全局模式会清理该对象字段对的自定义映射，因此执行前必须确认现有自定义映射可以被替换。

### 8. 删除字段依赖

```text
cloudcc delete fieldDependency . contact customer_type contact_level
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc get fieldDependency . contact
```

delete 删除该字段对的对象级依赖配置；它不是“清空 mappings”。Lightning UIAPI 会直接删除，不适用后续 apply/changes 命令。

### 9. 回滚 MSAPI 操作

```text
cloudcc rollback-plan msapi . <operationId>
cloudcc rollback msapi . <operationId>
cloudcc detail fieldDependency . contact customer_type contact_level
```

先审阅 rollback-plan，再执行 rollback，并通过 detail/get 回读。Lightning UIAPI 没有 MetadataService `operationId`，因此不适用此回滚流程。

## 参数和安全规则

- 控制字段和依赖字段必须属于同一对象，且不能是同一个字段。
- 控制字段支持复选框和单选列表；依赖字段支持单选列表和多选列表。返回中的内部类型代码仅用于诊断，不建议作为用户输入。
- 控制字段选项数必须满足平台限制；当前不接受达到或超过 300 个选项的控制字段。
- 字段和值选择器必须唯一命中；未知、重复或有歧义的值会被拒绝。
- 自定义模式必须显式提供 `mappings`，包括有意清空时的空数组。
- 全局模式禁止提供 `mappings`，并要求两个字段的全局选项列表之间已有有效依赖。
- 系统会拒绝自身依赖、直接循环和间接循环。
- 记录类型的选项分配不属于字段依赖；请使用 recordType 的选项值分配命令。

## 常见问题

- “字段不是候选字段”：重新运行 candidates，确认字段属于当前对象并满足控制字段或依赖字段类型要求。
- “选项值未知或不唯一”：使用 candidates/detail 返回的稳定 ID 或语言无关 code，不要使用本地化显示标签。
- “控制字段选项过多”：更换控制字段或缩减其有效选项；字段依赖不会绕过平台上限。
- “检测到循环依赖”：检查当前对象的全部依赖链，而不只是本次字段对；移除导致回路的关系后重新生成计划。
- “全局依赖不存在或已失效”：确认两个字段仍绑定到正确的全局选项列表，并先在全局选项列表之间建立依赖。
- “计划已发生漂移”：字段、选项或依赖配置在 plan 后发生变化，重新运行 get/candidates/detail 并生成新计划。
- “horizontal UIAPI 不支持”：为项目配置 MetadataService 并选择 MSAPI Provider。
- apply 后结果与预期不一致：使用 operation 和 changes 检查真实执行结果，再通过 detail/get 回读；不要只根据计划内容判断已生效。

能力概览使用：

```text
cloudcc doc fieldDependency introduction
```
