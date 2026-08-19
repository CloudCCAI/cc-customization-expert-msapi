# CloudCC 共享规则介绍

共享规则用于在组织默认共享和角色层级之外扩展记录级访问范围。它通常与 role、profile、permission 一起构成 CloudCC/Salesforce 风格的权限模型：

- role 控制上下级组织架构和记录级数据可见性。
- profile 控制应用、菜单、对象、字段、记录类型、布局和登录策略等功能权限。
- permission / permission set 做跨岗位或临时补充授权。
- sharingRule 做按所有人、角色、公用小组、队列或字段条件的例外共享扩展。

## 能力边界

- 原 CloudCC API 查询：`POST /api/sharingSettings/queryRule`，请求体 `{ "objid": "<对象id>" }`。
- MetadataService domain：`sharing-rules`，用于计划、执行、变更观测和回滚共享规则元数据。
- 共享规则会改变记录可见范围，上线前必须使用目标 role/profile 用户账号做正向和负向验收。

## 相关命令

```bash
cloudcc get sharingRule <objid> [projectPath]
cloudcc plan msapi sharing-rules @sharing-rule.json
cloudcc apply msapi <planId>
cloudcc changes msapi <operationId>
cloudcc rollback-plan msapi <operationId>
```
