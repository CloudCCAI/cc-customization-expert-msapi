# CloudCC 实施阶段计划

## 定位

本模块用于把 CloudCC 蓝图拆成分阶段上线计划。它帮助 AI Agent 在方案设计时避免“一次性全做”的风险，把业务闭环、系统依赖、主数据准备、接口联调和用户验收纳入节奏。

## 常见阶段

```text
阶段一：核心业务闭环
阶段二：扩展业务和移动端
阶段三：系统集成和数据驱动
```

实际项目应按客户组织、外部系统准备度、数据治理成熟度和上线窗口调整。

## 分阶段原则

- 先主数据后交易。
- 先手工闭环后自动化增强。
- 先核心流程后边缘异常。
- 先平台配置后复杂高代码。
- 先内部闭环后外部系统深集成。
- 先可验收范围后体验优化。

## 相关文档

```bash
cloudcc doc methodology/blueprint devguide
cloudcc doc methodology/moduleDesign devguide
cloudcc doc methodology/integrationDesign devguide
```
