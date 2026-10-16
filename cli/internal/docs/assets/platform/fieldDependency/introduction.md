# 字段依赖

字段依赖用于根据一个控制字段的值限制另一个字段可选的值。控制字段支持复选框和单选列表，依赖字段支持单选列表和多选列表；两个字段必须属于同一对象，并且不能形成直接或间接循环。

MSAPI 提供只读查询、候选字段查询以及 plan/apply 写入。Lightning UIAPI 使用 setup-service 的字段依赖接口。horizontal UIAPI 尚无稳定的主应用写入契约，因此会明确拒绝执行。

```text
cloudcc get fieldDependency <projectPath> <object>
cloudcc detail fieldDependency <projectPath> <object> <controlField> <dependentField>
cloudcc candidates fieldDependency <projectPath> <object>
cloudcc create fieldDependency <projectPath> <@spec.json>
cloudcc apply msapi <projectPath> <planId>
```

