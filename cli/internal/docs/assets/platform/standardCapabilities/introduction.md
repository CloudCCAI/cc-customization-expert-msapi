# CloudCC 标准能力目录

## 作用

本目录是 CloudCC 实施设计的离线标准能力底座。即使项目暂时无法连接目标租户，也必须先基于 CloudCC 标准 CRM、商务云、CPQ、现场服务云、客户服务云、项目云、伙伴云、利润云等内置应用能力进行复用优先设计，再决定是否扩展标准对象或创建自定义对象。

目标租户可用性仍需用只读扫描确认：

```bash
cloudcc scan msapi <projectPath> standard-catalog
```

## 标准业务对象判定规则

`tp_sys_object.TABLE_TYPE = '2'` 是 CloudCC 标准业务数据对象的底层判定规则。`standard-catalog` 必须把这类对象全部纳入标准业务对象全集，并输出：

| 输出字段 | 含义 |
|---|---|
| `objects[].tableType` | 来自 `tp_sys_object.TABLE_TYPE` |
| `objects[].standardBusinessObject` | `tableType == "2"` 时为 `true` |
| `objects[].fields` | 该对象的字段级属性清单 |
| `objects[].referenceFields` | 引用字段及目标对象 |
| `objects[].inferredRole` | 基于命名和字段引用推断的业务角色 |
| `capabilityObjects[]` | 所有标准业务对象和预置核心能力命中项 |

因此，联系方式、业务机会小组、联系人角色、业务机会产品等对象只要在 `tp_sys_object` 中 `TABLE_TYPE = '2'`，就必须进入扫描结果；不允许只按预置清单、菜单入口或用户逐个指出的对象扫描。

## 设计原则

| 原则 | 说明 |
|---|---|
| 标准应用优先 | 先判断业务是否属于 CloudCC 已内置应用能力，避免重复造标准 CRM 或行业云对象 |
| 标准对象优先 | 客户、联系人、线索、商机、活动、订单、合同、产品、服务、项目、伙伴等先查标准对象 |
| 标准字段优先 | 标准对象上已有语义清晰字段时优先复用，不创建同义字段 |
| 扩展优先于新建 | 标准对象生命周期匹配但字段不足时，优先扩展字段、记录类型、布局和规则 |
| 新建必须有理由 | 只有标准应用、标准对象或标准关系无法表达业务语义、生命周期、权限边界、集成事实或报表口径时才新建自定义对象 |

## 标准应用能力域

| 应用能力域 | 核心承载能力 | 优先检查对象/能力 | 典型扩展点 |
|---|---|---|---|
| CRM 全流程 | 客户、联系人、联系人角色、线索、商机、业务机会产品、市场活动、销售活动、预测、订单、合同、产品、价格、服务入口 | Account、Contact、ContactRole/AccountContactRole/OpportunityContactRole、Lead、Opportunity、OpportunityProduct/OpportunityLineItem、Campaign、Task/Event、Product、Contract、Order、Case、客户小组/客户团队 | 客户分级、行业字段、业务线字段、联系人角色、商机产品字段、审批状态、集成编号 |
| 商务云 | 商品浏览、报价、报价产品、合同、订单、订单产品、履约、开票、回款、客户门户交易体验 | Product、Price Book、PricebookEntry、Quote、Quote Line/QuoteLineItem、Order、Order Product/OrderItem、Contract、Invoice、Payment、Portal/Partner 应用 | 外部订单号、履约状态、开票口径、客户侧确认 |
| CPQ | 产品配置、报价、报价产品、价格规则、价格表条目、折扣、审批、报价单生成 | Product、Product Option、Bundle、Price Rule、Discount、Quote、Quote Line/QuoteLineItem、PricebookEntry、Approval | 行业定价因子、成本字段、毛利字段、特殊审批 |
| 现场服务云 | 服务请求、工单、派工、工程师、现场执行、备件、服务报告、回访 | Case、Work Order、Service Appointment、Service Resource、Asset、Part、Service Report | 设备属性、现场签到、备件消耗、服务结算 |
| 客户服务云 | 客诉、服务工单、知识库、SLA、服务合同、升级、满意度 | Case、Knowledge、Entitlement、Service Contract、SLA、Survey | 客诉分类、质量责任、赔付、闭环整改 |
| 项目云 | 项目、里程碑、任务、资源、工时、交付物、项目成本和收入 | Project、Milestone、Project Task、Resource、Timesheet、Deliverable、Project Cost | 项目类型、验收节点、项目利润、跨部门协作 |
| 伙伴云 | 渠道伙伴、外部用户、伙伴门户、线索/商机分发、伙伴订单、伙伴权限 | Partner Account、Partner User、Portal Application、Partner Role、Sharing Rule、Lead/Opportunity | 伙伴等级、授权范围、门户菜单、外部用户审计 |
| 利润云 | 预算、成本、收入、费用、利润中心、毛利、盈利分析 | Budget、Cost、Revenue、Expense、Profit Center、Margin、Profit Analysis | 行业成本口径、返利/佣金、利润快照、报表维度 |

## 标准对象复用线索

