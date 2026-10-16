# 字段追踪开发指南

- 配置只写 `tp_sys_schemetable.is_tracked`；`tp_sys_fieldhistory` 是运行时历史数据，不属于元数据发布、扫描或回滚范围。
- 长文本、富文本和图片字段的 `trackingMode` 为 `change-only`；其他支持字段为 `old-and-new`。
- 候选过滤与 setup-service 对齐；lead/account 的系统追踪字段由服务端管理，replace/clear 不会关闭。
- MetadataService 不把版本默认值 20 当成租户权威额度。增加字段前，部署必须通过 `metadata-service.field-tracking.authoritative-limit` 提供已从 FIELD_TRACK / RES010 同步的权威额度；缺失时仅允许数量不增加的替换或清空。
- plan 和 apply 都校验对象、字段集合、系统字段及配额；配置漂移时 apply 失败，需重新生成 plan。
- Lightning UIAPI 由 setup-service 校验动态配额；horizontal UIAPI 在网络访问前失败，horizontal MSAPI 可用。
