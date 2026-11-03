# 搜索布局

`searchLayout` 管理对象设置中的搜索布局，不是页面布局 `pagelayout`，也不是查找字段筛选器 `lookupFilter`。

## 能力范围

本 Domain 覆盖六个可独立维护的 section：

- `searchResult`：搜索结果字段、普通/精确搜索字段和自定义按钮。
- `lookupDialog`：查找条件字段和查找结果字段。
- `menu`：PC、移动端、移动屏幕字段、嵌入页面和默认移动菜单。
- `listView`：标准按钮、自定义按钮和最近使用按钮设置。
- `quickCreate`：快速新建启用状态和字段。
- `duplicateMerge`：查重合并启用状态和字段；只有部分对象支持。

## 对象能力发现

对象是否支持某个 section、候选字段和限制由目标环境返回，CLI 不使用固定对象白名单。配置前先运行：

```text
cloudcc capabilities searchLayout <projectPath> <object>
cloudcc candidates searchLayout <projectPath> <object> <section>
cloudcc get searchLayout <projectPath> <object>
cloudcc doc searchLayout devguide
```

写入使用 section 级补丁：省略 section 或属性会保留原值，显式数组会替换对应有序集合。MSAPI 写命令只生成计划，必须审阅后显式 apply，并在完成后回读。
