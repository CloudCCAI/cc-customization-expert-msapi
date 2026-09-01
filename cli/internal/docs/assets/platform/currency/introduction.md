# CloudCC 币种管理

币种管理是公司信息下的低代码元数据能力，对应 setup-web `/settings/companyInformation/companyCurrency` 和 `/settings/companyInformation/companyCurrencySet`。

MSAPI/Universal 选择 MetadataService 时，`currency`、`currencies`、`companyCurrency`、`companyCurrencySet` 都映射到 canonical domain `currencies`。

- 读：`GET /metadata/v1/currencies`、`GET /metadata/v1/currencies/{currencyCode-or-id}`、`GET /metadata/v1/currencies:available`、`GET /metadata/v1/currencies/dated-rates`
- 写：通过 MetadataService `currencies` domain 生成 plan，再显式 apply
- 范围：固定币种、公司本位币标记、启用/停用、固定汇率、高级多币种开关和 dated exchange rates
- 高风险：变更公司本位币必须显式提供重算后的汇率清单，MetadataService 不生成运行时动态 SQL

