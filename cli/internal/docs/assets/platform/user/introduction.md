# CloudCC 用户

用户（User）对应 setup-web 路由 `/settings/usermanage/user`，用于维护登录账号、启停状态、角色、简档、联系方式、MFA 和用户列表视图。

## 当前实现

用户管理不是 MetadataService 回滚型低代码 domain。本 CLI 保持 setup-svc 直连能力，并已从旧 `/api/user/*` 迁移到当前 setup-web 使用的 `/api/usermange/*` 接口。

## 常用 CLI

```bash
cloudcc get user .
cloudcc views user .
cloudcc view user . <userId>
cloudcc create user . '{"loginName":"new.user@example.com","lastName":"张","profileId":"aaa000001","role":"role-sales"}'
cloudcc update user . '{"id":"005...","lastName":"张三","isusing":"true"}'
cloudcc delete user . <userId>
```

`delete user` 不做物理删除，会通过 `/api/usermange/editandsave` 把 `isusing=false` 写入用户表单，实现停用。

## 相关辅助能力

用户页面会使用角色、简档、lookup、访问权限、登录历史等辅助接口。CLI 主命令覆盖用户主流程；复杂页面级操作可直接传 setup-web 同形 JSON。
