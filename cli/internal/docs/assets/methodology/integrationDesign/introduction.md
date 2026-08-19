# CloudCC 集成设计方法

## 定位

本模块用于把蓝图中的系统对接需求转换成可实施的 CloudCC 集成设计。它和 `platform/integrationPatterns` 的关系是：

- `platform/integrationPatterns`：判断按钮、类、触发器、定时类、sidecar 怎么选。
- `methodology/integrationDesign`：输出接口清单、字段映射、状态、异常和验收口径。

## 集成设计要覆盖

- 对接系统。
- 主数据还是业务单据。
- 数据流向。
- 触发时机。
- 实时、定时、按钮、回调。
- 认证方式。
- 幂等键。
- 状态字段。
- 失败重试。
- 监控和人工重放。

## 常见对接系统类型

- ERP：客户、物料、订单、交货、开票、库存。
- BPM/OA：合同评审、交货审批、调拨审批、变更审批。
- MDM：客户、物料、银行视图、组织主数据。
- TMS：发运计划、物流跟踪、签收确认。
- 呼叫中心：来电弹屏、问题记录、服务请求。
- 信用平台：客户信用评估。
- 资金系统：银行流水、票据流水、回款认领、凭证状态。
- 车联网：位置、工况、故障码、工作小时。
- QMS/WMS：质量、索赔、配件出入库。
- 小程序/门户：登录、认证、设备绑定、服务申请、评价。

## 相关文档

```bash
cloudcc doc platform/openapi devguide
cloudcc doc platform/integrationArchitecture devguide
cloudcc doc platform/integrationPatterns devguide
```
