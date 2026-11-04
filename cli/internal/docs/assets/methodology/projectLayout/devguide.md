# CloudCC 项目目录规范与只读检查

## 1. 先读项目规则，再选择目录

按职责划分源码、工具、设计、原始资料、测试资产、执行证据和交付物；每类内容保留一个权威位置。下述结构定义技能的通用标准，不从客户目录反推标准。实施前读取项目 AGENTS、目录登记和事实源索引；发现差异先报告，目录检查不自动迁移项目内容，也不把差异写回技能标准。

`docs/project-layout.json` 是可选的机器目录登记，不包含租户连接信息。登记只表达路径、职责和版本控制策略，不复制 FEAT、ADR、交付状态或真实配置。项目规范仍负责责任人、保留期限、批准和例外记录。

## 2. 完整目录结构、职责与创建条件

### 2.1 标记与适用范围

| 标记 | 创建条件 |
|---|---|
| 【必】 | 采用项目治理规范时建立的基础入口；不表示 CLI 已自动生成或默认 doctor 已强制检查 |
| 【阶段】 | 进入对应实施、测试、发布或交付阶段，且在项目范围内时建立 |
| 【按需】 | 使用相应能力、存在对应内容时创建；不为凑齐结构创建空目录 |
| 【本地】 | 按需生成，仅供本机使用；真实配置、缓存及临时资料必须被 Git 忽略 |

子项未单独标记时，继承父目录的创建条件。`<project-code>`、`<objectApi>`、`<name>`、`<outputId>` 等是待替换的标识，不是字面目录名。以下是通用完整结构；阶段目录是否适用由已确认的项目范围决定。

### 2.2 完整结构

