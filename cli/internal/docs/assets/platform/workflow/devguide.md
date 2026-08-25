# Workflow Dev Guide

Use workflow when a requirement maps to setup-web **自动化 > 工作流**. Writes create MetadataService plans; they do not call setup-service directly and must be applied explicitly.

## Commands

```bash
tools/bin/cloudcc get workflow <projectPath>
tools/bin/cloudcc get workflow <projectPath> <filter>
tools/bin/cloudcc detail workflow <projectPath> <workflowId>

tools/bin/cloudcc create workflow <projectPath> @workflow.json
tools/bin/cloudcc update workflow <projectPath> @workflow.json
tools/bin/cloudcc delete workflow <projectPath> <workflowId>
tools/bin/cloudcc enable workflow <projectPath> <workflowId>
tools/bin/cloudcc disable workflow <projectPath> <workflowId>

tools/bin/cloudcc plan msapi <projectPath> workflows @workflow.json create
tools/bin/cloudcc apply msapi <projectPath> <planId>
```

`workflowRule` is accepted as an alias for `workflow`.

## Rule Spec

```json
{
  "id": "acd_account_status_rule",
  "apiName": "account_status_rule",
  "name": "客户状态工作流",
  "description": "客户状态满足条件时触发",
  "targetObjectId": "obj_account",
  "targetObject": "客户",
  "evaluateRuleType": "U",
  "condtionOption": "R",
  "condition": {
    "filter": "1",
    "items": [
      {
        "id": "cond_account_status_rule",
        "fieldId": "field_status",
        "operator": "=",
        "value": "启用"
      }
    ]
  }
}
```

Field notes:

- `id` is the workflow rule id. If omitted on create, MetadataService allocates an `acd` id.
- `targetObjectId` maps to setup-service `targetobjectid` and `tp_sys_workflow.TARGET_OBJECT_ID`.
- `targetObject` is the display label stored by setup-service when saving the rule.
- `evaluateRuleType` uses setup-service values: `I`, `U`, or `A`.
- `condtionOption` intentionally keeps setup-service's spelling. Use `R` for criteria rows and `F` for formula mode.
- Formula mode uses `functionCode` or `formula`.
- Criteria mode accepts `conditions[]`, `conditionMaps[]`, `condition.items[]`, or setup-web-style `conditionVals`.

## Update

```json
{
  "id": "acd_account_status_rule",
  "name": "客户状态工作流更新",
  "targetObjectId": "obj_account",
  "conditionVals": "[{\"id\":\"cond_account_level\",\"fieldId\":\"field_level\",\"operator\":\"!=\",\"value\":\"停用\"}]",
  "bool_filter": "1"
}
```

Update follows setup-service replacement semantics for criteria rows: existing `tp_sys_condition` rows with `relatedId = workflowId` are deleted before the submitted criteria are inserted.

## Activation

```bash
tools/bin/cloudcc enable workflow <projectPath> acd_account_status_rule
tools/bin/cloudcc disable workflow <projectPath> acd_account_status_rule
```

The setup-service endpoint is a toggle, but the CLI exposes explicit enable/disable intent. The MetadataService plan updates `tp_sys_workflow.ISACTIVE` to the requested target state and records the source endpoint as `/api/workFlowSetup/activeWorkflow`.

## Phase Boundary

Phase 1 intentionally does not create or update workflow actions, time-dependent triggers, or recurring triggers. Detail readback includes these relationships so migration operators can detect them. Add them in a later workflow action/trigger sub-scope after setup-service parity evidence exists for:

- `/api/approvalsetup/saveExistingAction`
- `/api/workFlowSetup/saveTimeTrigger`
- `/api/workFlowSetup/saveRecurrenceWorkflow`
- task/email/sms/field-update/trigger action setup endpoints

