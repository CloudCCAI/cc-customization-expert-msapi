# 横纵版 VisualPage

`visualPage` 是横纵版 main-app 的页面资源，由 `page.action` 管理并由平台 `PageTemplate` 转换、编译和生成运行文件。源码扩展名为 `.jsp`，但作者必须使用以自闭合 `<cc:page ... />` 开始的平台标签 DSL 与 HTML，不能直接写 JSP directive、scriptlet 或 JSP namespace 标签。它不是 Lightning `customPage` 画布，也不依赖 devconsole。

本资源仅在项目显式配置 `platformMode=horizontal` 时可用。CLI 使用已登录的 main-app 会话与 binding 调用 Struts Action；写入只按 HTTP 网络结果判断是否已提交，不把返回的 JSP 页面内容解释成权威业务成功。提交后可在同一会话中精确定位并详情确认 ID，用于本地配置回写；ID 解析失败不改变提交状态。

常用命令：

```text
cloudcc create visualPage <name>
cloudcc publish visualPage <name> [projectPath]
cloudcc get visualPage <projectPath>
cloudcc detail visualPage <projectPath> <id>
cloudcc assignProfiles visualPage <projectPath> <pageId> <profileIds>
cloudcc enableForProfile visualPage <projectPath> <profileId> <pageIds>
cloudcc delete visualPage <projectPath> <id>
```

本地文件位于 `visualpage/<name>/<name>.jsp`，同目录 `config.json` 保存标签、目录、页面类型和已有记录 ID。
