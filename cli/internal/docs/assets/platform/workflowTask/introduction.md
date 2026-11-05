# 工作流任务 Domain

`workflowTask` 是工作流任务动作的一级 CLI Domain，MetadataService 规范名为 `workflow-tasks`，对应 setup-web `/#/settings/action/workflow-tasks`。它管理“工作流触发后创建一条任务”的可复用动作定义，不等同于钉钉待办、日程、业务对象记录或 Codex 任务。

## 适用场景

- 工作流满足条件后，为记录所有者、创建人、指定用户、角色或关系用户字段创建跟进任务；
- 统一维护任务主题、说明、状态、优先级、负责人、相对到期日和负责人通知；
- 在多个即时工作流中复用同一个任务动作；
- 迁移或审计现有任务动作，并查看它被哪些工作流或触发器引用。

## 能力与边界

该 Domain 支持 `capabilities`、列表、详情、候选项、引用影响、创建、部分更新和受保护删除。MSAPI 要求 MetadataService `1.1.105+`，直接读写关系库元数据，不调用 setup-service。所有写命令只生成计划，用户审阅后才能 `apply msapi`。

任务 Domain 只拥有 `tp_sys_actions_task` 中的动作定义。`tp_sys_actions_relation` 由 `workflow` Domain 管理：

- 已有动作只能绑定到即时工作流；
- 时间或周期作用域必须通过 `createAction workflow` 创建并绑定新任务；
- 解绑只删关系，不删任务定义；
- 被引用任务不能删除，也不能直接改到另一个目标对象；
- 工作流删除时保留可复用任务定义。

平台托管动作只读。CLI 不接受底层数据库列、审计字段或调用方生成的关系 ID。

## 建议工作流

```bash
cloudcc candidates workflowTask . Account
cloudcc create workflowTask . @task.json
# 记录创建计划返回的 target（acb...），它是后续查询和绑定使用的任务动作 ID
cloudcc apply msapi . <planId>
cloudcc detail workflowTask . <acb-action-id>
cloudcc bindAction workflow . @bind-task.json
```

删除前先运行：

```bash
cloudcc impact workflowTask . <acb-action-id>
```

任务动作表没有 API 名列。不要编造 `atask_*`：创建由 MetadataService 生成 `acb...` ID，详情、更新、影响、删除和工作流绑定都使用创建计划/readback 返回的真实 ID。

命令、全部字段、负责人类型、到期日、绑定/解绑、错误码和验收流程见：

```bash
cloudcc doc workflowTask devguide
```
