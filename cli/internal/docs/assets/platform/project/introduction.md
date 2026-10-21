# CloudCC 项目介绍

`project` 用于创建本地 CloudCC 二开项目，统一管理前端组件、页面、自定义类、触发器、定时类和环境配置，为后续校验与发布提供项目目录。

```bash
cloudcc create project demo-crm
cloudcc create project horizontal-demo --platform horizontal
```

缺省创建 Lightning 项目，横纵版需要显式选择。创建本地骨架不会自动发布 CRM 资源；完成配置后先运行 `cloudcc doctor platform <projectPath>` 和 `cloudcc doctor provider <projectPath>`，再执行具体资源操作。配置、认证与交付目录说明见 `cloudcc doc project devguide` 和 `cloudcc doc config devguide`。
