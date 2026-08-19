# CloudCC 多系统集成模式

## 定位

本模块用于判断多系统数据集成时，什么时候使用 CloudCC 内部能力（按钮、类、触发器、定时类），什么时候应使用外挂 `sidecar` 服务。

事实依据：

- 当前技能确认 CloudCC 支持 OpenAPI、类、触发器、定时类、定时作业、pagecomponent、客户端脚本和 `sidecar/` 目录约定。
- 《CloudCC平台操作手册》确认开发者空间包含类、定时类、定时作业、触发器、自定义页面、自定义组件、静态资源、客户端脚本。
- 手册自动化章节确认动作可关联任务、电子邮件通知、字段更新、触发器、Webhook。

方法论说明：

- 下文关于异步队列、重试、幂等、死信、监控、补偿和 sidecar 运维的内容属于通用集成架构建议，具体技术栈和运行机制需按项目实现。

## 一句话原则

在 CloudCC 事务内、低频、短耗时、强依赖当前用户操作结果的集成，可以用按钮、类、触发器或定时类。

跨系统链路长、耗时不可控、需要重试补偿、异步解耦、批处理、回调接入或独立运维的集成，应放到 `sidecar`。

## 能力边界

### 按钮调用

适合用户主动触发、需要即时反馈的外部查询或同步。

典型场景：

- 查工商信息
- 查库存
- 查物流状态
- 手动推送当前记录到外部系统

### 类

适合封装 CloudCC 服务端可复用业务逻辑。

典型场景：

- 给按钮或 pagecomponent 提供后端能力
- 封装外部 API 调用
- 被触发器复用
- 承载较复杂但仍适合在 CloudCC 内运行的业务规则

### 触发器

适合捕获记录保存事件，并执行强绑定的业务逻辑。

典型场景：

- 保存前校验
- 保存后生成关联记录
- 状态变更后写入待同步标记

注意：触发器不应承担复杂集成中台职责。外部 API 慢、不稳定或需要重试补偿时，推荐触发器只写 `Pending` 状态，由 sidecar 异步处理。

### 定时类和定时作业

适合 CloudCC 内可承载的周期性任务。

典型场景：

- 周期性扫描待处理记录
- 轻量批量同步
- 数据修正和回填
- 定期提醒和催办

### Sidecar

`sidecar` 是外挂中间程序目录约定，适合承载不适合放在 CloudCC 事务或平台运行时中的集成逻辑。

典型场景：

- 多系统编排
- 大批量同步
- 接收外部系统 Webhook
- 异步队列和失败重试
- 数据清洗、字段映射、幂等控制
- 独立日志、监控、告警和部署
- 隔离外部系统密钥、证书、VPN、内网访问

## 五种推荐模式

### 1. 按钮即时查询模式

```text
用户点击按钮
  -> CloudCC 类调用外部 API
  -> 返回结果给页面
```

适合短耗时、低风险、需要用户立即看到结果的查询。

### 2. 类封装服务模式

```text
pagecomponent / customPage / button / trigger
  -> 调用 CloudCC 类
  -> 类封装业务逻辑或外部调用
```

适合复用服务端逻辑，避免在多个前端组件或触发器中重复实现。

### 3. 触发器标记待同步模式

```text
CloudCC 触发器
  -> 校验必要字段
  -> 写入 sync_status = Pending
  -> 写入 sync_request_id / business_key
  -> 保存成功

sidecar
  -> 拉取 Pending 记录
  -> 调用外部系统
  -> 成功回写 Success + external_id
  -> 失败回写 Failed + error_message + retry_count
```

这是复杂集成的推荐默认模式。

### 4. 定时批处理模式

```text
CloudCC 定时作业
  -> 执行定时类
  -> 扫描满足条件的数据
  -> 批量处理或回写
```

适合平台内可承载、数据量和耗时可控的周期性任务。

### 5. Sidecar 集成中台模式

```text
CloudCC / 外部系统 / Webhook
  -> sidecar
  -> 队列、映射、幂等、重试、日志
  -> CloudCC OpenAPI 或外部 API
```

适合多系统、长链路、高可靠要求和独立运维的集成。

## 相关文档

```bash
cloudcc doc platform/integrationArchitecture introduction
cloudcc doc platform/openapi introduction
cloudcc doc platform/classes introduction
cloudcc doc platform/triggers introduction
cloudcc doc platform/timer introduction
cloudcc doc platform/scheduleJob introduction
cloudcc doc platform/pagecomponent introduction
```
