# CloudCC 自定义设置操作指南

直接查看 CLI 文档：`cloudcc doc platform/customSetting`

---

## 1. 入口与列表页

进入路径：`设置 -> 开发者空间 -> 自定义设置`。

列表页可查看标签、可见性、设置类型、描述等信息，并支持新建、编辑、删除。

---

## 2. 新建/编辑主对象

主对象常见输入项：

- 标签
- 对象名
- 设置类型（列表/层次结构）
- 可见性
- 描述

保存后会回到列表页并显示最新结果。

---

## 3. 详情页与字段管理

进入详情页后通常包含两块：

1. 自定义设置主对象信息
2. 自定义设置字段列表

在字段列表区域可完成新增、编辑、删除字段。

---

## 4. 字段类型（新建字段）

当前常用字段类型代码：

- `U` URL
- `P` 百分比
- `c` 币种
- `H` 电话
- `E` 电子邮件
- `B` 复选框
- `D` 日期
- `F` 日期和时间
- `N` 数字
- `S` 文本
- `X` 长文本

---

## 5. 常用 CLI 命令

### 主对象

#### 1) 查询自定义设置列表
命令：
```bash
cloudcc get customSetting <projectPath> [encodedCondJson]
```
请求参数：
| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| body | object | 非必须 | `{}` | `encodedCondJson` 解码后的请求体 |

#### 2) 查询自定义设置详情（含字段列表）
命令：
```bash
cloudcc detail customSetting <projectPath> <settingId>
```
请求参数：
| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| objid | string | 必须 | — | 对应命令中的 `settingId` |

#### 3) 新建/更新自定义设置主对象
命令：
```bash
cloudcc create customSetting <projectPath> <encodedBodyJson>
```
请求参数：
| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| tpSysObjectVO | object | 必须 | — | 主对象载体，见下表子字段 |
**tpSysObjectVO 子字段：**
| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| accessable | string | 视业务 | — | 可见性（如 `"2"` 表示公用） |
| dataType | string | 视业务 | — | 设置类型：`L` 列表，`H` 层次结构 |
| id | string | 视业务 | `""` | 新建传空；编辑传记录 id |
| label | string | 视业务 | — | 标签 |
| remark | string | 非必须 | — | 描述 |
| schemetableName | string | 视业务 | — | 对象名 |

#### 4) 编辑前回显主对象表单数据
命令：
```bash
cloudcc modify customSetting <projectPath> <settingId>
```
请求参数：
| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| objid | string | 必须 | — | 对应命令中的 `settingId` |

#### 5) 删除自定义设置主对象
命令：
```bash
cloudcc delete customSetting <projectPath> <settingId>
```
请求参数：
| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| objid | string | 必须 | — | 对应命令中的 `settingId` |

### 字段

#### 6) 进入字段编辑（取字段编辑所需数据）
命令：
```bash
cloudcc editCustomSettingField customSetting <projectPath> <queryrecordid>
```
请求参数：
| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| queryrecordid | string | 必须 | — | 字段行 id |

#### 7) 保存字段（新增/编辑字段）
命令：
```bash
cloudcc saveCustomSettingField customSetting <projectPath> <encodedBodyJson>
```
请求参数：
| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| tpSysSchemetableVO | object | 必须 | — | 字段对象，见下方子字段表 |
| fdtype | string | 必须 | — | 字段类型代码（如 `U` 表示 URL） |
**tpSysSchemetableVO 子字段：**
| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| schemetableId | string | 必须 | — | 所属自定义设置 id |
| id | string | 非必须 | — | 字段 id；编辑时传 |
| nameLabel | string | 必须 | — | 字段显示名称 |
| apiname | string | 必须 | — | 字段 API 名 |
| edittype | string | 非必须 | — | 字段编辑类型（如 URL 打开方式） |
| schemefieldLength | string | 非必须 | — | 字段长度 |
| defaultValue | string | 非必须 | — | 默认值 |
| datafieldRef | string | 非必须 | — | 数据字段引用名 |
| isDeleted | string | 非必须 | `"0"` | 删除标记 |
| fieldState | string | 非必须 | `"enable"` | 字段状态 |
| iscustom | string | 非必须 | `"1"` | 是否自定义字段 |
| schemefieldType | string | 必须 | — | 字段类型代码（如 `U`） |
| schemefieldName | string | 必须 | — | 字段物理名/存储名 |

#### 8) 删除自定义字段
命令：
```bash
cloudcc deleteCustomSettingField customSetting <projectPath> <fieldId> <settingId>
```
请求参数：
| 参数名称 | 类型 | 是否必须 | 默认值 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| id | string | 必须 | — | 对应命令中的 `fieldId` |
| objid | string | 必须 | — | 对应命令中的 `settingId` |

---

## 6. Checklist

- [ ] 已确认目标自定义设置（标签/对象名）
- [ ] 保存主对象前已校验设置类型与可见性
- [ ] 字段新增/编辑时已校验字段类型与 API 名
- [ ] 删除前已确认字段/主对象无业务依赖

---

