# CloudCC 客户端脚本操作指南

客户端脚本属于前端高代码。唯一源码位置为 `frontend/scripts/<objectApi>/<name>/`，每个资源包含 `<name>.js` 和 `config.json`。完整项目目录标准见 `cloudcc doc projectLayout devguide`。

## 1. 创建本地资源

```bash
cloudcc create script Account/BeforeSave /path/to/project
```

`Account/BeforeSave` 是 `<objectApi>/<name>` 资源标识；projectPath 可省略，默认当前目录。对象 API 名称须以目标环境元数据为准。

生成：

```text
frontend/scripts/Account/BeforeSave/
├── BeforeSave.js
└── config.json
```

源码模板提供 `function main($CCDK, obj) { ... }`，配置初始包含 `name`。根据目标环境要求补齐发布配置；不要将租户凭据写入源码配置。

create 不访问租户，不覆盖已有资源目录。标识不得包含绝对路径、路径穿越或跨出项目的符号链接。采用 `docs/project-layout.json` 的项目须登记 frontend 为共享源码目录。

## 2. 发布

```bash
cloudcc publish script Account/BeforeSave /path/to/project
```

从规范位置读取 `BeforeSave.js` 与 `config.json`，提取 main 函数体作为 `scriptContent`，随配置提交至 `/devconsole/script/saveClientScript`，输出服务端响应。源码与连接配置均取自同一 projectPath。

缺失源码或配置时失败；发送请求前检查路径和 main 函数。当前命令不自动把响应 id 写回本地配置；需要更新已有资源时，应先核实服务端资源身份及配置字段。提交响应不等同于业务行为验收，发布后应在目标页面验证事件绑定和实际效果。

## 3. 远程查询

```bash
cloudcc get script /path/to/project '{"pageSize":50,"pageNo":1,"condition":{"scriptName":"","pageLabel":"","objName":""}}'
```

查询采用 `<projectPath> [encodedBodyJson]` 参数顺序，请求体支持 JSON 或 URI 编码 JSON。`condition.objName` 按查询接口要求传目标环境中的对象名称，不通过中文语义推测对象 API 名。

当前 `get/detail/pull/pullList/delete script` 共用 `/devconsole/script/pageClientScript` 查询实现。这些命令不会恢复本地源码，也不实现删除操作；不得将命令名当作已实现能力。需删除或拉取源码时，应采用已核实支持该操作的平台能力。

## 4. 本地检查

```bash
cloudcc scan msapi /path/to/project highcode
cloudcc doctor project-layout /path/to/project
```

高代码扫描按 `frontend/scripts/` 统计源码、配置和结构问题；目录检查覆盖根路径登记及共享/本地归属。两者均不能替代目标环境的事件、权限和业务行为验证。

横纵版当前不支持 script 操作；本指南适用于已实现客户端脚本接口的 Lightning 路由。
