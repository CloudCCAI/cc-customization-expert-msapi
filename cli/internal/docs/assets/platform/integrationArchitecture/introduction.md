# CloudCC 集成架构

## 定位

CloudCC 集成能力用于连接外部身份、外部数据、外部业务系统和外部交互渠道。

## 已确认能力

手册确认的集成和沟通能力包括：

- 电话/CTI
- Email to Case
- Email to Lead
- 在线聊天
- 在线个案
- 社交媒体设置
- 客户满意度
- 消息通知
- Mailchimp、天眼查、企业微信、钉钉、短信等第三方应用
- 身份提供商
- 单点登录

当前技能文档确认的开发和集成能力包括：

- OpenAPI
- 服务端类
- 定时类和定时作业
- 自定义组件
- 静态资源
- 身份提供商和 SSO 文档
- sidecar 目录约定

`cc-setup-service` 源码结构中可见 integration、authorizeProvider、securityControls、webhook、dingding、callcenteradapter 等相关 controller/service 包，说明后端服务侧存在对应 setup 能力。但具体接口参数仍以现有文档或源码进一步确认。

## 集成类型

### 数据接口集成

使用 OpenAPI 读取、创建、更新、删除业务对象数据。

### 服务端业务集成

使用类封装服务端逻辑，必要时由触发器、定时类、自定义页面或组件调用。

### 定时同步

使用定时类和定时作业周期性同步外部系统或做补偿处理。

### 前端集成

使用 pagecomponent、自定义页面、静态资源接入外部 SDK、地图、图表、AI 助手等前端能力。

### 身份集成

使用身份提供商、单点登录、OAuth/OpenID Connect/SAML 等能力对接企业身份体系。

### Sidecar 中间程序

当逻辑不适合放在 CloudCC 平台内运行时，可放在 `sidecar/` 中作为外挂中间程序。当前技能只定义目录约定，具体运行框架需按项目选择。

## 相关文档

```bash
cloudcc doc platform/openapi introduction
cloudcc doc platform/classes introduction
cloudcc doc platform/timer introduction
cloudcc doc platform/scheduleJob introduction
cloudcc doc platform/pagecomponent introduction
cloudcc doc platform/identityProvider introduction
cloudcc doc platform/singleSignOn introduction
```
