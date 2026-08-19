# CloudCC 能力地图

## 作用

能力地图用于把用户的业务问题路由到正确的 CloudCC 平台能力，避免直接跳到单点命令。

## 常见需求到模块映射

### 盘点标准应用和标准对象承载能力

推荐模块：

```text
msapi scanner -> application/menu -> object/fields -> recordType/pagelayout -> profile/sharingRule
```

推荐命令：

```bash
cloudcc scan msapi <projectPath> standard-catalog
```

说明：

- 该扫描只读输出目标租户已安装或已启用的标准应用、菜单、对象、字段，以及标准/自定义 origin。
- CloudCC 平台本身可能内置 CRM 全流程标准功能，以及商务云、CPQ、现场服务云、客户服务云、项目云、伙伴云、利润云等标准应用。实施方案必须先扫描目标租户实际能力，再决定复用、扩展或新建。
- 对象字段建模时，标准对象优先；标准对象字段不足时优先扩展；只有标准能力无法表达业务语义或生命周期时才创建自定义对象。

### 创建一个业务应用

推荐模块：

```text
object -> fields -> pagelayout -> menu -> application -> profile/permission
```

适用场景：业务台账、项目管理、设备巡检、售后工单、渠道管理等。

### 做数据质量控制

推荐模块：

```text
fields -> validationRule -> dupeCatcher -> triggers
```

说明：

- 字段类型先约束数据结构。
- 验证规则阻止错误保存。
- 查重过滤器控制重复记录。
- 触发器处理配置规则无法覆盖的复杂逻辑。

### 做审批或流程

推荐模块：

```text
object/fields -> workflow/approval -> action -> notification -> profile/role
```

说明：审批过程依赖对象、字段、用户层级、审批人规则和通知动作。

### 做复杂页面

推荐模块：

```text
pagelayout -> script -> customPage -> pagecomponent -> staticResource -> classes/openapi
```

说明：

- 标准页面优先用页面布局。
- 局部前端行为用客户端脚本。
- 页面编排用自定义页面。
- 复杂 UI 用 pagecomponent。
- 复用文件用静态资源。
- 后端数据或复杂逻辑用类或 OpenAPI。

### 做外部系统集成

推荐模块：

```text
openapi -> classes -> timer/scheduleJob -> identityProvider/singleSignOn -> sidecar
```

说明：

- 数据 CRUD 优先看 OpenAPI。
- 服务端业务封装用类。
- 定时同步用定时类和定时作业。
- 身份统一用 IdP/SSO。
- 平台内不适合承载的中间程序放 sidecar。

### 做移动端业务

推荐模块：

```text
pagelayout -> mobileCapabilities -> validationRule -> profile -> menu/application
```

说明：移动端能力与布局、菜单、权限、验证规则联动。

### 做多环境交付

推荐模块：

```text
project/config -> almRelease -> package -> classes/triggers/pagecomponent/staticResource
```

说明：先区分配置迁移、代码发布、软件包迁移和手工后台设置。

## 能力分层速查

| 层级 | 能力 | 代表模块 |
|---|---|---|
| 数据模型 | 对象、字段、关系、记录类型、选项集 | object, fields, recordType, globalSelectList |
| 数据质量 | 验证、查重、字段依赖、历史跟踪 | validationRule, dupeCatcher |
| 权限治理 | 角色、简档、权限集、共享规则、SSO | role, profile, permission, sharingRule, identityProvider, singleSignOn |
| 页面入口 | 应用、菜单、视图、布局 | application, menu, view, pagelayout |
| 自动化 | 工作流、审批、动作、触发器、定时 | automation, triggers, timer, scheduleJob |
| 前端扩展 | 页面、组件、脚本、静态资源 | customPage, pagecomponent, script, html, staticResource |
| 集成 | OpenAPI、服务端类、SSO、sidecar | openapi, classes, integrationArchitecture |
| 移动端 | 移动布局、离线、扫一扫、签到 | mobileCapabilities |
| 交付 | 项目配置、软件包、多环境发布 | project, config, almRelease |
| 标准应用目录 | CRM、商务云、CPQ、现场服务云、客户服务云、项目云、伙伴云、利润云 | application, menu, object, fields, standard-catalog |

## 使用建议

Agent 在回答方案前，应先说明命中的能力路径。例如：

```text
本需求命中：object -> fields -> pagelayout -> validationRule -> profile。
原因：它是一个以新业务对象为核心的台账应用，暂不需要高代码。
```