```text
<project-root>/
├── README.md                              【必】项目说明、使用方法和稳定入口
├── AGENTS.md                              【必】智能体工作规则、规范和门禁
├── .gitignore                             【必】本地配置、缓存及临时产物忽略规则
├── cloudcc-cli.config.example.json         【按需】可提交的占位配置示例
├── cloudcc-cli.config.json                 【本地】项目级环境连接配置
│
├── .claw/                                 【必】项目治理状态和工作记录
│   ├── current-status.md                         当前状态、活动任务和下一步入口
│   ├── goals.md                                  项目目标、范围与成功标准
│   ├── decisions.md                              架构与关键决策记录
│   ├── task-board.md                             任务索引、优先级和依赖
│   ├── task-archive.md                           已完成或取消任务的归档索引
│   ├── tasks/                             【阶段】每个任务的独立状态文件
│   ├── issue-list.md                             问题、缺陷和阻塞项
│   ├── test-report.md                            验证结果摘要及证据链接
│   └── devops.md                                 环境、构建、发布和运维约定
│
├── references/                            【必】原始参考资料入口
│   ├── README.md                                 来源、用途、许可和采纳状态索引
│   ├── raw/                               【按需】未经整理的原始输入
│   ├── docs/                              【按需】外部参考文档
│   ├── code/                              【按需】参考代码、外部样例
│   └── assets/                            【按需】图片、媒体和其他参考素材
│
├── docs/                                  【必】项目规格、设计和实施文档
│   ├── project-layout.json                【按需】机器可检查的目录登记
│   ├── specs/                             【必】项目基线、需求规格和设计事实
│   │   ├── PROJECT-BASELINE.md             【阶段】存量接入或需要共享现状时建立基线
│   │   ├── FEAT-xxx-*.md                   【阶段】功能规格及验收条件
│   │   └── cloudcc/                        【按需】元数据变更等机器执行输入
│   ├── handoff/                           【阶段】交接说明，引用现行设计与证据
│   └── delivery/<project-code>/           【阶段】正式实施过程文档
│       ├── 00-governance/                         实施治理、范围和标准
│       │   └── standards/                        项目标准及标准索引
│       ├── 01-blueprint/                          业务蓝图、模块与流程边界
│       │   └── processes/                        端到端业务流程
│       ├── 02-global-modeling/                    对象、字段、关系、状态和全局选项
│       ├── 03-module-design/                      各业务模块的详细设计
│       ├── 04-security/                           角色、简档、共享和字段权限设计
│       ├── 05-integration/                        系统接口、映射、幂等与异常处理
│       ├── 06-data-migration/                     数据迁移映射、清洗、执行与核验方案
│       ├── 07-testing-cutover/                    测试范围、UAT、切换与回滚方案
│       └── 08-release-evidence/                   发布证据索引、门禁和上线验证结论
│
├── backend/                               【按需】平台后端高代码
│   ├── classes/<ClassName>/                      服务端类源码及资源配置
│   ├── triggers/<objectApi>/<TriggerName>/        对象触发器源码及资源配置
│   ├── schedule/<TimerClassName>/                定时执行类源码及资源配置
│   └── lib/                                      后端开发、编译使用的依赖
│
├── frontend/                              【按需】前端高代码
│   ├── scripts/<objectApi>/<name>/                客户端业务脚本
│   │   ├── <name>.js                             脚本源码
│   │   └── config.json                           脚本资源配置
│   ├── pagecomponents/<name>/                    Lightning 页面组件源码及配置
│   └── build/                                    可发布的预构建组件 bundle
│
├── html/                                  【按需】平台 HTML 资源及配置
├── staticResource/                        【按需】平台静态资源
├── visualpage/                            【按需】横纵版 VisualPage 页面
├── customComponent/                       【按需】横纵版自定义组件
├── sidecar/                               【按需】平台外部服务，内部按技术栈组织
│                                                  承载编排、回调、队列、重试等
│
├── tools/                                 【按需】审计、构建、迁移、检查与运维工具
│
├── test-assets/                           【阶段】可复用的测试资产
│   ├── schemas/                                  测试输入和资产结构约束
│   ├── catalog/                                  资源影响映射与场景索引
│   ├── scenarios/                                带稳定编号的测试场景定义
│   ├── suites/                                   测试集合，引用场景编号
│   ├── fixtures/                                 生成式或获准脱敏的数据与初始化逻辑
│   ├── assertions/                               状态、关系、权限和读回断言
│   └── automation/                               API/UI 自动化和测试执行器
│
├── evidence/                              【阶段】实际执行产生的可追溯证据
│   └── testing/                                  测试范围决定与实际运行证据
│       ├── decisions/<changeSetId>/               测试建议与人工范围决定
│       │   ├── recommendation.json
│       │   └── human-decision.json
│       └── runs/<runId>/                          每次独立测试运行
│           ├── manifest.json                     运行清单、状态和证据引用
│           ├── summary.md                        执行结果摘要
│           ├── readbacks/                        实际读回与核对结果
│           ├── screenshots/                      实际运行截图
│           ├── logs/                             实际运行日志
│           └── defect-links.md                   缺陷及关联记录
│
├── outputs/                               【阶段】接收方实际获得的正式交付物
│   ├── README.md                                 交付目录说明
│   ├── 00-output-index.md                        人工阅读的交付索引
│   ├── output-manifest.json                      交付物、版本、状态和摘要登记
│   └── <outputId>/                               按交付物组织的实体或外部制品引用
│
├── .agents/                               【按需】项目专属智能体能力
│   └── skills/                                   项目专属技能，不重复镜像通用技能
│
├── .claw-local/                           【本地】个人工作状态与临时资料
├── captures/                              【本地】临时采集内容，正式证据归入 evidence
└── .cloudcc-cache.json                    【本地】CLI 缓存
```

### 2.3 内容归属与阶段规则

| 内容 | 权威位置 | 边界 |
|---|---|---|
| 外部原始资料、参考代码与样例 | references | 尚不等于已确认需求或生产源码；采纳后在正式规格或源码中落实 |
| 项目状态、任务和验证摘要 | .claw | 链接详细规格及证据，不复制整份设计或运行记录 |
| 功能需求、设计与验收条件 | docs/specs | 一个功能维护一份主规格；交接文档引用现行事实源 |
| 实施方案、标准、蓝图及发布证据索引 | docs/delivery | 保存工作文档和索引，实际运行文件归 evidence |
| 业务页面运行的客户端脚本 | frontend/scripts | 前端高代码源码及资源配置 |
| 帮助开发、迁移和运维的工程脚本 | tools | 工程辅助工具；需要正式交付时另在 outputs 登记冻结版本 |
| 可复用测试场景、脚本、数据和断言 | test-assets | 不存放某次运行日志；fixtures 只采用生成式或获准脱敏的数据 |
| 某次执行的日志、截图、读回和审批 | evidence | 与可复用测试资产分开；测试重跑或决策修订使用新的 ID |
| 接收方实际获得的冻结成果 | outputs | 维护 manifest、版本和摘要，引用设计与运行证据；大型制品可登记外部引用 |

