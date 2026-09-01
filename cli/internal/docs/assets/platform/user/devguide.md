# 用户 CLI 用户级与开发说明

用户管理走 setup-svc 直连，不纳入 MetadataService plan/apply/rollback。原因是用户账号属于运行态安全主体，创建、停用、重置密码、MFA 和登录历史会触发账号、安全、通知等运行态逻辑，不能简单按元数据表回滚。

## 主命令映射

| CLI | setup-service |
|-----|---------------|
| `cloudcc get user` / `query` / `getList` | `/api/usermange/queryUserList` |
| `cloudcc views user` / `queryViews` | `/api/usermange/queryUser` |
| `cloudcc newInfo user` / `addUserQuery` | `/api/usermange/addUserQuery` |
| `cloudcc view user` / `detail` | `/api/usermange/viewUser` |
| `cloudcc editInfo user` | `/api/usermange/editUserQuery` |
| `cloudcc create user` / `save` | `/api/usermange/saveUser` |
| `cloudcc update user` / `editSave` | `/api/usermange/editandsave` |
| `cloudcc delete user` / `deactivate` / `disable` | `/api/usermange/editandsave` with `isusing=false` |
| `cloudcc resetpw user` | `/api/usermange/resetpw` |
| `cloudcc unlock user` / `unlocked` | `/api/usermange/unlocked` |
| `cloudcc unBindMfa user` / `mfa-unbind` | `/api/usermange/unBindMfa` |
| `cloudcc choseemail user` | `/api/usermange/choseemail` |
| `cloudcc setSendFrom user` | `/api/usermange/setSendFrom` |
| `cloudcc sendemail user` | `/api/usermange/sendemail` |

## 查询用户列表

```bash
cloudcc get user <projectPath> [encodedJson]
```

示例：

```bash
cloudcc get user .
cloudcc get user . '{"start":0,"limit":50,"viewId":"view-user-default","keyword":"张"}'
```

常用字段：

| 字段 | 说明 |
|------|------|
| `start` | 起始位置，setup-web 默认从 `0` 开始 |
| `limit` | 返回条数 |
| `viewId` | 用户列表视图 ID |
| `keyword` | 搜索关键字 |
| `sort` / `dir` | 排序字段和方向 |

## 查询用户视图

```bash
cloudcc views user <projectPath> [encodedJson]
cloudcc queryViews user <projectPath> [encodedJson]
```

该命令调用 `/api/usermange/queryUser`，用于获取用户页面的视图列表。列表页通常先取视图，再用选中的 `viewId` 调 `/api/usermange/queryUserList`。

## 查看用户详情

```bash
cloudcc view user <projectPath> <userId>
cloudcc detail user <projectPath> '{"userId":"005..."}'
```

普通 ID 参数会自动组装为：

```json
{"userId":"005...","id":"005..."}
```

## 新增用户

推荐传 setup-web 表单同形 JSON：

```bash
cloudcc create user <projectPath> '{"loginName":"new.user@example.com","lastName":"张","firstName":"三","email":"new.user@example.com","profileId":"aaa000001","role":"role-sales","isusing":"true","isSendEmail":"false"}'
```

CLI 会自动包装为 setup-web 保存形状：

```json
{
  "dataJson": "{\"loginName\":\"new.user@example.com\",\"lastName\":\"张\",\"profileId\":\"aaa000001\",\"role\":\"role-sales\",\"isusing\":\"true\"}",
  "sendemail": false
}
```

如果调用方已经传入 `dataJson`，CLI 会原样发送：

```bash
cloudcc create user . '{"dataJson":"{\"loginName\":\"new.user@example.com\",\"profileId\":\"aaa000001\"}","sendemail":true}'
```

兼容简写：

```bash
cloudcc create user <projectPath> <name> <profileId> [email]
```

简写只适合最小化试运行；生产建议使用 JSON 显式传 `loginName`、`profileId`、`role`、邮箱、语言、时区、启用状态等字段。

## 编辑用户

```bash
cloudcc update user <projectPath> '{"id":"005...","lastName":"张三","email":"zhangsan@example.com","profileId":"aaa000001","role":"role-sales","isusing":"true"}'
```

CLI 会包装为：

```json
{"dataJson":"{\"id\":\"005...\",\"lastName\":\"张三\",\"isusing\":\"true\"}"}
```

如果要完全复刻 setup-web 编辑页，先执行：

```bash
cloudcc editInfo user . <userId>
```

基于返回表单补齐字段后再 `update user`。

## 停用用户

```bash
cloudcc delete user <projectPath> <userId>
cloudcc deactivate user <projectPath> <userId>
cloudcc disable user <projectPath> <userId>
```

这些命令都调用 `/api/usermange/editandsave`，发送：

```json
{"dataJson":"{\"id\":\"005...\",\"isusing\":\"false\"}"}
```

不要把用户停用理解为物理删除。若目标租户要求保留登录名、邮箱或审计信息，停用符合 setup-web 行为。

## 密码、锁定和 MFA

```bash
cloudcc resetpw user <projectPath> <userId>
cloudcc unlock user <projectPath> <userId>
cloudcc unBindMfa user <projectPath> <userId>
```

普通 ID 参数会自动携带 `userId`、`id`、`checkedid`，以兼容不同 setup-service 方法读取字段的习惯。需要页面同形参数时也可以传 JSON。

## 邮件发送流程

```bash
cloudcc choseemail user <projectPath> '{"checkedid":"005..."}'
cloudcc setSendFrom user <projectPath> '{"checkedid":"005...","emailid":"template-id"}'
cloudcc sendemail user <projectPath> '{"checkedid":"005...","biaoti":"标题","zhengwen":"正文","emailid":"template-id"}'
```

邮件流程属于 setup-svc 运行态操作，不进入 MetadataService。

## 页面辅助接口

setup-web 用户页还会调用这些接口，CLI 主命令不会把它们伪装成用户元数据 domain：

| 用途 | 接口 |
|------|------|
| 角色弹窗 | `/api/role/popup` |
| 简档弹窗 | `/api/profile/popup` |
| lookup 字段信息 | `/api/lookup/getLookupInfo` |
| lookup 数据 | `/api/lookup/getLookupData` |
| lookup 关联字段值 | `/api/lookup/getLookupRelatedFieldValue` |
| 登录历史 | `/api/log/getLoginHistory` |
| 当前用户信息 | `/api/user/getUserInfo` |
| 用户视图列表 | `/api/view/list/getViewList` |
| 访问权限用户列表 | `/api/access/permission/getUserList` |
| 访问权限 token | `/api/access/permission/getAccessToken` |

需要这些页面辅助接口时，优先使用相应已有 CLI domain；没有专用命令时按 setup-service JSON 契约调用。
