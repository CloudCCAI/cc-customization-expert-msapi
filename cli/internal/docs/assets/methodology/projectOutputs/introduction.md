# CloudCC 项目产出物治理

项目根目录 `outputs/` 是正式交付和移交边界，不局限于文档，也可以包含项目专用运维工具、数据迁移工具、脱敏数据包、部署包、培训材料和接口联调包。

治理原则是“固定容器和登记规则，不固定内容模板”：

- 固定 `README.md`、`00-output-index.md` 和 `output-manifest.json`。
- 具体产出物由合同、客户或项目要求决定，确认后才创建目录。
- `docs/delivery` 保存实施过程，`test-assets` 保存测试资产，`evidence` 保存真实执行证据，`outputs` 保存实际交付件。
- approved/delivered 本地交付件必须具有 SHA-256；在线或制品库交付需要稳定外部引用。
- 人类评审、验收和签字仍是正式批准事实，AI 生成或技术验证不能替代。

```bash
cloudcc doc methodology/projectOutputs devguide
cloudcc init project-outputs <projectPath> <projectCode>
cloudcc doctor project-outputs <projectPath>
```
