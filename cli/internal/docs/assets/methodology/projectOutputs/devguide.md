# CloudCC 项目产出物治理指南

## 1. 定位

这里的 `outputs/` 指项目仓库根目录下的 `<project-root>/outputs/`，不是操作系统根路径 `/outputs`，也不是普通程序运行产生的临时 output 目录。

```text
<project-root>/
├── docs/delivery/<project-code>/   # 实施设计、矩阵、计划、测试和上线治理
├── test-assets/                    # 可复用测试场景、数据模板、断言和自动化
├── evidence/                       # 真实决策、运行、日志、截图、读回和批准证据
└── outputs/                        # 客户或下游团队实际接收的项目产出物
```

`dist/`、`build/`、`tmp/`、缓存和普通程序生成结果不属于项目 outputs。不要把整个 `outputs/` 加入 `.gitignore`。

## 2. 固定骨架与动态内容

执行：

```bash
cloudcc init project-outputs <projectPath> <projectCode>
```

初始化器只创建：

```text
outputs/
├── README.md
├── 00-output-index.md
└── output-manifest.json
```

它不预建 documents、tools、operations、migration、training 等目录。项目确认需要某项产出后，使用稳定 `outputId` 动态创建，例如：

```text
outputs/
├── solution-presentation/
├── acceptance-report/
├── user-manual/
├── operations-tool-health-check/
└── migration-tool-customer-import/
```

目录名称不是模板要求，只需与 manifest 的稳定 outputId 和实际交付边界一致。

## 3. 支持的产出类型

| kind | 用途 |
|---|---|
| document | 方案汇报、验收报告、用户/管理员/运维手册 |
| tool | 运维、迁移、诊断、校验等项目专用工具 |
| data-package | 批准脱敏的数据模板、初始化包、转换结果 |
| deployment-package | 部署脚本、配置样例、安装包 |
| training-package | 培训课件、讲义、演练材料 |
| integration-package | SDK、接口样例、联调工具 |
| other | 合同或项目要求的其他产出物 |

项目没有某类产出物时不创建空目录。

## 4. Manifest

`outputs/output-manifest.json` 是机器事实源：

```json
{
  "schemaVersion": "cloudcc-project-outputs/v1",
  "projectCode": "demo-crm",
  "outputs": [
    {
      "outputId": "migration-tool-customer-import",
      "kind": "tool",
      "title": "Customer Import Migration Tool",
      "status": "delivered",
      "ownerRole": "integration-agent",
      "requirementSource": "project delivery plan",
      "audience": ["customer-operations"],
      "formats": ["zip"],
      "workingPaths": [
        "outputs/migration-tool-customer-import/source"
      ],
      "releaseArtifacts": [
        {
          "path": "outputs/migration-tool-customer-import/release/migration-tool.zip",
          "sha256": "<64-hex-sha256>"
        }
      ],
      "snapshotPaths": [],
      "evidenceRefs": [
        "evidence/migration/runs/MIGRATION-001/manifest.json"
      ],
      "externalRefs": []
    }
  ]
}
```

状态支持 planned、draft、review、approved、delivered、retired。approved/delivered 必须至少存在本地 release artifact 或 externalRef；本地冻结包必须记录并通过 SHA-256 读回。

所有本地路径必须是项目相对路径，禁止绝对路径、`..` 穿越以及解析后指向项目外部的符号链接。大型文件可以存入受控制品库，在 externalRefs 中记录稳定地址，并在相应交付说明中记录版本和摘要。

## 5. 人类索引

`00-output-index.md` 是人类和 Agent 的入口，可以按项目需要扩展列，但不复制正文或工具源码。至少记录：

- outputId、kind、名称和需求来源。
- 受众、负责人、格式和状态。
- 当前工作源、冻结交付件或在线地址。
- 批准人、验收状态和证据引用。

机器事实以 manifest 为准；发现不一致时先修复 manifest，再同步 Markdown 索引。

## 6. 文档类交付物

- 可编辑工作源放在产出物目录的 `source/` 或 manifest 指向的其他唯一项目路径。
- PDF、PPTX、DOCX、HTML 等实际交付文件放在 `release/` 或受控制品库。
- 客户评审、签字、验收或冻结版本使用 `snapshots/<YYYYMMDD>-<artifact-key>-v<major>.<minor>-<status>.<ext>`。
- 在线文档应同时保留稳定的 Agent 可读 Markdown 入口或摘要、版本和访问边界。
- 验收报告引用 `07-testing-cutover` 和 `evidence/testing`，不复制原始日志、截图和读回。

## 7. 工具类交付物

项目专用运维、迁移、验证或集成工具至少说明：

- 用途、范围、输入、输出和依赖。
- 安装、执行、dry-run 或预检查方式。
- 幂等、失败处理、重试、备份、回滚和执行后读回。
- 测试方式、已验证环境、版本和 SHA-256。
- 配置样例与真实配置边界。

项目专用一次性工具可以把源码和测试保存在其 outputs 目录。需要跨项目复用的通用工具应迁移到独立 tools 模块或仓库；项目 outputs 只保存所采用的冻结版本、配置样例和使用说明。

数据迁移工具不得打包真实客户数据、数据库连接口令或访问 Token。实际执行日志和结果属于 evidence，不属于工具源码。

## 8. Git、制品库和安全

- README、索引、manifest、工具源码、测试和配置样例应进入版本控制。
- 不应把整个 outputs 忽略；只能按项目策略忽略明确的大型生成文件，并通过 manifest 保留制品位置、版本和摘要。
- 禁止 `.env`、`.pem`、`.key`、credential 文件和 JSON 非空 password/token/secret/privateKey/openSecretKey。
- 禁止未脱敏客户数据、真实账号、环境密钥和一次性运行日志进入通用技能或项目输出源。
- 交付前对冻结包执行病毒/恶意代码扫描、依赖检查、许可证检查和人工安全复核；CLI doctor 只覆盖结构、引用、哈希和明显敏感内容风险。

## 9. 与测试、发布证据的关系

- `evidence/testing` 保存实际执行 manifest、日志、截图和读回。
- `outputs/acceptance-report` 汇总并引用测试/UAT 证据，但不替代业务负责人签字。
- `08-release-evidence` 引用本次采用的 outputs 路径、版本和 SHA-256，不复制整个 outputs。
- 技术测试通过不能把 document/tool 自动标记为 approved 或 delivered；正式状态由项目责任人确认。

## 10. 只读诊断

```bash
cloudcc doctor project-outputs <projectPath>
```

诊断检查：

- 三个固定治理文件。
- schemaVersion、projectCode、outputId 唯一性、kind 和 status。
- 项目相对路径、路径穿越和符号链接逃逸风险。
- 本地引用是否存在。
- approved/delivered 交付件和 SHA-256。
- `.env`、私钥、credential 文件和 JSON secret 字段。

doctor 不创建、修改、批准或删除任何项目产出物。
