# 横纵版自定义组件

`customComponent` 是横纵版独有的 main-app 组件资源，与 Lightning `pagecomponent` 完全不同。其源文件使用 `<cc:component>` DSL，经 main-app 编译为组织隔离的 JSP，可被 VisualPage 通过平台 component 标签引用。

本资源仅在 `platformMode=horizontal` 时可用，命令如下：

```text
cloudcc create customComponent <name>
cloudcc publish customComponent <name> [projectPath]
cloudcc get customComponent <projectPath>
cloudcc detail customComponent <projectPath> <id>
cloudcc used customComponent <projectPath> <id>
cloudcc delete customComponent <projectPath> <id>
```

本地源码位于 `customComponent/<name>/<name>.cccomponent`。不要把 Lightning Vue/UMD bundle、customPage canvas JSON 或 pagecomponent 配置复制到该目录。
