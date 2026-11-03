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
审计字段。不要提交 `parentClosure`、`ancestors`、`closureRows`、`gap`、`currentLevel`、`rowId`、
`columns` 或任何原始数据库列。

角色名称不是稳定标识。读取允许模糊 filter，但写入 selector 必须是角色 ID 或唯一的完整名称；同名匹配
会失败关闭并返回候选 ID。

完整 JSON、move、assign、删除保护、错误码和排障流程见：

```bash
cloudcc doc role devguide
```
