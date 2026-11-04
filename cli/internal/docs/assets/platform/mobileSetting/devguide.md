# 移动端设置 CLI 完整指南

## 1. 适用范围与执行模型

资源名为 `mobileSetting`，兼容别名 `mobileSettings`、`mobile-setting`、`mobile-settings`。七类资源均支持 `get`、`candidates` 和 `update`；MSAPI 另提供 `capabilities`、`apply`、`changes`、`rollback`。

提供者行为不同：

- **MSAPI**：读取走 MetadataService；`update` 只编译计划。计划保存执行前状态哈希，应用时重新校验，避免用陈旧计划覆盖新配置。
- **Lightning UIAPI**：直接调用 setup-service；更新后强制回读并比较目标状态。
- **Horizontal UIAPI**：当前没有可靠契约，CLI 明确拒绝，不会猜测接口或静默降级。

项目配置中的 provider/package 选择方式与其他 CLI 资源一致。修改前建议先执行 `get` 和 `candidates`，不要从页面显示文案猜 ID。

## 2. 通用命令

```powershell
# 查看 MetadataService 暴露的资源和动作
cloudcc capabilities mobileSetting <projectPath>

# 读取标准化状态
cloudcc get mobileSetting <projectPath> <resource>

# 获取合法候选项
cloudcc candidates mobileSetting <projectPath> <resource>

# MSAPI：生成计划；Lightning：写入并回读
cloudcc update mobileSetting <projectPath> <resource> <encodedSpecJson|@file>
```

推荐使用 `@file`，便于评审和避免 shell 转义。文件必须是单个 JSON 对象。MSAPI 的完整闭环：

```powershell
cloudcc update mobileSetting . offlineCache '@offline-cache.json'
cloudcc apply msapi . <planId>
cloudcc changes msapi . <changeId>
cloudcc get mobileSetting . offlineCache
```

## 3. 移动端参数 `parameters`

通用布尔字段均为可选补丁：

- `mobileAppNewTaskEnabled`
- `wechatContentRequired`
- `signInPhotoRequired`
- `scanValidationRulesEnabled`
- `recordCallDetails`
- `allowModifyCallDetails`
- `callLocationRequired`

`eventImage` 也是可选补丁，字段如下：

- `uploadMode`：`cameraAndAlbum`、`cameraOnly`、`albumOnly`
- `cameraWatermarkEnabled`、`uploadWatermarkEnabled`、`showFieldLabel`：布尔值
- `watermarkStyle`：`topOpaque` 或 `bottomTranslucent`
- `watermarkFields`：候选 ID 数组，最多 5 项，不允许重复

约束：`albumOnly` 时 `cameraWatermarkEnabled` 必须为 `false`；任一水印开启时，`watermarkFields` 至少一项。活动图片固定对应 `event.ffeEventPicture01`，无需也不允许通过 CLI 改成其他字段。

```json
{
  "signInPhotoRequired": true,
  "recordCallDetails": true,
  "eventImage": {
    "uploadMode": "cameraAndAlbum",
    "cameraWatermarkEnabled": true,
    "uploadWatermarkEnabled": false,
    "watermarkStyle": "topOpaque",
    "showFieldLabel": true,
    "watermarkFields": ["field-id-from-candidates"]
  }
}
```

先执行：

```powershell
cloudcc candidates mobileSetting . parameters
```

返回的 `objectFields` 和 `variables` 都可作为 `watermarkFields` 的 ID。更新由移动参数和活动图片字段两部分组成；MSAPI 执行器在后半段失败时会尝试恢复前态。

## 4. 移动端主页 `homepage`

`components` 是非空补丁数组，不必提交所有组件。每项只允许 `id` 和 `state`：

```json
{
  "components": [
    {"id": "component-id", "state": "enabled"},
    {"id": "another-id", "state": "disabled"}
  ]
}
```

状态值为 `enabled`、`disabled`、`closed`。ID 必须来自 `candidates`；重复或未知 ID 会失败。

## 5. 移动端离线缓存 `offlineCache`

三个字段都必须显式提供：

```json
{
  "enabled": true,
  "overrideUserSelections": false,
  "selectedTabs": ["tab-a", "tab-b"]
}
```

`selectedTabs` 最多 10 项；`enabled=true` 时至少一项。候选 ID 取自 `availableTabs`。

