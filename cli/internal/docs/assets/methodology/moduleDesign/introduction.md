# CloudCC 模块设计方法

## 定位

本模块用于把业务模块拆成 CloudCC 可落地的设计卡片。它适用于渠道、市场、商机、合同、订单、开票回款、服务工单、资产、客户门户等模块。

模块设计的核心不是“列功能”，而是把业务目标映射到 CloudCC 对象、字段、页面、权限、自动化、高代码扩展和集成边界。

## 模块设计层次

```text
业务目标
  -> 流程节点
  -> 角色职责
  -> 对象和记录类型
  -> 字段和状态机
  -> 页面和相关列表
  -> 规则、审批、触发器、类、定时作业
  -> pagecomponent / sidecar
  -> 接口和报表
```

## 常见模块类型

- 主数据模块：客户、联系人、产品、资产、服务商、服务网点。
- 过程模块：线索、商机、报价、合同、订单、服务请求、服务工单。
- 交易模块：交货、开票、收款、索赔、结算。
- 协同模块：审批申请、调拨申请、技术支持申请、变更申请。
- 门户模块：小程序、客户自助服务、在线客服、设备绑定。

## 相关文档

```bash
cloudcc doc platform/object devguide
cloudcc doc platform/fields devguide
cloudcc doc platform/recordType devguide
cloudcc doc platform/pagelayout devguide
cloudcc doc platform/permission devguide
cloudcc doc platform/triggers devguide
cloudcc doc platform/classes devguide
cloudcc doc platform/pagecomponent devguide
```
