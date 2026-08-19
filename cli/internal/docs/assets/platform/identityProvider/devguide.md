# CloudCC 身份提供商 CLI 命令说明

## 支持的命令

| 操作 | 说明 |
|------|------|
| `create` | 创建新身份提供商 |
| `get` | 查询身份提供商列表 |
| `delete` | 删除身份提供商 |
| `download` | 下载身份提供商证书 |

## CLI 命令详解

### 创建身份提供商

```bash
cloudcc create identityProvider <path> <entityid> <acsurl>
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `path` | 是 | 项目路径，`.` 表示当前目录 |
| `entityid` | 是 | 实体 ID（Service Provider Entity ID）|
| `acsurl` | 是 | ACS URL（Assertion Consumer Service URL）|

**示例：**

```bash
# 创建身份提供商
cloudcc create identityProvider . "http://testdev.cloudcc.cn/ccdomaingateway/apisvc" "http://testdev.cloudcc.cn/ccdomaingateway/apisvc/saml2/sp/sso/acs"
```

### 查询身份提供商列表

```bash
cloudcc get identityProvider <projectPath> [encodedCondJson]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `encodedCondJson` | 否 | URI 编码后的查询条件 JSON |

**查询条件参数：**

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `start` | number | 起始位置，默认 0 |
| `limit` | number | 每页条数，默认 30 |
| `keyword` | string | 搜索关键词 |

**示例：**

```bash
# 获取所有身份提供商
cloudcc get identityProvider .

# 带查询条件（搜索关键词）
cloudcc get identityProvider . '%7B%22keyword%22%3A%22test%22%2C%22limit%22%3A50%7D'
```

### 删除身份提供商

```bash
cloudcc delete identityProvider <projectPath> <appId>
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `appId` | 是 | 身份提供商的 app 字段值（从列表接口获取）|

**示例：**

```bash
# 删除身份提供商
cloudcc delete identityProvider . QbjH30d9Oy
```

### 下载身份提供商证书

```bash
cloudcc download identityProvider <projectPath> [savePath]
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `projectPath` | 否 | 项目路径，默认当前目录 |
| `savePath` | 否 | 证书保存路径，默认当前目录 |

**示例：**

```bash
# 下载证书到当前目录
cloudcc download identityProvider .

# 下载证书到指定路径
cloudcc download identityProvider . ./certificates
```
