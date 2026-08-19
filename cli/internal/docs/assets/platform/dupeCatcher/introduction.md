# CloudCC 查重过滤器（Dupe Catcher）介绍

查重过滤器用于在**新建或编辑业务数据**时，按配置的**对象、字段与条件**检测是否与已有记录重复，从而提示或阻断保存。平台侧通过 **过滤器（Filter）** 与 **规则（Rule）** 两层建模：过滤器绑定目标对象与启用范围，规则描述具体匹配字段与逻辑。

## 适用场景

- 客户、联系人、线索等主数据**去重**（姓名 + 手机、邮箱等组合）。
- 同一业务对象下**多套查重策略**（不同过滤器、不同规则集）。
- 与 **one-setup-web** 中查重配置能力一致，CLI 面向**导出配置、脚本化维护、与本地工程联调**。

## 能力边界（与开发指南的关系）

- **介绍（本文）**：说明是什么、何时用、与平台概念对应关系。
- **开发指南**：`cloudcc doc platform/dupeCatcher devguide` — 列出 `get` / `detail` / `create` / `delete` 等 CLI 命令、参数与接口前缀 `{setupSvc}/api/duplication/*`。

## 相关命令入口

```bash
cloudcc doc platform/dupeCatcher introduction   # 本文
cloudcc doc platform/dupeCatcher devguide       # CLI 与接口说明
```
