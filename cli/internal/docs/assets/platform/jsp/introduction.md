# CloudCC JSP 迁移介绍

## 1. 模块定位

`jsp` 模块用于辅助将旧 JSP 页面迁移为 CloudCC 自定义类与自定义组件。

当前资源主要提供三类能力：

- 读取固定迁移规则
- 对单个 JSP 做 dry-run 分析
- 对单个 JSP 输出迁移产物

可通过以下命令查看：

```bash
cloudcc doc platform/jsp introduction
cloudcc doc platform/jsp devguide
```

## 2. 当前支持的命令

```bash
cloudcc doc platform/jsp devguide
cloudcc analyze jsp <encodeURI(JSON.stringify(params))>
cloudcc split jsp <encodeURI(JSON.stringify(params))>
```

其中 `params` 主要包含：

- `jspPath`
- `helpDocPath`
- `projectPath`
- `outputProjectPath`
- `className`
- `componentName`
- `reportDir`
- `mode`
- `overwrite`

## 3. 适用场景

- 旧系统 JSP 页面迁移评估
- 识别 `reason` 分支、对象依赖、`PageClsInvoker` 调用
- 生成 CloudCC 自定义类、自定义组件和迁移报告草稿

## 4. 使用建议

- 先阅读 `cloudcc doc platform/jsp devguide`
- 再执行 `cloudcc analyze jsp ...` 做分析
- 确认命名、输出目录和覆盖策略后，再执行 `cloudcc split jsp ...`
