# 工作流 Domain

`workflow`（别名 `workflowRule`）是工作流规则的一级 CLI Domain，MetadataService 规范名为 `workflows`，对应 setup-web `/#/settings/workflowRule/workflow-rule`。它管理规则条件、启停状态、即时动作关系、时间触发器、周期触发器及其动作关系。

## 适用场景

- 按对象字段条件或公式定义自动化规则；
- 创建、修改、启用、停用和删除工作流；
- 把任务、电子邮件通知、字段更新绑定到即时作用域；
- 创建时间/周期触发器，并为触发器创建动作；
- 审计一条规则拥有的条件、触发器和动作图。

## 四个 Domain 的职责

- `workflowTask`：任务动作定义；
- `workflowEmailAlert`：邮件通知动作定义；
- `workflowFieldUpdate`：字段更新动作定义；
- `workflow`：规则、触发器和 `tp_sys_actions_relation` 绑定关系。

动作定义与关系严格分离。删除工作流或解绑动作不会删除可复用动作定义；删除动作前必须通过对应动作 Domain 的 `impact` 确认引用为零。

## 能力与边界

MSAPI 工作流完整能力要求 MetadataService `1.1.105+`。所有读写由 MetadataService 直接访问关系库，不调用 setup-service；写命令只生成计划。

已有动作只能绑定即时作用域。时间和周期作用域只允许创建并绑定新任务/邮件动作；时间作用域也允许新字段更新，周期作用域禁止字段更新。启用前会校验已知动作存在且目标对象一致。

工作流删除会按依赖顺序清理触发器动作关系、条件、即时关系和触发器，再删除规则根记录。动作根记录保留。

## 建议工作流

```bash
cloudcc create workflow . @workflow.json
cloudcc apply msapi . <planId>
cloudcc detail workflow . <workflowId>
cloudcc bindAction workflow . @bind.json
cloudcc createTimeTrigger workflow . @time-trigger.json
cloudcc enable workflow . <workflowId>
```

规则字段、条件替换、绑定/解绑、时间与周期调度字段、启停校验、删除顺序和错误码见：

```bash
cloudcc doc workflow devguide
```
