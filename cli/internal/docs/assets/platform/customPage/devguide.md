# CloudCC 自定义页面开发指南

## 1. 模块定位

`customPage` 属于 high-code 页面资源，不纳入 MetadataService/MSAPI 写域。Go CLI 只做 CloudCC devconsole 原接口封装，底层仍调用 `/devconsole/custom/pc/1.0/post/*`。

低代码菜单、应用、简档可见性等元数据仍应走 MetadataService plan/apply；自定义页面本身和 pagecomponent 绑定走本模块。

## 2. 命令总览

```bash
cloudcc get customPage <projectPath> [pageApi]
cloudcc detail customPage <projectPath> <pageApi|id>
cloudcc create customPage <projectPath> <payload|@file>
cloudcc update customPage <projectPath> <pageApi|id> <payload|@file>
cloudcc delete customPage <projectPath> <pageApi|id>
cloudcc bind pagecomponent <projectPath> <pageApi> <componentIdOrName> [--embedded true|false] [--workspace-url <url>] [--dry-run]
cloudcc verify injectionPage <projectPath> <pageApi> [--expected-component <id|name>] [--expected-component-id <id>] [--expected-component-version <version>] [--stale-policy warning|failure] [--snapshot <@json|json>]
```

## 3. Devconsole Envelope

customPage 读取和保存必须使用 devconsole envelope：

```json
{
  "head": {
    "appType": "lightning-devconsole",
    "accessToken": "<runtime-token>",
    "source": "lightning-devconsole",
    "version": "public"
  },
  "body": {}
}
```

customPage devconsole 链路优先使用 `accessToken`，仅在缺失时回退 `pluginToken`。`detailCustomPage` 和 `deleteCustomPage` 按 identifier 类型二选一传参：24 位 ObjectId 传 `{ "id": "..." }`，其他页面 API 名传 `{ "pageApi": "..." }`，不要同时传 `id` 和 `pageApi`。CloudCC 响应必须优先判断 `returnCode`，即使 `result=true`，`returnCode=500` 也必须视为失败。

CLI 输入允许把 `pageContent` 传为对象数组、把 `canvasStyleData` 传为对象；真正发送给 devconsole 时必须遵循服务端实体类型：

- `pageContent`：JSON 字符串。
- `canvasStyleData`：JSON 字符串。
- `compList`：对象数组，每项包含 `id` 和 `compUniName`。
- `update`：先读取当前页面并携带现有 `id`，否则服务端会把同一 `pageApi` 当作重复创建。

CLI 会在 wire 层自动完成字符串化；调用者不要对已经是 JSON 字符串的字段重复编码。

不要使用 CRM Web shell 的 `lightning-main` 上下文保存 customPage。token/source 不匹配时应 fail closed。

## 4. 查询与回读

```bash
cloudcc get customPage . customer_interaction_workbench
cloudcc detail customPage . customer_interaction_workbench
```

`detail` 会输出 `id`、`pageApi`、`renderVersion`、`canvasStyleData` 和 `componentRefs`。排查白页时先确认 `componentRefs[].comId` 是否为预期 pagecomponent id。

## 5. 更新页面

```bash
cloudcc update customPage . customer_interaction_workbench @custom-page.json
```

payload 至少应包含：

```json
{
  "pageLabel": "客户互动工作台",
  "pageApi": "customer_interaction_workbench",
  "pageContent": [
    {
      "name": "component-customer-workbench",
      "comId": "6a4db950e4b0a577cbba1eca",
      "embedded": true,
      "propObj": {
        "workspaceUrl": "https://x.agentcici.com/app?aiApp=customer-workbench"
      }
    }
  ],
  "compList": [
    {
      "id": "6a4db950e4b0a577cbba1eca",
      "compUniName": "component-customer-workbench"
    }
  ]
}
```

`compList` 必须使用 `{ id, compUniName }`。只传 `{ id, compName }` 会在本地校验阶段被拒绝。

保存前 CLI 会本地预检 lightning-devconsole 保存 payload：`pageContent` 和 `compList` 必须能解析为对象数组，`pageContent[].comId` 必须能在 `compList[].id` 中找到；若存在 `pageContent[].componentInfo.id`，它必须与同条 `comId` 一致。预检失败不会调用 `insertCustomPage`。

当 CloudCC devconsole 返回非成功 `returnCode`/`code` 时，CLI 错误会保留 `returnInfo/msg`，并附加 `responseBody=<json>`，用于排查底层保存协议或网关异常，而不是只暴露“系统发生异常”。

## 6. 绑定 PageComponent

pagecomponent 发布新版本或新 id 后，已有 customPage 不会自动跟随。使用绑定命令更新页面引用：

```bash
cloudcc bind pagecomponent . customer_interaction_workbench component-customer-workbench \
  --embedded true \
  --workspace-url https://x.agentcici.com/app?aiApp=customer-workbench \
  --dry-run
```

绑定流程：

1. `detailCustomPage` 回读当前页面。
2. `detailCustomComp` 或组件列表解析目标 pagecomponent。
3. 更新 `pageContent[].comId`、`embedded`、`propObj.workspaceUrl`、`componentInfo.id/component/loadModel`。
4. 保存 `compList: [{ id, compUniName }]`。
5. `--dry-run` 时只输出待保存 payload，不生成新的 customPage renderVersion。
6. 去掉 `--dry-run` 后调用 `insertCustomPage` 保存，再次 `detailCustomPage` 回读并输出前后摘要。

## 7. 注入页诊断

```bash
cloudcc verify injectionPage . customer_interaction_workbench \
  --expected-component component-customer-workbench \
  --expected-component-id 6a4f2c24e4b0a577cbba1f4c \
  --expected-component-version V7.0 \
  --snapshot @runtime-snapshot.json
```

snapshot 可由浏览器验收脚本提供，例如：

```json
{
  "hasElement": true,
  "hasIframe": true,
  "iframeSrc": "https://x.agentcici.com/app?aiApp=customer-workbench"
}
```

常见诊断状态：

- `passed`
- `stale_component_reference`
- `warning`（当 `--stale-policy warning` 且发现 stale component id/version 时）
- `custom_page_missing`
- `component_not_mounted`
- `iframe_missing`

`--expected-component` 传组件名时，CLI 会尝试解析当前线上 pagecomponent 的最新 id；若 customPage 仍引用旧 `comId`，即使组件名匹配也会输出 `stale_component_reference`。已知最新组件 id 时优先显式传 `--expected-component-id`；只想提示但不阻断验收时可传 `--stale-policy warning`。

仅 customPage 回读成功不代表运行态成功；CRM 注入页还需要确认 CDN 脚本、DOM mount、iframe 或主内容可见。
