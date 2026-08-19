# CloudCC PaaS 平台方案设计指南

## 使用目标

当用户提出一个业务需求时，Agent 不应直接跳到某个 CLI 命令，而应先识别需求属于哪一层能力：

1. 数据建模
2. 权限治理
3. 页面和应用入口
4. 自动化流程
5. 前端体验增强
6. 服务端高代码扩展
7. 集成和数据迁移
8. 移动端适配
9. 多环境交付

## 标准设计顺序

### 1. 识别业务对象

先判断需求是否已有标准对象可承载；标准对象不够时再创建自定义对象。

相关文档：

```bash
cloudcc doc platform/object introduction
cloudcc doc platform/fields introduction
cloudcc doc platform/dataModeling devguide
```

### 2. 设计字段和关系

确认字段类型、选项集、查找关系、主详关系、公式、累计汇总、历史跟踪和字段依赖。

优先避免：

- 把同一含义拆成多个重复字段
- 过早使用长文本保存结构化数据
- 用代码弥补本可由字段类型或关系表达的模型

### 3. 设计权限

确认用户是谁、能进入哪个应用、能看到哪些菜单、能操作哪些对象、能看到哪些记录、能看到哪些字段。

相关文档：

```bash
cloudcc doc platform/security introduction
cloudcc doc platform/profile introduction
cloudcc doc platform/permission introduction
cloudcc doc platform/sharingRule introduction
```

### 4. 设计页面和入口

先用应用、菜单、视图、页面布局满足标准业务；复杂体验再使用自定义页面和 pagecomponent。

相关文档：

```bash
cloudcc doc platform/application introduction
cloudcc doc platform/menu introduction
cloudcc doc platform/view introduction
cloudcc doc platform/pagelayout introduction
cloudcc doc platform/customPage introduction
cloudcc doc platform/pagecomponent introduction
```

### 5. 设计自动化

根据触发时机选择能力：

- 保存前校验：验证规则
- 重复数据控制：查重过滤器
- 常规通知和字段更新：工作流/动作
- 多级人工决策：批准过程
- 记录事件中的复杂逻辑：触发器
- 可复用服务端能力：类
- 周期性后台处理：定时类 + 定时作业

相关文档：

```bash
cloudcc doc platform/automation introduction
cloudcc doc platform/validationRule introduction
cloudcc doc platform/dupeCatcher introduction
cloudcc doc platform/triggers introduction
cloudcc doc platform/classes introduction
cloudcc doc platform/timer introduction
cloudcc doc platform/scheduleJob introduction
```

### 6. 设计集成

优先确认是预置集成、OpenAPI 集成、服务端类集成、定时同步、前端组件集成，还是 sidecar 中间程序。

相关文档：

```bash
cloudcc doc platform/integrationArchitecture introduction
cloudcc doc platform/openapi introduction
cloudcc doc platform/identityProvider introduction
cloudcc doc platform/singleSignOn introduction
```

### 7. 设计移动端和交付

如需求涉及外勤、移动审批、扫码、签到、离线数据，应先查看移动端能力文档。

如需求涉及多环境迁移，应先查看 ALM 文档。

```bash
cloudcc doc platform/mobileCapabilities introduction
cloudcc doc platform/almRelease introduction
```

## Agent 输出要求

给出方案时，应包含：

- 使用了哪些 CloudCC 能力
- 为什么优先选择这些能力
- 哪些能力属于低代码配置
- 哪些能力属于高代码扩展
- 哪些事项需要用户确认
- 哪些能力当前文档无法确认

## 不确定事项处理

如果手册、当前技能文档和源码结构都没有明确说明，必须标注为“待确认”，不要编造接口或后台能力。
