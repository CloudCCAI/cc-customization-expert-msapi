# CloudCC 批准过程 CLI 命令说明

## 命令总览

批准过程属于 MetadataService domain，domain 名为 `approval-processes`，常用别名为 `approval`。

```bash
cloudcc normalize approval @approval-process.json
cloudcc validate approval @approval-process.json
cloudcc plan approval @approval-process.json
cloudcc apply msapi <planId>
cloudcc operation msapi <operationId>
cloudcc changes msapi <operationId>
cloudcc rollback-plan msapi <operationId>
cloudcc rollback msapi <operationId>
cloudcc mutate msapi approval update @approval-process-patch.json
```

完整 domain 写法同样可用：

```bash
cloudcc plan msapi approval-processes @approval-process.json
```

## 最小示例

```json
{
  "id": "demo_appr_contract",
  "apiName": "demo_contract_approval",
  "name": "合同审批",
  "targetObject": "contract",
  "active": false,
  "enabled": true,
  "conditions": [
    {
      "id": "demo_apc_amount",
      "fieldId": "discount_amount",
      "operator": "greaterThan",
      "value": "5000"
    }
  ],
  "steps": [
    {
      "id": "demo_apst_manager",
      "name": "经理审批",
      "apiName": "manager_review",
      "index": 1,
      "approverType": "role",
      "approvers": [
        {
          "type": "role",
          "id": "manager"
        }
      ],
      "layoutFields": [
        {
          "id": "demo_apsf_amount",
          "fieldId": "discount_amount",
          "required": true
        }
      ]
    }
  ]
}
```

## 智能体工作流

1. 使用 `normalize` 检查 domain 和基本结构是否能被服务识别。
2. 使用 `validate` 获取缺失字段和警告。
3. 使用 `plan` 生成计划；报告中记录 `planId`、`operationId`、`riskLevel` 和步骤数量。
4. 需要写入时再执行 `apply`。
5. 写入后用 `changes` 查看每个表的 before/after/rollback 快照。
6. 回退前先用 `rollback-plan` 预览，再用 `rollback` 执行。

## 约束

- 新建或测试批准过程建议 `active=false`，待公式、审批人和通知动作全部确认后再启用。
- 运行时审批实例、审批请求和历史记录不是元数据配置，MetadataService 不会写入这些表。
- 条件和动作字段目前按 CloudCC 元数据表通用结构保存，复杂公式和动作执行体仍需结合项目规范复核。