- 项目治理接入先建立 README、AGENTS、忽略规则、状态入口、规格入口和参考资料索引。存量项目在广泛实施前建立 PROJECT-BASELINE；其他项目按现状共享需要建立基线，不伪造未验证的项目事实。
- 正式实施按已确认范围逐步建立 delivery 子目录。集成、迁移等不适用项在交付索引中说明，不创建空文档充数。固定交付文件名以 `cloudcc doc deliveryPlan devguide` 为准。
- 进入测试治理时，资产、决策和运行目录遵循 `cloudcc doc testGovernance devguide`。目录树展示完整职责，不代表每次运行都必须产生截图或日志；实际证据按本次测试范围和契约生成。
- 实施文档的同级 `snapshots/` 按评审、签字或发布冻结需要创建。稳定工作文件不加日期，冻结版本按 deliveryPlan 的版本命名规则保存。
- `outputs/` 在存在计划或实际交付物时建立，具体结构和状态遵循 `cloudcc doc projectOutputs devguide`。构建缓存及一般临时输出不属于正式交付物。
- sidecar 内部按技术栈组织，平台资源按对应开发指南组织；本目录标准不另造一套通用框架目录。
- `.claw-local/`、`captures/`、缓存和系统文件属于本地内容；正式采用的证据进入 evidence，并更新引用。IDE 目录等额外入口按实际需要登记。
- 当前包另外识别 `.ai-dev/` 状态根别名；新项目采用 `.claw/`，不建立两套状态源。系统文件 `.DS_Store` 仅作为应忽略的本地文件识别，不是应创建的项目资产。

### 2.4 规范要求与当前工具行为

| 项目 | 当前行为与实施要求 |
|---|---|
| 目录树中的【必】 | 表示治理要求，不等于默认登记已设置机器门禁 |
| init project-layout | 只创建 docs/project-layout.json；默认登记的条目未设置 required:true，不创建完整目录树 |
| required:true | 对已登记的一级条目启用存在性检查；应依据项目采用的规范显式设置 |
| 嵌套治理文件 | 一级目录存在不证明内部文件齐全，须结合项目治理、交付及测试专用检查 |
| create project | 当前一次创建 frontend/pagecomponents、frontend/build、frontend/scripts、backend/classes、backend/triggers、backend/schedule、backend/lib 和 sidecar 开发骨架；尚不支持按能力选择生成 |
| 完整项目初始化 | 开发骨架不包含完整治理、设计和交付材料；应按项目治理流程补齐必要入口，不能把命令成功当作实施完成 |

目录登记的允许集合、required 存在性要求和实际创建时机是三个不同概念。先确定采用哪些目录及何时需要，再设置登记；默认允许某目录不代表必须创建它。目录检查不会读取自然语言标记并自动修改登记，也不会迁移现有文件。

## 3. 命令与返回

```bash
# 现状只读盘点，不要求配置、认证或网络
cloudcc doctor project-layout /path/to/project

# 仅在决定采用目录登记后执行；只新建 docs/project-layout.json
cloudcc init project-layout /path/to/project

# 人工按本项目规范编辑登记后重新检查
cloudcc doctor project-layout /path/to/project
```

省略 projectPath 使用当前目录。init 不覆盖已有登记，不自动认可未知目录，不生成其他能力目录，不移动文件，不修改 AGENTS 或 .gitignore。初始登记包含通用可选目录，必须按项目批准的白名单裁剪或扩展；init 成功不代表项目通过检查。

doctor 输出 `readOnly`、`registered`、`clientScripts`、一级 `entries`、`errors`、`warnings` 和计数。错误返回非零；无错误有警告为 `passed_with_warnings`，无错误警告为 `passed`。没有登记时会明确提示通用建议尚未成为项目白名单，不能宣称已通过项目自定义规范。

## 4. 登记格式

schemaVersion 固定 `cloudcc-project-layout/v2`。init 生成完整默认登记；下面是裁剪到小型项目的完整格式示例，按实际根目录继续添加条目：

