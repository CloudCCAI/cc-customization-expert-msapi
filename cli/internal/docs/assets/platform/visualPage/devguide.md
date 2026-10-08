# 横纵版 VisualPage 开发指南

## 生成前必须先做的分析

生成或修改页面前，先明确页面入口、业务对象、需要展示和提交的字段、权限边界、移动端要求、引用的静态资源与自定义组件、错误处理及数据量。标准对象页、布局、按钮、验证规则或流程能够完成时，仍优先使用低代码元数据；VisualPage 只承担标准页面无法覆盖的交互。

## 源码模型

- 页面源文件是 main-app `PageTemplate` 的输入，不是 Lightning 画布 JSON。
- 源文件第一条声明必须是自闭合的 `<cc:page ... />`。扩展名虽然是 `.jsp`，但作者编写的是平台 `cc` 标签 DSL 与 HTML；不得写 `<%@ ... %>`、`<% ... %>` 或 `<jsp:...>` 原生 JSP 语法，CLI 会在网络请求前阻断。
- 页面保存时固定提交 `versiontype=oldsystem`；`functiontype`、`folderid`、`appliedmobile` 来自本地 `config.json`。
- main-app 提供登录用户上下文。访问 CloudCC 数据时优先使用 `CCService`、`CCObject` 等平台能力，不直接拼接数据库连接或租户表 SQL。
- 页面可通过平台支持的 component 标签引用横纵版 `customComponent`，但被引用组件应独立创建、发布和验证。
- 静态资源使用 main-app 的 staticResource 运行地址，不把本机路径、密码、binding、token 或租户专属地址写入页面。

## 生成提示词约束

要求智能体生成 VisualPage 时，提示词必须同时写清：

1. 目标是横纵版 VisualPage 的 `cc` 标签 DSL，文件以自闭合 `<cc:page ... />` 开始；禁止套用原生 JSP、Lightning customPage、Vue pagecomponent 或 devconsole 模型。
2. 给出页面用途、对象/字段、读写行为、权限、空态、错误态、分页/数据量和移动端要求。
3. 只选取必要字段，禁止 `SELECT *`、无界循环、全表加载、逐行查询和直接物理删除。
4. 复杂业务、批量处理和可复用逻辑下沉到横纵版自定义类；页面只做参数校验、编排和展示。
5. 输出平衡的 HTML 与平台 `cc` 标签结构，并说明使用的 staticResource 和 customComponent API 名；不得输出 JSP directive、scriptlet 或 JSP namespace 标签。
6. 不生成或输出凭据，不修改用户、简档、角色、许可、启停状态等安全字段。

## 发布语义

```text
cloudcc create visualPage OrderWorkbench
cloudcc publish visualPage OrderWorkbench .
cloudcc assignProfiles visualPage . <pageId> <profileId1,profileId2>
cloudcc enableForProfile visualPage . <profileId> <pageId1,pageId2>
```

CLI 仍以 HTTP 2xx/3xx 输出 `submitted`。随后在同一会话中按页面名称和标签精确查找、调用详情确认，并把唯一 ID 原子写入 `visualpage/<name>/config.json`；解析失败只输出 `idResolution=unresolved`，不会把已经提交的请求改判失败。网络错误、超时或 HTTP 4xx/5xx 才判为失败。
