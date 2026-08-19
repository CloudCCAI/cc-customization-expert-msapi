# CloudCC 自动化模型

## 定位

CloudCC 自动化能力用于在数据变化、流程推进或时间到达时自动执行规则、通知、审批、代码或外部调用。

## 事实依据

手册和当前技能文档确认的自动化能力包括：

- 验证规则
- 查重过滤器
- 工作流和批准
- 动作
- 触发器
- 类
- 定时类
- 定时作业
- Webhook（手册在动作中提到）

## 能力分工

### 验证规则

保存记录时执行。如果满足错误条件，则终止保存并显示错误信息。适合数据质量和必填/范围/逻辑一致性校验。

### 查重过滤器

在新建或编辑业务数据时，按对象、字段与条件检测是否与已有记录重复，从而提示或阻断保存。

### 工作流

手册说明工作流可基于对象和条件触发规则，并指定工作流操作。手册列出的工作流操作包括任务、电子邮件通知、短信通知、字段更新。

### 批准过程

用于多级人工审批。可配置进入条件、审批步骤、批准人、最终批准/拒绝/调回相关动作。

### 动作

手册说明动作可与工作流规则、批准过程关联。手册列出任务、电子邮件通知、字段更新、触发器、Webhook 五类动作。

### 触发器

手册说明触发器是在记录插入、新增、删除等操作之前或之后执行的一段 Java 代码。当前技能文档覆盖 `beforeInsert`、`beforeUpdate`、`beforeUpsert`、`beforeDelete`、`afterInsert`、`afterUpdate`、`afterUpsert`、`afterDelete`、commit 后时机以及 approval 等触发时间。

### 类

服务端 Java 业务代码，可用于自定义页面和触发器。适合封装可复用业务能力。

### 定时类与定时作业

定时类承载按时间执行的 Java 逻辑；定时作业管理作业程序、频率、开始结束日期和执行时间。

## 选择原则

| 需求 | 优先能力 |
|---|---|
| 保存前阻止错误数据 | 验证规则 |
| 防止客户/联系人/线索重复 | 查重过滤器 |
| 保存后发通知或更新字段 | 工作流/动作 |
| 多级人工决策 | 批准过程 |
| 复杂记录事件逻辑 | 触发器 |
| 可复用后端服务 | 类 |
| 周期性后台扫描处理 | 定时类 + 定时作业 |
| 调外部系统 | OpenAPI/类/Webhook/sidecar，按场景确认 |

## 相关文档

```bash
cloudcc doc platform/validationRule introduction
cloudcc doc platform/dupeCatcher introduction
cloudcc doc platform/triggers introduction
cloudcc doc platform/classes introduction
cloudcc doc platform/timer introduction
cloudcc doc platform/scheduleJob introduction
cloudcc doc platform/integrationArchitecture introduction
```
