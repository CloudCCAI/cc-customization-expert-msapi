# CloudCC 项目标准治理指南

## 1. 先判断文档职责

不要按文件当前所在位置判断职责，要按内容的变化速度和事实类型分类。

| 内容 | 唯一事实源 | 其他文档允许做什么 |
|---|---|---|
| AI 必须执行的简短规则和读取门禁 | `AGENTS.md` | 引用标准 ID 和稳定路径 |
| 跨模块长期生效的方法、表达、命名、绘图、文档和评审规则 | `docs/delivery/<project-code>/00-governance/standards/` | FEAT、蓝图和任务只引用 |
| 项目交付入口和目录状态 | `00-governance/00-delivery-index.md` | README 提供短入口 |
| 功能需求、设计和验收事实 | `docs/specs/FEAT-xxx-*.md` | 标准只引用，不复制 |
| L0/L1 实际业务流程图 | `01-blueprint/processes/` | 蓝图索引登记 |
| L2/L3 详细流程和系统实现 | 对应 FEAT、模块详细设计或集成设计 | 流程索引可引用 |
| 当前任务、优先级、负责人和下一步 | `.claw/task-board.md`、任务卡、`current-status.md` | 标准不维护动态清单 |
| 技术决策和反转 | `.claw/decisions.md` | 标准引用 ADR，不重写决定 |
| 构建、运行、发布和回滚事实 | `.claw/devops.md` | 标准只定义质量门禁 |
| 历史交接材料 | `docs/handoff/` | 不反向覆盖现行事实源 |

满足以下任一条件时，文档通常属于项目标准：

- 跨两个以上模块或多个交付阶段长期生效。
- 规定统一的表达、命名、绘图、文档结构、评审或质量门禁。
- 内容变化来自方法、工具或治理机制变化，而不是单个功能需求变化。

当前业务范围、真实系统职责、对象字段、接口契约、任务优先级和验收结论不属于标准正文。

## 2. 标准目录与索引

项目标准统一放在：

```text
docs/delivery/<project-code>/00-governance/standards/
├── 00-standard-index.md
├── <NN>-<stable-standard-name>.md
└── archive/
```

要求：

- 项目根目录不保存长篇标准镜像；历史根目录材料迁移后只在历史说明中保留引用。
- `00-standard-index.md` 登记标准 ID、路径、状态、责任角色、强制读取场景和替代关系。
- 标准文件使用稳定文件名；版本号和日期写入元数据与变更记录，不为每次小修订复制新文件。
- 只有正式废止的标准才进入 `archive/`。active 标准不得位于 archive。
- 核心固定交付物继续使用技能定义的英文文件名。项目扩展标准可遵循项目 `AGENTS.md` 明确声明的文档语言规范，但 `standard_id` 必须稳定且唯一。

标准索引至少包含：

```markdown
| 标准 ID | 标准 | 状态 | 责任角色 | 强制读取场景 | 替代关系 |
|---|---|---|---|---|---|
| STD-001 | [示例标准](01-example-standard.md) | active | project-manager | 执行对应工作前 | - |
```

## 3. 标准元数据

每份标准至少包含：

```yaml
---
kind: project-standard
standard_id: STD-001
title: 示例项目标准
status: active
version: 1.0
owner_role: project-manager
effective_date: 2026-01-01
review_trigger: 项目阶段、主要工具或评审方式变化
supersedes: none
---
```

状态建议：

- `draft`：正在编写，尚未成为强制规则。
- `active`：当前生效；相关场景必须通过 `AGENTS.md` 或标准索引发现。
- `deprecated`：仍可追溯但不再用于新工作；必须声明替代标准或废止原因。
- `archived`：已移入 archive，不再生效。

`version` 表示标准正文版本，不放入文件名。`standard_id` 是跨语言、跨文件名调整时的稳定身份。

## 4. AGENTS 读取门禁

`AGENTS.md` 只写短门禁，不复制正文。例如：

```markdown
- 创建、修改或评审项目业务流程图前，必须读取 STD-001：
  `docs/delivery/<project-code>/00-governance/standards/01-business-process-standard.md`。
```

门禁必须说明：

- 触发场景。
- 标准 ID。
- 当前稳定路径。

标准的图形、颜色、节点命名、评审清单和版本规则只在标准正文维护。

## 5. 标准正文与业务事实分层

标准可以维护：

- 表达符号、颜色含义、泳道规则和节点命名方法。
- 文档或流程图的分层、结构、质量检查和评审门禁。
- 稳定命名、版本、变更和归档规则。
- 如何引用 FEAT、ADR、接口契约和验证证据。

标准不得独立维护：

- 当前端到端业务主链路。
- 当前各系统的真实职责边界。
- 当前对象、字段、状态、枚举和数据口径。
- 当前优先级、负责人、完成状态和下一步。
- 当前功能是否已通过客户或业务验收。

遇到第二类内容时，把正文改成指向当前事实源的引用。示例可以说明表达方式，但必须明确“示例不是当前业务事实”。

## 6. 流程图治理

创建或评审流程图前：

1. 读取项目 `AGENTS.md`。
2. 查找 `00-governance/standards/00-standard-index.md`。
3. 存在适用标准时读取标准正文；不存在时只使用本技能的通用建议，不擅自建立全项目颜色或图形强制规则。
4. 从对应 FEAT、蓝图、ADR 或接口契约获取业务事实。
5. 从任务板获取绘制优先级、责任人和状态。

产物位置：

```text
L0/L1：docs/delivery/<project-code>/01-blueprint/processes/
L2：对应 FEAT 或模块详细设计
L3：对应 FEAT、模块详细设计或集成设计
```

每张正式流程图至少登记：层级、状态、事实源、责任角色和评审证据。静态检查或自动化测试不能替代业务责任人的流程确认。

## 7. 版本、废止与归档

- 小修订更新原文件的 `version` 和变更记录。
- 业务事实变化时更新事实源，标准只在方法或门禁变化时升级。
- 标准被替代时，旧标准先改为 `deprecated` 并登记替代标准；完成迁移后再移入 `archive/`。
- 标准索引必须保留替代关系，避免旧链接被误当成现行规则。
- 评审快照可以进入同级 `snapshots/`，但稳定工作文件仍是唯一现行入口。

## 8. 只读校验

运行：

```bash
cloudcc doctor project-governance <projectPath>
```

该命令只检查本地文件，不创建目录、不修改文档、不调用 CloudCC 或 MetadataService。它验证：

- `docs/delivery/<project-code>` 的发现结果。
- standards 目录、标准索引和流程图索引是否存在。
- `standard_id` 是否唯一，必需元数据是否完整。
- active 标准是否被索引且未位于 archive。
- `AGENTS.md` 是否包含对应标准 ID 和稳定路径读取门禁。
- 稳定文件名是否误带版本或日期。

校验通过只证明结构和门禁成立，不证明标准内容、业务流程或客户验收正确。

## 9. 通用技能包边界

- 客户名称、真实项目短码、客户系统职责、客户 FEAT 编号、租户标识和本地绝对路径不得进入技能说明、内置文档、dist 或归档。
- 客户项目可以作为外部验证输入，但证据保留在项目仓库、`.claw/`、`docs/specs/`、测试专用目录或外部 evidence。
- 通用示例使用虚构项目短码和中性系统名称，不从客户案例反推 CloudCC 平台通用事实。
