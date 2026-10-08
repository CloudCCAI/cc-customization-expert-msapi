# 横纵版自定义组件开发指南

## DSL 边界

- 文件必须以 `<cc:component>` 开始并以 `</cc:component>` 结束。
- `<cc>...</cc>` 用于 Java 语句，`<cc!>...</cc>` 用于声明，`<cc:outprint>...</cc:outprint>` 用于表达式输出；该标签不会自动提供可靠的上下文转义，动态值必须按 HTML/属性/JavaScript/URL 上下文显式转义。
- 需要服务端转发时可使用 `<cc:forward>` 与 `<cc:param>`，不要改写成 JSP namespace 标签。
- main-app 提供 `UserInfo userInfo`，可使用平台允许的 `CCService`、`CCObject`、`SendEmail`、`Util.*`、`Math.*` 能力。
- 禁止原始 JSP scriptlet、JSP namespace 标签和嵌套 component 引用；`cc` 代码块中也禁止注解以及 `com.g3cloud.*`、`com.cloudcc.*`、`org.apache.cayenne.*` 直接定义/导入。CLI 在网络请求前阻断这些结构。
- HTML 标签必须平衡。组件用于详情嵌入时，JavaScript 函数名和 DOM id 必须带组件/API 命名空间，避免同页多实例冲突。

## 生成前分析与提示词写法

生成组件前先明确：承载它的 VisualPage 或详情嵌入位置、输入数据、输出 UI、用户操作、对象字段、权限、空态/错误态、重复实例、依赖的静态资源和服务端类。提示词不得只写“生成一个组件”，至少应包含：

1. 明确目标为横纵版 `customComponent` DSL，不是 Lightning pagecomponent/Vue 组件。
2. 明确完整 wrapper、允许的三类 cc 标签，以及禁止的 JSP/nested-component 语法。
3. 指定数据读取条件、字段清单、最大行数、排序和分页；禁止全表查询与循环内查询。
4. 指定所有动态输出的转义策略，不把请求参数直接写入 HTML、JavaScript、SQL 或 URL。
5. 指定 CSS class、DOM id 和 JavaScript 函数的 API 名前缀，保证一个页面多实例可共存。
6. 复杂写入、事务、远程调用和复用逻辑下沉到自定义类；组件负责输入校验、调用和渲染。
7. 不生成凭据、binding、token、固定租户 URL 或直接数据库连接。

## 示例骨架

```text
<cc:component>
<div class="cc-order-card">
  TODO: 使用平台对象服务读取必要字段并输出转义后的展示值
</div>
</cc:component>
```

```text
cloudcc create customComponent OrderCard
cloudcc publish customComponent OrderCard .
```

写入成功状态为 `submitted`，表示 HTTP 2xx/3xx 已接收；main-app 的 HTML 页面不作为业务成功标志。CLI 会在同一会话中按名称和标签精确查找、调用详情确认，并把唯一 ID 原子写入同目录 `config.json.id`；无法确认唯一 ID 时仅报告 `idResolution=unresolved`，后续 publish 会再次尝试恢复。
