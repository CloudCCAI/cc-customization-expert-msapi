# CloudCC 批准过程

批准过程用于描述记录提交审批后的准入条件、审批步骤、审批人、审批页面字段、相关列表和动作关系。MSAPI 版技能把批准过程视为一个完整元数据意图，不再要求智能体按 UI 页面步骤拆成多次后台接口调用。

## 推荐路径

批准过程创建、调整和回滚优先走 MetadataService：

```bash
cloudcc plan approval @approval-process.json
cloudcc apply msapi <planId>
cloudcc changes msapi <operationId>
cloudcc rollback-plan msapi <operationId>
cloudcc rollback msapi <operationId>
```

也可以使用完整形式：

```bash
cloudcc plan msapi approval-processes @approval-process.json
```

## 意图结构

批准过程 JSON 的核心字段：

| 字段 | 说明 |
|------|------|
| `id` | 批准过程元数据 ID；缺省时由 MetadataService 生成 |
| `apiName` | 稳定 API 名 |
| `name` | 页面显示名称 |
| `targetObject` / `objectId` | 目标对象 ID |
| `active` | 是否启用；新建和测试建议先保持 `false` |
| `conditions` | 进入批准过程的条件 |
| `approvalPageFields` | 审批页面展示字段 |
| `steps` | 审批步骤数组 |
| `relatedLists` | 审批页面相关列表 |
| `actions` | 提交、批准、拒绝等触发动作关系 |

步骤 `steps[]` 支持 `approverType`、`approvers`、`conditions`、`layoutFields`、`actions`、`rejectAction` 等字段。

## 表级范围

MetadataService 当前会生成下列表的可审计变更计划：

| 表 | 作用 |
|----|------|
| `tp_sys_approval` | 批准过程主记录 |
| `tp_sys_approval_step` | 审批步骤 |
| `tp_sys_approval_step_layout` | 审批页面字段 |
| `tp_sys_condition` | 过程和步骤条件 |
| `tp_sys_apralrellist` | 审批相关列表 |
| `tp_sys_apralrellist_fields` | 审批相关列表字段 |
| `tp_sys_actions_relation` | 动作绑定关系 |

运行时实例表不属于元数据变更范围。
