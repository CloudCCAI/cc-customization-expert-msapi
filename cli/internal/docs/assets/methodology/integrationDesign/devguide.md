# CloudCC 接口设计卡片

## 标准模板

```text
接口名称：

业务场景：
- ...

对接系统：
- ...

数据类型：
- 主数据 / 业务单据 / 状态回传 / 文件 / 附件 / 日志

数据流向：
- CloudCC -> 外部系统
- 外部系统 -> CloudCC
- 双向

触发方式：
- 用户按钮
- 触发器标记
- 定时作业
- sidecar 定时拉取
- 外部回调

调用方式：
- HTTP POST / GET / 文件上传 / Webhook / 待确认

认证方式：
- Token / AK-SK / OAuth / IP 白名单 / VPN / 待确认

关键入参：
- ...

关键出参：
- ...

字段映射：
- CloudCC 字段 -> 外部字段 -> 转换规则

幂等键：
- ...

同步状态：
- Pending / Processing / Success / Failed / Canceled

异常处理：
- 超时:
- 重试:
- 人工重放:
- 告警:

日志与监控：
- ...

验收口径：
- ...

待确认：
- ...
```

## 触发方式选择

| 场景 | 推荐方式 |
|---|---|
| 用户提交外部审批 | 按钮 + 类 + sidecar 或外部 API |
| 审批通过后生成内部数据 | 触发器或审批后动作 |
| 外部系统回传状态 | sidecar 接收回调后回写 CloudCC |
| 大批量主数据同步 | sidecar 定时或批处理 |
| 轻量周期检查 | CloudCC 定时类或 sidecar，按耗时和数据量决定 |
| 小程序提交服务申请 | sidecar/API 接入，CloudCC 创建服务请求 |
| 车联网故障轮询 | sidecar 定时拉取，按规则生成工单 |

## 状态字段建议

```text
sync_status
sync_request_id
external_id
external_status
last_sync_time
retry_count
next_retry_time
sync_error_message
sync_payload_hash
```

字段名称仅为建议，实际 API 名称应按项目命名规范设计。

## 接口评审清单

- 是否明确数据流向？
- 是否明确触发时机？
- 是否有幂等键？
- 是否有同步状态和错误字段？
- 是否定义超时、重试、补偿？
- 是否需要回调入口？
- 是否需要附件或文件归档？
- 是否涉及敏感凭据？
- 是否需要 sidecar 隔离网络和密钥？
- 是否有验收样例？
