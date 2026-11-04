# CloudCC 角色 Domain

`role` 是独立的 CLI 一级 Domain，MetadataService 规范名为 `roles`。它负责角色列表、详情、父角色候选、
创建、重命名、子树移动、用户分配和受保护删除。

```bash
cloudcc get role .
cloudcc detail role . <角色ID或唯一名称>
cloudcc capabilities role .
cloudcc candidates role . createParent
```

所有写命令只生成 MetadataService 计划；检查计划后再显式执行：

```bash
cloudcc apply msapi . <planId>
cloudcc changes msapi . <operationId>
cloudcc rollback-plan msapi . <operationId>
```

创建角色只提交名称和父角色，MetadataService 自动计算角色 ID、closure rows、gap、level、配套角色组和
审计字段。显式 `"parent":null` 创建根角色；省略 parent 不会在已有角色体系中意外创建根。迁移可以提交
`sourceSystem + externalKey` 获得稳定目标 ID，父角色也可用同一外部身份引用。不要提交
`parentClosure`、`ancestors`、`closureRows`、`gap`、`currentLevel`、`rowId`、`columns` 或原始数据库列。

角色名称不是稳定标识，允许不同角色使用同一名称。读取允许模糊 filter；写入 selector 必须是角色 ID
或唯一的完整名称，同名匹配会失败关闭并返回候选 ID。

`cloudcc normalize msapi` 和 `cloudcc validate msapi` 都执行领域校验并返回结构化 `issues`。CLI 收到
`valid:false` 时仍输出完整 JSON，但进程返回非零退出码，批处理不得只解析屏幕文字。

完整 JSON、move、assign、删除保护、错误码和排障流程见：

```bash
cloudcc doc role devguide
```