高风险提示：把 `overrideUserSelections` 从 `false` 改成 `true` 会清除用户级离线选择。计划会标记 `HIGH` 风险和 `mobile-offline-user-selections-reset` 非可逆副作用。回滚只能恢复组织配置，不能找回已删除的个人选择；应用前应确认业务窗口和影响范围。

## 6. 推荐模块 `recommendedModules`

`modules` 是非空补丁数组：

```json
{
  "modules": [
    {"id": "module-a", "enabled": true},
    {"id": "module-b", "enabled": false}
  ]
}
```

禁用模块前，CLI 会读取 `firstPage`。如果该模块仍是一级页面入口，计划以 `mobile_recommended_module_in_first_page` 冲突失败。正确顺序是先更新并应用 `firstPage`，再创建新的推荐模块计划；不要复用冲突前生成的旧计划。

## 7. 扫一扫模块 `scanModules`

结构与推荐模块相同：

```json
{
  "modules": [
    {"id": "scan-module-id", "enabled": true}
  ]
}
```

数组为补丁语义，未列出的模块保持不变。

## 8. 主页快捷功能 `homepageShortcuts`

必须同时给出 `hidden` 和 `selectedItems`：

```json
{
  "hidden": false,
  "selectedItems": ["shortcut-a", "shortcut-b", "shortcut-c", "shortcut-d"]
}
```

`selectedItems` 必须恰好 4 项，且都来自 `availableItems`。顺序会原样保留，代表页面显示顺序。

## 9. 一级页面 `firstPage`

```json
{
  "selectedItems": ["menu-a", "menu-b", "menu-c", "menu-d"]
}
```

必须恰好 4 个互不重复的候选 ID，数组顺序就是一级页面顺序。

保存该配置会清理当前菜单的运行时行，因此计划包含 `mobile-current-menu-runtime-reset` 非可逆副作用。回滚可恢复配置值，但不能恢复被清理的运行时记录；应在低峰期执行并验证移动端会话。

## 10. 计划审查与回滚

MSAPI 的 `update` 输出至少应检查：

- `operation` 和目标 `resource` 是否正确；
- `risk`、`warnings`、`nonReversibleEffects`；
- `beforeStateHash` 与 `desiredStateHash`；
- 变更是否只包含预期 ID，尤其是排序数组。

应用后使用 `changes` 查看审计记录，并再次 `get`。需要回滚时：

```powershell
cloudcc rollback msapi . <changeId>
```

若变更带非可逆副作用，回滚结果中的警告必须被视为剩余人工验证项，而不是“完全恢复”的证明。

## 11. 常见错误

| 错误码/前缀 | 含义与处理 |
|---|---|
| `mobile_settings_resource_invalid` | 资源名错误；使用本文七个资源名 |
| `mobile_settings_spec_invalid` | 字段未知、类型错误或缺少必填字段 |
| `mobile_settings_candidate_not_found` | ID 不在当前租户候选集；重新执行 `candidates` |
| `mobile_settings_selection_limit` | 超过上限或未满足“恰好 4 项” |
| `mobile_settings_duplicate_selection` | 数组包含重复 ID |
| `mobile_settings_no_changes` | 补丁与当前状态一致，无需生成计划 |
| `mobile_recommended_module_in_first_page` | 先从 `firstPage` 移除该模块并应用 |
| `mobile_settings_state_conflict` | 计划基于的前态已变化；重新读取并生成计划 |
| `mobile_settings_readback_mismatch` | Lightning 写后回读不一致；停止后续操作并人工核查 |
| `mobile_settings_horizontal_uiapi_unsupported` | Horizontal 不支持；改用 MSAPI 或 Lightning |

## 12. 安全与运维建议

- 不在命令、JSON 文件或日志中保存 access token；认证继续使用项目配置的安全凭据机制。
- 配置文件只保存 API ID，不保存页面文案；不同语言下文案会变化，ID 才是契约。
- 候选集和当前状态可能被其他管理员修改。长时间搁置的计划不要直接应用，应重新生成。
- 批量调整时按依赖顺序拆计划：先 `firstPage`，再禁用 `recommendedModules`；每个计划应用后回读。
- 生产执行前保留 `get` 输出和计划/变更 ID，作为审计证据；不要把租户数据打包进通用 skill 发布物。
