# Workflow

Workflow covers the setup-web menu path **自动化 > 工作流**. In MSAPI mode it is a low-code metadata domain named `workflows`; the CLI resource aliases are `workflow` and `workflowRule`.

The first supported scope is workflow rule CRUD, activation state, and criteria/formula condition metadata. Immediate actions, time-dependent triggers, recurring triggers, and inline action creation are read back in detail output but are not created by the phase-1 workflow writer.

## Setup-Web Mapping

| Setup-web action | Setup-service endpoint | CLI/MSAPI mapping |
| --- | --- | --- |
| 列表查询 | `/api/workFlowSetup/listworkflows` | `cloudcc get workflow <projectPath>` |
| 详情查询 | `/api/workFlowSetup/detailWorkflow` | `cloudcc detail workflow <projectPath> <workflowId>` |
| 打开新建页 | `/api/workFlowSetup/newWorkflow` | Use object/field scan before authoring the spec |
| 打开编辑页 | `/api/workFlowSetup/newWorkflow` | `cloudcc detail workflow` for readback context |
| 新建保存 | `/api/workFlowSetup/saveNewButton` | `cloudcc create workflow <projectPath> @workflow.json` |
| 编辑保存 | `/api/workFlowSetup/saveWorkflow` | `cloudcc update workflow <projectPath> @workflow.json` |
| 删除 | `/api/workFlowSetup/deleteWorkflow` | `cloudcc delete workflow <projectPath> <workflowId>` |
| 启用/停用 | `/api/workFlowSetup/activeWorkflow` | `cloudcc enable workflow ...` / `cloudcc disable workflow ...` |

## Metadata Tables

Phase-1 workflow plans touch:

- `tp_sys_workflow`
- `tp_sys_condition`
- delete cleanup also removes root rows from `tp_sys_actions_relation`, `tp_sys_workflowdepend`, and `tp_sys_schedular`

Detail readback exposes the workflow row, criteria rows, action relation rows, time-dependent trigger rows, and recurring scheduler rows so operators can see whether a rule has child automation that should be handled separately.