| 业务概念 | 优先复用 | 不应直接新建的同义对象 |
|---|---|---|
| 客户主档 | Account / 客户 | 客户资料表、客户档案表 |
| 联系人 | Contact / 联系人 | 客户联系人表 |
| 联系人角色 | ContactRole / AccountContactRole / OpportunityContactRole | 客户联系人关系表、商机联系人关系表，除非标准角色关系无法表达客户专属角色口径 |
| 多人协同服务客户 | 客户小组 / 客户团队 | 客户销售视图、客户服务组，除非标准客户小组无法承载业务线和目标口径 |
| 多人协同服务商机 | 业务机会小组 / 商机团队 / OpportunityTeam | 商机销售视图、商机协作组，除非标准团队对象无法承载业务线和目标口径 |
| 线索 | Lead / 线索 | 潜客表、市场线索表 |
| 商机 | Opportunity / 商机 | 销售机会表、项目机会表 |
| 业务机会产品 | OpportunityProduct / OpportunityLineItem | 商机产品明细、项目产品清单，除非商机产品生命周期完全不同 |
| 市场活动 | Campaign / 市场活动 | 活动台账，除非活动明细生命周期明显不同 |
| 跟进、拜访、待办 | Task/Event / 活动任务 | 拜访记录表，除非需要独立复杂生命周期 |
| 产品和物料 | Product / 产品 | 产品主数据表，除非需要外部 GSKU 镜像或行业规格扩展 |
| 价格明细 | PricebookEntry / 价格表条目 | 产品价格表，除非价格主事实在外部系统且 CloudCC 只做镜像 |
| 报价 | Quote / 报价 | 价格申请表，除非报价审批不是标准 CPQ 生命周期 |
| 报价产品 | QuoteLine / QuoteLineItem | 报价明细表，除非非标准 CPQ 生命周期 |
| 订单 | Order / 订单 | 销售订单表，除非 CRM 只做外部订单镜像 |
| 订单产品 | OrderProduct / OrderItem / OrderLineItem | 订单明细表，除非订单主事实在 ERP/NCC |
| 合同 | Contract / 合同 | 合同台账 |
| 服务/客诉 | Case / 服务工单 | 客诉工单，除非客户服务云未启用或闭环语义不匹配 |
| 现场服务 | Work Order / Service Appointment | 派工单、现场服务单 |
| 项目 | Project / Project Task | 项目台账、项目任务表 |
| 外部用户和门户 | Partner User / Portal Application | 门户账号表，除非只做外部账号镜像审计 |
| 成本利润 | Budget / Cost / Profit Analysis | 利润快照表，除非标准利润云未启用或仅做外部报表镜像 |

## 使用限制

- 本目录是标准能力预置知识，不等于目标租户一定已经购买、启用或开放对应应用。
- 真实实施前必须扫描目标租户，确认应用、对象、字段、记录类型、布局、权限和许可。
- 标准对象 API 名、字段 API 名和字段属性以目标租户扫描结果为准。
- 如果预置目录与目标租户扫描结果冲突，以目标租户扫描结果和客户授权范围为准。

## 字段级扫描要求

`standard-catalog` 的验收粒度必须到字段级。扫描结果中每个标准对象都应保留 `fields` 清单，至少用于判断：

| 对象能力 | 必查关键字段语义 |
|---|---|
| 联系人角色 | 客户、联系人、业务机会、角色、是否主要联系人 |
| 联系方式 | 客户、联系人、线索、电话、手机、邮箱、是否默认、用途 |
| 业务机会产品 | 业务机会、产品、数量、销售价格、折扣、小计 |
| 业务机会小组 | 业务机会、成员、角色、访问权限 |
| 价格表条目 | 价格表、产品、价格、币种、启用状态 |
| 报价产品 | 报价、产品、数量、单价、折扣、小计 |
| 订单产品 | 订单、产品、数量、单价、金额、履约状态 |

如果对象存在但关键字段缺失或字段属性不匹配，应优先判断是否扩展标准对象字段、记录类型、布局、验证规则或审批过程，而不是直接新建同义对象。

## 动态发现机制

`standard-catalog` 不能依赖固定对象清单，也不能只读取应用菜单。正确扫描方式是：

1. 从对象元数据表读取租户中存在的所有对象，不论是否有菜单入口，并以 `TABLE_TYPE = '2'` 判定标准业务数据对象。
2. 从字段元数据表读取每个对象的所有字段，包含字段 API、标签、类型、必填、唯一、引用对象和全局选项。
3. 根据字段引用和对象命名动态识别标准能力角色：
   - 名称包含 Team/小组/团队，识别为 `team`。
   - 存在两个及以上引用字段，识别为 `relationship` 候选。
   - 名称或字段包含 Product/Line/Item/产品/明细，且存在产品引用、数量、价格、金额或折扣字段，识别为 `line_item`。
   - 名称包含 Role/角色，识别为角色关系对象候选。
4. 把 `TABLE_TYPE = '2'` 的标准业务对象全部写入 `capabilityObjects`；动态识别出的对象状态为 `discovered`，预置核心对象清单只作为提示，不作为扫描上限。
