# CloudCC ALM 与发布指南

## 本地项目结构

当前 Go 技能生成结构：

```text
project/
├── cloudcc-cli.config.json
├── frontend/
│   └── pagecomponents/
├── backend/
│   ├── classes/
│   ├── triggers/
│   └── schedule/
└── sidecar/
```

## 发布前检查

### 1. 环境配置

确认：

- `cloudcc-cli.config.json` 存在
- `use` 指向正确环境
- `safetyMark` 和 `CloudCCDev` 已配置
- 不要把生产环境误设为当前环境

### 2. 元数据清单

列出本次变更涉及：

- 对象
- 字段
- 布局
- 菜单
- 应用
- 权限
- 自动化
- 代码和组件

### 3. 权限验收

至少用目标简档用户验证：

- 应用可见
- 菜单可见
- 对象操作权限
- 字段可见/只读
- 记录共享范围
- 移动端入口

### 4. 自动化验收

验证：

- 验证规则
- 查重过滤器
- 触发器
- 工作流/批准
- 定时作业

### 5. 高代码发布

按能力选择命令：

```bash
cloudcc publish classes <name>
cloudcc publish triggers <object/name>
cloudcc publish timer <name>
cloudcc publish pagecomponent <name> [projectPath]
cloudcc publish staticResource ...
```

类、触发器和定时类不是直接 save：classes 会先做本地 `FagTemplate` 编译验证，再调用目标 setup-svc validate；triggers 和 timer 不做本地包装编译，只调用目标 setup-svc validate。验证通过后才保存。

从 CLI/技能 `2.2.7` 开始，高代码发布依赖 CLI 发布能力和 setup-svc validate 接口；目标 setup-svc 建议为 `19.3.R20` 或更高版本。高代码发布本身不要求 MetadataService 版本。setup-svc 可能存在 `DEV/B/G` 或客户分支版本，CLI 只在初始化/doctor/config 检查中提醒，不因版本字符串低于基线直接阻断；若实际缺少 `/api/ccfag/validate`、`/api/trigger/validate` 或 `/api/ccPeak/validate`，publish 会在远程 validate 阶段失败并返回原始响应。

| 资源 | 本地验证 | 远程 validate | 保存 |
| --- | --- | --- | --- |
| classes | `FagTemplate` 编译 | `POST /api/ccfag/validate` | `POST /api/ccfag/save` |
| triggers | 不执行本地包装编译 | `POST /api/trigger/validate` | `POST /api/triggerSetup/saveTrigger` |
| timer | 不执行本地包装编译 | `POST /api/ccPeak/validate` | `POST /api/ccPeak/save` |

validate 实际读取字段需要按资源区分：classes/timer 的 validate 只编译 `source`；trigger 的 validate 会读取 `triggerSource`、`apiname`、`triggerTime`、`version`，其中 `triggerTime` 决定是否按 batch trigger 编译。CLI 可能额外携带 `id`、`name`、`folderId/folderid`、`isactive`、`targetObjectId`、`remark` 等字段，是为了和后续 save payload 保持一致，不代表这些字段都参与 validate 编译判断。

从 CLI/技能 `2.2.38` 开始，classes、triggers、timer 适配 setup-svc 新版自定义代码版本语义：创建默认发送 `version=3`；更新先读取目标 detail，优先沿用线上记录的 `version`，如果线上 version 为空则按旧版 `2` 处理，保存后再把线上返回的 ID 和版本写回本地 `config.json`。这样旧本地配置里的 `version=2` 不会把已升级到版本 3 的线上自定义类、触发器或定时类降级。

validate 失败时，CLI 会把本地 classes 编译诊断或 setup-svc 的 `returnInfo`、`data.errors`、`data.warnings`、原始 `responseBody` 返回给调用方，并停止后续 save。trigger 需要特别注意编码：`/api/trigger/validate` 直接传原始源码，`/api/triggerSetup/saveTrigger` 才传 URLDecoder-compatible 编码源码；classes/timer 的 validate 和 save 都传编码源码，源码字面量 `+` 会保留为 `%2B`。

具体参数以对应模块 `devguide` 为准。

## 软件包使用边界

手册确认软件包可迁移类、自定义页面、自定义组件、应用程序等。具体哪些组件类型可添加、依赖关系如何解析、是否覆盖已有配置，需要以目标租户的软件包页面为准。

## 行业通用发布建议

- 每次发布准备变更清单和回滚方案。
- 配置变更和代码变更分开验收。
- 先发布后端类和触发器，再发布依赖它们的页面或组件。
- 生产发布前冻结目标环境手工配置。
- 记录已安装软件包 id、版本和安装 URL。
- sidecar 独立管理部署、环境变量、日志和回滚。

## 待确认

当前技能未完整实现软件包 CLI，也未覆盖所有后台配置迁移能力。涉及软件包安装、上载、组件依赖解析时，应以后台页面或后续补充文档为准。
