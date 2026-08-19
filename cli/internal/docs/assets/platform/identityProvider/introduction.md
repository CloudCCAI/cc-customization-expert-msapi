# 身份提供商介绍

## 功能概述

身份提供商（Identity Provider）模块用于管理 CloudCC 平台的 SAML 身份提供商配置，支持以下功能：

- **查看身份提供商列表**：查询已配置的所有 SAML 身份提供商
- **新建身份提供商**：创建新的 SAML 身份提供商配置
- **删除身份提供商**：删除已配置的 SAML 身份提供商
- **下载证书**：下载身份提供商的 SAML 证书

## 适用场景

- 企业单点登录（SSO）集成
- SAML 2.0 身份提供商配置
- 多身份提供商管理

## 字段说明

### 身份提供商字段

| 字段名 | 说明 | 默认值 |
|--------|------|--------|
| `entityid` | Service Provider Entity ID | 用户输入 |
| `acsurl` | Assertion Consumer Service URL | 用户输入 |
| `issuername` | 签发者名称 | 空字符串 |
| `nameidtype` | Name ID 类型 | FederationId |
| `nameidformat` | Name ID 格式 | 1 |
| `enablelogout` | 启用注销 | false |
| `logouturl` | 注销 URL | 空字符串 |
| `custattrname` | 自定义属性名称 | 空字符串 |
