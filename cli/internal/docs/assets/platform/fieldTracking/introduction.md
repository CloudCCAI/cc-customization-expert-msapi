# 字段追踪

`fieldTracking` 管理对象字段的历史追踪开关，与字段依赖无关。

```bash
cloudcc get fieldTracking <projectPath> <object>
cloudcc candidates fieldTracking <projectPath> <object>
cloudcc update fieldTracking <projectPath> @tracking.json
cloudcc clear fieldTracking <projectPath> <object>
```

MSAPI 写操作先生成 plan，再通过 `cloudcc apply msapi <projectPath> <planId>` 应用；可使用标准 changes 和 rollback 命令审计或回滚。Lightning UIAPI 直接复用 setup-service，因此不提供 MetadataService rollback。

输入示例：

```json
{"object":"account","trackedFields":["name","industry","description"]}
```

`trackedFields` 是完整替换集合，必须显式提供；空数组等价于 clear。字段选择器可使用字段 ID 或 API 名。系统管理字段由服务端保留，不要放入输入。
