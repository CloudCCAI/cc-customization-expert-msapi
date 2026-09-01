# 币种管理 CLI 用户级与开发说明

币种管理属于公司信息低代码元数据能力。MSAPI/Universal 选择 MetadataService 时，`currency`、`currencies`、`companyCurrency`、`companyCurrencySet`、`currencyManage` 都映射到 canonical domain `currencies`。

## 读取

```bash
cloudcc get currency <projectPath> [filter]
cloudcc detail currency <projectPath> <currencyCode-or-id>
cloudcc newInfo currency <projectPath>
cloudcc get currency <projectPath> '{"selectedDate":"2026-01-01"}'
```

读取调用 MetadataService 只读接口：

- `cloudcc get currency .` -> `GET /metadata/v1/currencies`
- `cloudcc get currency . USD` -> `GET /metadata/v1/currencies/USD`
- `cloudcc detail currency . USD` -> `GET /metadata/v1/currencies/USD`
- `cloudcc newInfo currency .` -> `GET /metadata/v1/currencies:available`
- `cloudcc get currency . '{"selectedDate":"2026-01-01"}'` -> `GET /metadata/v1/currencies/dated-rates?selectedDate=2026-01-01`

列表返回公司本位币、启用币种、停用币种、高级多币种开关和原始 `tp_sys_currency_setup` 行。详情返回固定币种记录、`datedRates[]`、`relatedRows.tp_sys_currency`、`relatedRows.tp_sys_currency_adv`，便于和 setup-web 页面读数做差异验收。

## 新增币种

简写：

```bash
cloudcc create currency <projectPath> <currencyCode> <rate> <decimalDigits>
cloudcc apply msapi <projectPath> <planId>
```

示例：

```bash
cloudcc create currency . USD 7.200000 2
cloudcc apply msapi . <planId>
```

JSON：

```bash
cloudcc plan msapi . currencies '{"currencyCode":"USD","rate":"7.200000","decimalDigits":2}' create
cloudcc apply msapi . <planId>
```

新增计划会写入：

- `tp_sys_currency`：固定币种、固定汇率、启用状态、小数位数。
- `tp_sys_currency_adv`：默认高级汇率行，`is_default=1`。

plan metadata 标记 setup-service 来源 `/api/currency/saveNewCurrencyPage`。

## 修改币种和固定汇率

修改小数位：

```bash
cloudcc update currency <projectPath> <currencyCode-or-id> <decimalDigits>
cloudcc plan msapi <projectPath> currencies '{"currencyCode":"USD","decimalDigits":2}' update
```

修改固定汇率：

```bash
cloudcc updateRate currency <projectPath> <currencyCode-or-id> <rate>
cloudcc plan msapi <projectPath> currencies '{"rates":[{"currencyCode":"USD","rate":"7.200000"}]}' update-rates
```

固定汇率计划对齐 `/api/currency/saveCurrencyRate`。公司本位币固定汇率必须保持 `1`，plan/apply 会在目标表可读时阻断非法计划。

## 启用和停用币种

```bash
cloudcc activate currency <projectPath> USD
cloudcc deactivate currency <projectPath> USD CNY
cloudcc apply msapi <projectPath> <planId>
```

停用币种对齐 setup-service `/api/currency/enableCurrencyByCode` 语义。若传入 `fallbackCurrencyCode`，计划会把使用被停用币种的 `tp_sys_user.currency` 改回公司本位币；未传时只停用币种，并在 plan warnings 中提醒调用方自行处理用户币种。

保护规则：

- 不能停用公司本位币。
- 停用非本位币时建议传 `fallbackCurrencyCode`，通常是当前公司本位币。
- 删除固定币种记录不作为常规 CLI 快捷命令暴露；页面语义以启用/停用为主。

## 高级多币种

启用或停用高级多币种：

```bash
cloudcc enableAdvanced currency <projectPath>
cloudcc disableAdvanced currency <projectPath>
```

JSON 可显式传入设置行和默认高级汇率：

```bash
cloudcc plan msapi . currencies '{"setupId":"curset-default","enableAdvanceCurrency":"1","rates":[{"currencyCode":"CNY","rate":"1"},{"currencyCode":"USD","rate":"7.2"}]}' enable-advanced
cloudcc apply msapi . <planId>
```

