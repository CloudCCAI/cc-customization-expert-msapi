# CloudCC ALM 与多环境交付

## 定位

ALM 用于管理 CloudCC 二开项目从开发、测试到生产的配置、代码、组件和软件包迁移。

## 已确认能力

手册确认：

- 软件包可将平台开发好的类、自定义页面、自定义组件、应用程序等打包。
- 软件包可通过安装 URL 或 id 在新环境安装。
- 静态资源用于上传 zip、图像、样式表、JavaScript 程序和其他文件。

当前技能确认：

- 项目根目录使用 `cloudcc-cli.config.json` 管理环境。
- 本地目录分为 `frontend/`、`backend/`、`sidecar/`。
- `classes`、`triggers`、`timer`、`pagecomponent`、`staticResource` 等可通过 CLI 管理部分本地文件和发布。
- `publish pagecomponent` 调用项目 `frontend/` 中的 Vue 构建链。

## 交付对象

常见交付内容包括：

- 对象和字段
- 页面布局、菜单、应用
- 简档、权限集、共享规则
- 验证规则、查重过滤器
- 类、触发器、定时类、定时作业
- 自定义页面、pagecomponent
- 客户端脚本、HTML、静态资源
- OpenAPI/集成配置
- sidecar 中间程序

## 多环境原则

建议至少区分：

- dev：开发环境
- test：测试/验收环境
- prod：生产环境

`cloudcc-cli.config.json` 的 `use` 字段控制当前环境。

## 相关文档

```bash
cloudcc doc platform/project devguide
cloudcc doc platform/config devguide
cloudcc doc platform/classes devguide
cloudcc doc platform/triggers devguide
cloudcc doc platform/pagecomponent devguide
cloudcc doc platform/staticResource devguide
```
