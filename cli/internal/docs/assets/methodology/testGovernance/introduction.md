# CloudCC 建议式端到端测试治理

## 定位

本方法为 CloudCC 项目提供“自动分析建议、人工确认范围、执行与跳过均留证”的端到端测试治理。它用于在开发效率与上线风险之间取得可审计的平衡，不把每次触发器、模块或元数据修改自动升级成强制全量回归。

## 双层机制

```text
建议层：变更集 -> 影响目录 -> 场景目录 -> 推荐范围、理由、风险和预计时间
确认层：人类选择 skip / smoke / feature-closure / affected-chain / full-core
```

AI 建议始终是 `advisory: true`、`blocking: false`。人类 decision 是本次测试范围的唯一事实源；缩减或跳过必须说明原因。未执行测试时只能标记 `unverified` 或 `risk_accepted`，不得声明测试通过。

## 资产分层

- `00-governance/standards`：长期测试治理标准。
- `07-testing-cutover`：项目 UAT、权限、影响、数据、环境和追踪矩阵。
- `test-assets`：可复用机器测试资产。
- `evidence/testing/decisions`：建议与人工决定。
- `evidence/testing/runs`：单次运行证据。
- `08-release-evidence`：发布候选采用的证据入口。
- `.claw/test-report.md`：当前测试状态摘要。

## 边界

- CloudCC classes/triggers/timer 的 compile/validate-before-save 仍是低成本技术安全检查，不等于业务闭环通过。
- SIT、UAT、生产授权和 Go/No-Go 仍由目标角色和业务负责人确认。
- 通用技能只包含中性方法、模板和 CLI，不包含客户用例、租户数据、测试报告、账号或凭据。

详细目录、输入契约和命令见：

```bash
cloudcc doc methodology/testGovernance devguide
```