启用计划写入 `tp_sys_currency_setup.enable_advance_currency=1`，并可按 `rates[]` 补齐 `tp_sys_currency_adv` 默认汇率行。停用计划只修改开关并给出 warnings：业务机会、业务机会产品及相关报表将改用固定汇率口径。

## 高级汇率 dated rates

新增生效日期汇率：

```bash
cloudcc createDatedRate currency <projectPath> <beginDate> <currencyCode> <rate>
cloudcc apply msapi <projectPath> <planId>
```

示例：

```bash
cloudcc createDatedRate currency . 2026-01-01 USD 7.300000
```

JSON：

```bash
cloudcc plan msapi . currencies '{"beginDate":"2026-01-01","rates":[{"currencyCode":"USD","rate":"7.300000"},{"currencyCode":"EUR","rate":"8.100000"}]}' create-dated-rate
cloudcc plan msapi . currencies '{"id":"curadv-usd-20260101","rate":"7.250000"}' update-dated-rate
cloudcc plan msapi . currencies '{"id":"curadv-usd-20260101"}' delete-dated-rate
```

保护规则：

- 同一 `beginDate` 不允许重复新增 dated rate。
- `is_default=1` 的默认高级汇率行不能删除。
- `beginDate` 必须使用 `yyyy-MM-dd`。

setup-service 对照：

| CLI/MetadataService | setup-service |
|---------------------|---------------|
| dated rate list by selectedDate | `/api/currency/queryCurrencyListBySelectDate` |
| new dated rate page | `/api/currency/newCurrencyRate` |
| create dated rate | `/api/currency/saveNewCurrencyRate` |
| dated rate detail | `/api/currency/currencyDetail` |
| edit dated rate page | `/api/currency/editCurrency`、`/api/currency/currencyEditPage` |
| update dated rate | `/api/currency/currencyEditSave` |
| delete dated rate | `/api/currency/currencyDelete` |

## 变更公司本位币

公司本位币变更是高风险操作。setup-service 的 `/api/currency/saveCorpCurrency` 会基于旧汇率动态重算固定汇率和高级汇率；MetadataService 为保持静态 plan、可审计和可 rollback，不在服务端生成动态 SQL 表达式。

必须传显式重算后的 `rates[]`，必要时再传 `datedRates[]`：

```bash
cloudcc plan msapi . currencies '{
  "currencyCode": "USD",
  "rates": [
    {"currencyCode": "CNY", "rate": "0.138889"},
    {"currencyCode": "USD", "rate": "1"},
    {"currencyCode": "EUR", "rate": "1.125000"}
  ],
  "datedRates": [
    {"id": "curadv-cny-20260101", "rate": "0.138889"},
    {"id": "curadv-usd-20260101", "rate": "1"}
  ]
}' change-corporate
cloudcc apply msapi . <planId>
```

若只执行：

```bash
cloudcc changeCorporate currency . USD
```

MetadataService 会返回 `currency_rebase_rates_required`，提醒调用方先从 setup-web/setup-service 读出并确认目标汇率，再生成可复核 plan。

## setup-service 对照

| CLI/MetadataService | setup-service |
|---------------------|---------------|
| list | `/api/currency/queryCurrencyList`、`/api/currency/currencyList` |
| available currencies | `/api/currency/newCurrencyPage` |
| create currency | `/api/currency/saveNewCurrencyPage` |
| activate/deactivate | `/api/currency/enableCurrencyByCode` |
| change fixed rate page | `/api/currency/changeCurrencyRate` |
| save fixed rates | `/api/currency/saveCurrencyRate` |
| change corporate page | `/api/currency/changeCorpCurrency` |
| save corporate currency | `/api/currency/saveCorpCurrency` |
| advanced currency toggle | `/api/currency/currencyAdvEnableOrUnEnable` |

## 推荐验收

1. `cloudcc get currency <projectPath>` 确认公司本位币、启用币种、停用币种和高级多币种开关。
2. `cloudcc newInfo currency <projectPath>` 确认只返回尚未启用的标准币种。
3. `cloudcc create currency ...` 后先审 plan，再 `apply msapi`。
4. `cloudcc detail currency <projectPath> <currencyCode>` 回读 `tp_sys_currency` 和 `tp_sys_currency_adv`。
5. 对停用、删除 dated rate、变更公司本位币类高风险计划，必须检查 plan warnings、target rows 和 rollback-plan。

