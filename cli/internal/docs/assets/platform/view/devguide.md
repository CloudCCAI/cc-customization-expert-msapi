# CloudCC 对象列表视图开发指南

## 1. Contract 2 读取

```bash
cloudcc capabilities view <projectPath> [object]
cloudcc list-v2 view <projectPath> <object>
cloudcc detail-v2 view <projectPath> <viewId> <object>
cloudcc candidates view <projectPath> <object> <fields|sortFields|conditionOperators|queues|visibilityMembers|buttons>
```

`object` 可使用对象 ID、API 名或前缀。Contract 2 返回 `contractVersion: 2`、`provider`、规范化 `object`、fingerprint/revision。虚拟视图详情的 `definition` 为 `null`，不会伪造字段或条件。

旧命令保持 Contract 1：

```bash
cloudcc get view <projectPath> [objectId|objectApiName|encodedJsonFilter]
cloudcc detail view <projectPath> <viewId>
```

## 2. Contract 2 写入

创建 spec 示例：

```json
{
  "viewContractVersion": 2,
  "objectId": "<对象ID>",
  "label": "重点客户",
  "apiName": "priority_accounts",
  "filterScope": {"type": "all"},
  "visibility": {"type": "private", "memberIds": []},
  "fields": ["<字段ID>"],
  "conditions": {"expression": "", "items": []},
  "sort": {"fieldId": "<字段ID>", "direction": "desc"},
  "buttonIds": []
}
```

```bash
cloudcc plan msapi <projectPath> object-views @view.json create
cloudcc plan msapi <projectPath> object-views @view-update.json update
cloudcc plan msapi <projectPath> object-views @view-copy.json copy
cloudcc plan msapi <projectPath> object-views @view-delete.json delete
cloudcc set-default view <projectPath> <objectId> <viewId>
cloudcc reorder view <projectPath> <objectId> "<viewId1>;<viewId2>;0;1;2"
cloudcc reorder-public view <projectPath> <objectId> "<viewId1>;<viewId2>;0;1;2"
cloudcc apply msapi <projectPath> <planId>
```

复制 spec 使用 `sourceViewId` 和 `newLabel`；默认生成 private 新视图。默认值与顺序是独立用户偏好，不是 update view 的字段。`reorder-public` 修改对象公共顺序，需要对象视图管理权限。

## 3. 更新与删除规则

- update 是 patch：省略属性表示保留；显式集合表示完整替换。
- `fields` 在 create 时必填且非空，update 时 `fields: []` 非法。
- `conditions.items: []` 与 `buttonIds: []` 表示明确清空；服务端先删除旧 owned rows 再重建。
- charts、kanban 和 dashboard 关系不属于 Contract 2 聚合更新，省略时保留。
- `0..4` 是虚拟只读 ID；任何语言标签变化都不改变其语义。
- 删除前必须检查 dashboard、用户默认值和顺序引用。存在默认引用时需显式提供替代默认值；CLI 不会自动确认破坏性引用处理。
- 写入计划绑定 revision/fingerprint；审核计划后显式 apply，并用 `detail-v2` 回读。
- MSAPI 的用户顺序要求 `apiSvc` 已解析虚拟 ID `3`、`4` 的动态出现状态；状态不可达时失败关闭，避免完整替换顺序时误删动态项。

## 4. Lightning UIAPI

UIAPI 的 `get/detail/create/update/delete/copy/set-default/get-order/reorder/list-public-order/reorder-public view` 均调用项目配置的 `apiSvc`。这些接口由 api-service 转发 main-api；不要把 `setupSvc` 的旧 `/api/view/**` 子集作为完整列表视图合同。

当前 main-api 的 `saveView` 在 `customButtons` 为空时不会清理旧按钮，因此 UIAPI 对 update 的 `buttonIds: []` 明确失败关闭；MSAPI 直连事务支持清空。UIAPI 创建/更新/复制必须详情回读，删除/default/order 必须对应列表或偏好回读，缺少保存 ID 或语义不一致均返回 `view_remote_result_unknown`。

## 5. 变更审计

```bash
cloudcc changes msapi <projectPath> <operationId>
cloudcc rollback msapi <projectPath> <operationId>
cloudcc doc view introduction
cloudcc doc object-views devguide
```