```json
{
  "schemaVersion": "cloudcc-project-layout/v2",
  "referenceMaterials": "references",
  "rootEntries": [
    {"name":"README.md","kind":"file","role":"project-entry","tracking":"tracked","required":true},
    {"name":"AGENTS.md","kind":"file","role":"agent-rules","tracking":"tracked","required":true},
    {"name":".gitignore","kind":"file","role":"ignore-policy","tracking":"tracked","required":true},
    {"name":".claw","kind":"directory","role":"project-state","tracking":"tracked","required":true},
    {"name":"docs","kind":"directory","role":"design-and-governance","tracking":"tracked","required":true},
    {"name":"references","kind":"directory","role":"source-materials","tracking":"tracked","required":true},
    {"name":"frontend","kind":"directory","role":"frontend-code","tracking":"tracked"},
    {"name":"cloudcc-cli.config.json","kind":"file","role":"local-configuration","tracking":"local"}
  ]
}
```

字段规则：

| 字段 | 约定 |
|---|---|
| referenceMaterials | 项目相对路径，默认 references；必须位于已登记共享目录下 |
| rootEntries | 完整一级路径允许集合；默认条目不会隐式合并进手写登记 |
| name | 单个一级目录或文件名，禁止路径穿越、大小写重复；`.git` 保留给 Git |
| kind | `directory` 或 `file` |
| role | 非空职责说明 |
| tracking | `tracked` 表示允许共享，不要求每个文件已提交；`local` 必须未跟踪且被忽略 |
| required | 可选，true 时工作区必须存在此路径 |

参考资料使用其他入口时，先在项目规范和适用治理技能间解决约定冲突，再设置 referenceMaterials，并保留该目录的 README 来源索引。登记文件本身不授权绕过项目规则，不自动迁移资料。

## 5. 客户端脚本的唯一源码位置

客户端脚本属于前端高代码，统一采用 `frontend/scripts/<objectApi>/<name>/`。每个资源包含 `<name>.js` 和 `config.json`。创建、发布及 `scan msapi <projectPath> highcode` 均使用此位置。目录登记不提供源码路径选择参数；已采用登记的项目必须将 frontend 登记为共享目录。

```bash
cloudcc create script Account/BeforeSave /path/to/project
cloudcc publish script Account/BeforeSave /path/to/project
```

资源标识必须为 `<object>/<name>`。拒绝绝对路径、路径穿越及越出项目的符号链接；create 拒绝覆盖已有资源目录。publish 在读取连接信息和发送请求前校验源码与配置，缺少规范位置文件即失败。

省略项目路径时使用当前目录；显式项目路径同时用于源码和连接配置。远程查询命令的实际能力和参数见 `cloudcc doc platform/script devguide`。

## 6. 检查范围与治理方式

| 检查 | 结果 |
|---|---|
| 工作区、Git 跟踪和暂存一级路径 | 显式登记后未知路径报错；无登记时待分类警告 |
| 类型、必需路径、路径逃逸、注册格式 | 错误；不自动修复 |
| 本地配置、缓存、captures 等 | 被跟踪/暂存或未忽略时报错 |
| 固定交付文件名与参考资料索引 | 缺失/改名警告；不创建空壳让检查通过 |
| 双状态根、重复 JAR、执行输入重叠 | 提示核实事实源及消费者 |
| outputs 一级实体登记覆盖 | 未被工作源、制品或快照引用时提示分类；深层状态/哈希用 outputs doctor |
| 发布目录中的 JSON、日志、源码 | 提示复核是否应归 evidence；可能是历史证据，不自动移动 |
| 状态入口超过 100 KiB | 建议压缩入口和归档历史，不删除任务记录 |

Git 不可用或项目不是 Git 根时，明确报告未检查 Git 跟踪/忽略。linked worktree 的 Git 自有控制文件可用，不能把普通 `.git` 文件作为交付内容。检查不读取真实连接配置内容，不执行项目代码，不进行完整凭据或业务数据扫描。

多模块通常共享主项目治理，在模块 FEAT、设计和证据中按范围组织。独立交付项目应声明其边界；存在多个交付根时本检查给出范围警告。现有 `doctor test-governance` 仍要求一个已采用测试治理的交付根，本次不改变其契约，也不自动填补历史人工决定。

补充检查分别执行：

```bash
cloudcc doctor project-governance /path/to/project
cloudcc doctor project-outputs /path/to/project
cloudcc doctor test-governance /path/to/project
```

历史整理先列出用途、现行源、引用者、拟迁移位置和保留方式，再按项目授权执行。冻结证据通过稳定索引、版本和摘要保持可追溯；运行日志、截图、生产源码快照不因同名而覆盖开发源码。
