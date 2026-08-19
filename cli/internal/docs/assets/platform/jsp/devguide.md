# CloudCC JSP 迁移开发指南

## 1. 命令总览

```bash
cloudcc doc platform/jsp devguide
cloudcc analyze jsp <encodeURI(JSON.stringify(params))>
cloudcc split jsp <encodeURI(JSON.stringify(params))>
```

## 2. 参数结构

推荐将参数组织成 JSON，再使用 `encodeURI(JSON.stringify(params))` 传入 CLI。

参数示例：

```json
{
  "jspPath": "customize/wt_ly.jsp",
  "helpDocPath": "jspTemp/help.md",
  "projectPath": "/abs/project",
  "outputProjectPath": "/abs/output-project",
  "className": "WtLyMigratedService",
  "componentName": "wt-ly-migrated",
  "reportDir": "migration-report",
  "mode": "apply",
  "overwrite": false
}
```

字段说明：

- `jspPath`：必填，JSP 文件路径，可为绝对路径或相对 `projectPath`
- `helpDocPath`：可选，帮助文档路径
- `projectPath`：可选，项目根目录，默认当前工作目录
- `outputProjectPath`：可选，输出项目目录，默认等于 `projectPath`
- `className`：可选，生成的自定义类名
- `componentName`：可选，生成的组件名
- `reportDir`：可选，报告目录，默认 `migration-report`
- `mode`：仅 `split jsp` 有意义，支持 `apply` / `dry-run`
- `overwrite`：是否覆盖现有文件，默认 `false`

## 3. 常用命令示例

### 3.1 查看开发文档与规则

```bash
cloudcc doc platform/jsp devguide
```

### 3.2 分析 JSP（不写文件）

```bash
cloudcc analyze jsp '{"jspPath":"customize/wt_ly.jsp","projectPath":"."}'
```

### 3.3 生成迁移结果

```bash
cloudcc split jsp '{"jspPath":"customize/wt_ly.jsp","projectPath":".","overwrite":false}'
```

## 4. 输出结果

`split jsp` 默认会输出：

- `classes/<ClassName>/<ClassName>.java`
- `classes/<ClassName>/<ClassName>Test.java`
- `classes/<ClassName>/config.json`
- `frontend/pagecomponents/<componentName>/<componentName>.vue`
- `frontend/pagecomponents/<componentName>/config.json`
- `<reportDir>/<jspBaseName>.migration.md`

## 5. 推荐流程

1. 先阅读 `cloudcc doc platform/jsp devguide` 中的规则章节
2. 根据旧 URL 或仓库结构定位 `customize/*.jsp`
3. 用 `cloudcc analyze jsp ...` 做单文件分析
4. 调整类名、组件名、输出目录等参数
5. 用 `cloudcc split jsp ...` 生成迁移草稿
6. 手工补全生成类中的业务逻辑并验证依赖

## 6. JSP 迁移规则

### 6.1 从旧系统 URL 定位 JSP 源文件

- 若页面或配置中出现形如 `/controller.action?name=<name>` 的地址，应在项目根目录下的 `customize` 目录中查找对应 JSP
- 常见约定：`<projectPath>/customize/<name>.jsp`
- 解析出真实路径后，可将其作为 `jspPath` 传入 CLI `cloudcc analyze jsp` / `cloudcc split jsp`，或对应的 MCP 工具

### 6.2 批量迁移时的建议流程

1. 先查看本节规则；若走 CLI，可执行 `cloudcc doc platform/jsp devguide`
2. 在仓库中列举或搜索 `customize` 下需处理的 `.jsp`
3. 对每个 JSP 单独执行 `cloudcc analyze jsp` 或 `cloudcc split jsp`
4. 不要假设服务端会自动循环多个文件

### 6.3 PageClsInvoker 调用约定

在服务端自定义类中，通过 `PageClsInvoker` 调用其它自定义类方法，典型形式如下：

```java
Object result = new PageClsInvoker(userInfo).invoker("TargetClassName", "targetMethodName", argList);
```

迁移 JSP 内出现的同类调用时，保持上述调用形态，并核对目标类、方法在平台中已存在且参数类型一致。

### 6.4 与迁移工具的配合

- `cloudcc analyze jsp` / `cloudcc split jsp`（以及对应 MCP 工具）会从 JSP 文本中识别 `PageClsInvoker(userInfo).invoker("类名","方法名", ...)` 形式的依赖，并写入迁移报告
- 具体业务实现仍需在生成后的自定义类中补全或调整
