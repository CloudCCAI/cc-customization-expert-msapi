# CloudCC 定时器开发规范

## 1. 核心原则

定时器只做任务入口，复杂业务逻辑统一写入自定义类。

推荐职责划分：

- 定时器：初始化上下文、调用自定义类、记录必要日志
- 自定义类：承载业务规则、聚合其他类、负责编排逻辑

## 2. 开发规范

- 一个定时任务对应一个定时器类
- 创建定时器时，推荐使用自动创建自定义类模式：`cloudcc create timer <name> true`
- 发业务逻辑需要在创建的自定义类中实现
- 定时器类中实现编排自定义类的调用
- 定时任务需求也要先判断平台标准元数据能力是否可满足，不局限于某个低代码功能；对象/字段、验证规则、查重过滤器、工作流/审批、共享/权限、公式/汇总、自动编号、查找筛选、相关列表等能满足时，优先使用平台元数据，只有定时扫描、外部同步、历史数据治理、复杂批处理等低代码能力无法覆盖的场景才写定时类/自定义类
- 定时任务或其调用的自定义类生成普通流水号、单据号、客户号、合同号时，优先使用平台自动编号字段；不要用定时任务扫描已有数据后自行计算下一个编号
- 定时任务用于治理重复数据时，应先确认保存入口是否可以用 `dupeCatcher` 查重过滤器前置阻断；只有历史数据治理、跨对象复杂匹配或外部系统校验等查重过滤器无法覆盖的场景，才在定时类/自定义类里做代码查重
- 定时类或其调用的自定义类做查重、存在性判断、幂等判断、待处理数据扫描时，必须有状态、时间范围、业务键或分页边界；禁止用 `1=1` 全表查询后在 Java 中循环比对。`cquery*` 相关方法有平台返回条数上限，默认通常为 5000 且可配置，所以全表后循环不仅会拖慢定时任务，还可能漏处理或漏判 5000 条以后的数据
- 定时类或其调用的自定义类不得在 `for`/`while` 循环内调用 `cquery*`、`cqlQuery*`、`pagedQuery` 等查询方法逐条补字段、逐条查重或逐条判断存在性；定时任务默认按批量数据处理，必须先收集 key，批量查询并构建 Map，避免 N+1 查询和任务超时
- 定时类或其调用的自定义类使用 `cqlQuery` 扫描待处理数据或做联查时，默认必须排除逻辑删除数据；每个业务对象或别名都要拼接平台逻辑删除条件，常见写法是 `is_deleted = '0'`，若目标对象实际字段为 `is_delete` 或其他名称，必须按实际字段拼接；除非需求明确是回收站、删除审计或删除状态对比
- 定时类调用的自定义类不要把 `net.sf.json.JSONObject`/`JSONArray` 作为业务返回结构；优先返回 `Map<String,Object>`、`List<Map<String,Object>>` 或明确 DTO，避免 `JSONNull` 运行时问题和低效 JSON 转换
- 定时类调用外部 HTTP 服务时，必须通过接口注册器 `CCRemoteClient`，源码只使用已调试且为 `ACTIVE` 的 `apiCode`，不得硬编码 URL 或误用 `apiName`。任务必须分页并保留业务游标/幂等键，区分 HTTP 成功、远端业务失败和本地处理失败；重试必须有次数上限。完整规则见 `platform/apiRegistrar devguide`

## 3. 文件约束

- 定时器目录位于 `schedule/<类名>/`
- 业务代码只写在 `SOURCE_CONTENT` 区域内
- 不要手工创建目录或修改 `config.json` 关键内容

## 4. 可直接使用的上下文

- `userInfo`

## 5. 推荐写法

定时器中只保留入口逻辑：

```java
UserInfo userInfo = new UserInfo();
// @SOURCE_CONTENT_START
MyTimerService service = new MyTimerService(userInfo);
service.execute();
// @SOURCE_CONTENT_END
```

自定义类中承载业务实现：

```java
// @SOURCE_CONTENT_START
public class MyTimerService{
    private UserInfo userInfo;
    private CCService cs;

    public MyTimerService(UserInfo userInfo){
        this(userInfo,new CCService(userInfo));
    }
    public MyTimerService(UserInfo userInfo,CCService cs){
        this.userInfo = userInfo;
        this.cs=cs;
    }
    
    public void execute(String str){
       // 在这里编写业务逻辑
    }
}
// @SOURCE_CONTENT_END
```

## 6. 结论

定时器的定位是定时执行入口，不是业务实现容器。

最佳实践是：

- 定时器保持精简
- 业务逻辑写入自定义类
- 复杂流程通过自定义类继续拆分和编排
- 定时器源码和它调用的自定义类都必须遵守单个 Java 文件低于 2000 行的限制；复杂定时任务应拆分为多个自定义类

定时任务通常会处理批量数据，因此查询必须显式收敛：

- 查重/存在性/幂等判断：用业务键条件 + `pagedQuery(..., "1", "1", ..., "id")` 或带条件的 `cqueryByFields`
- 扫描待处理数据：必须带状态、时间窗口、分页参数或批次上限
- 禁止为了判断是否存在、是否重复或计算下一编号而 `cqueryByFields("Object", "1=1", ...)` 后循环遍历；`cquery*` 默认通常最多返回 5000 条且可配置，不能当作全量数据读取
- 禁止循环内查询：待同步数据、明细数据或关联对象补齐必须先收集 key，使用 `IN` 条件或分批查询取回必要字段，再构建 `Map` 做内存匹配
- 使用 `cqlQuery` 时：状态/时间窗口/批次边界和逻辑删除条件必须同时下推到 CQL，例如 `WHERE is_deleted = '0' AND sync_status__c = '待同步'`

正确示例：

```java
String expression = "sync_status__c = '待同步' and lastmodifydate >= '2026-08-01 00:00:00'";
List rows = cs.pagedQuery("Account", expression, "1", "200", "false", "id,name,tyshxydm");
```

## 7. 文件与交付规范

这是 AI 必须遵守的第一层约束。

### 7.1 目录与文件来源

- 定时器目录必须位于 `schedule/<类名>/`
- 每个定时器目录包含主类 `*.java` 和 `config.json`
- 定时器目录必须通过 `cloudcc create timer <name>` 创建

### 7.2 AI 不得做的事情

- 不要手工新建 `schedule/` 子目录
- 不要整包复制其他项目的定时器目录
- 不要手工修改 `config.json` 中的 `id` 或版本字段
- 不要在 `// @SOURCE_CONTENT_START` 与 `// @SOURCE_CONTENT_END` 之外写业务逻辑
- 不要把复杂批处理、同步、生成、通知逻辑全部塞进一个定时类或一个配套自定义类；单个 Java 文件不得超过 2000 行

### 7.3 AI 允许做的事情

- 只在 CLI 生成的主类 SOURCE 区域内实现业务逻辑
- 保持 `package schedule.<类名>;` 与目录名一致
- 构造方法名与类名一致
- 当需求复杂或预计超过 1500 行时，先设计多个自定义类分工，再由定时类入口编排调用

### 7.4 timer 模块支持的 CLI 命令总览（重点：入参）

说明：

- 下文命令中的资源名使用 `timer`，也可使用别名
  `schedule`（两者路由到同一模块）。
- `projectPath` 未传时，默认使用当前工作目录。

#### 1) 创建定时类

命令：

```bash
cloudcc create timer <name> [autoCreateClass]
```

参数：

| 参数   | 必填 | 类型     | 说明                                    |
| ------ | ---- | -------- | --------------------------------------- |
| `name` | 是   | `string` | 定时类名称,必须使用英文（同时作为目录名、Java 类名） |
| `autoCreateClass` | 否 | `boolean` | 是否自动创建配套自定义类；默认 `false`，传 `true` 开启推荐自动创建模式 |

示例：

```bash
cloudcc create timer DailySyncJob

# 推荐：自动创建配套自定义类
cloudcc create timer DailySyncJob true
```

---

#### 2) 发布定时类

命令：

```bash
cloudcc publish timer <name> [projectPath]
```

参数：

| 参数   | 必填 | 类型     | 说明                           |
| ------ | ---- | -------- | ------------------------------ |
| `name` | 是   | `string` | 本地 `schedule/<name>/` 目录名 |
| `projectPath` | 否 | `string` | 项目根目录；未传时使用当前工作目录 |

示例：

```bash
cloudcc publish timer DailySyncJob <projectPath>
```

发布顺序固定为：

1. 远程 validate，调用 `POST /api/ccPeak/validate`。
2. 保存，调用 `POST /api/ccPeak/save`。

从 CLI/技能 `2.2.7` 开始，`publish timer` 要求目标 setup-svc 至少为 `19.3.R20`，因为旧版本 setup-svc 不提供 `/api/ccPeak/validate`。

`/api/ccPeak/validate` 的实际入参类型也是 `CCfagVo`，服务端实际读取并编译的是 `source`，编译入口固定为 `PeakFunctionImpl`。CLI 发送的 `id`、`name`、`version`、`folderId` 是为了复用后续 save payload 和保留上下文；这些字段不参与 validate 编译判断。`source` 在远程 validate 和 save 中都会使用 URLDecoder-compatible 编码；源码中的字面量 `+` 会编码为 `%2B`。远程 validate 失败时，CLI 必须返回 setup-svc 的 `returnInfo`、`data.errors`、`data.warnings` 和原始 `responseBody`，并且不能继续 save。

---

#### 3) 拉取定时类（按本地名称）

命令：

```bash
cloudcc pull timer <name>
```

参数：

| 参数   | 必填 | 类型     | 说明                                                                            |
| ------ | ---- | -------- | ------------------------------------------------------------------------------- |
| `name` | 是   | `string` | 本地 `schedule/<name>/` 目录名；会读取该目录 `config.json` 中的 `id` 到线上拉取 |

示例：

```bash
cloudcc pull timer DailySyncJob
```

---

#### 4) 查询定时类列表（支持条件查询）

命令：

```bash
cloudcc get timer [listQueryJson] [projectPath]
```

`listQueryJson` 推荐结构：

```json
{
    "shownum": 2000,
    "showpage": 1,
    "sname": ""
}
```

字段说明：

| 字段       | 类型     | 必填    | 说明                     |
| ---------- | -------- | ------- | ------------------------ |
| `shownum`  | `number  | string` | 否                       |
| `showpage` | `number  | string` | 否                       |
| `sname`    | `string` | 否      | 名称筛选关键字，模糊查询 |

示例：

```bash
cloudcc get timer '{"shownum":2000,"showpage":1,"fid":"","sname":"Daily","rptcond":"","rptorder":""}'
```

---

#### 5) 查看定时类详情

命令：

```bash
cloudcc detail timer <name> <id>
```

参数规则（实现口径）：

| 参数   | 必填     | 类型     | 说明                                         |
| ------ | -------- | -------- | -------------------------------------------- |
| `name` | 条件必填 | `string` | 传 `name` 时优先查本地；本地不完整时再走线上 |
| `id`   | 条件必填 | `string` | 当 `name` 为空时，必须传 `id` 走线上查询     |

等价理解：`name` 与 `id` 至少传一个，优先使用 `name` 路径。

示例（按本地名）：

```bash
cloudcc detail timer DailySyncJob
```

示例（按线上 id）：

```bash
cloudcc detail timer "" a0Bxxxxxxxxxxxx
```

---

#### 6) 按 ID 拉取并落地到本地目录

命令：

```bash
cloudcc pullList timer <id> <projectPath>
```

参数：

| 参数          | 必填 | 类型     | 说明                                                  |
| ------------- | ---- | -------- | ----------------------------------------------------- |
| `id`          | 是   | `string` | 线上定时类 ID                                         |
| `projectPath` | 是   | `string` | 项目根目录；会写入到 `<projectPath>/schedule/<name>/` |

示例：

```bash
cloudcc pullList timer a0Bxxxxxxxxxxxx /path/to/project
```

---

#### 7) 删除定时类

命令：

```bash
cloudcc delete timer <nameOrId> [projectPath]
```

参数规则：

| 参数          | 必填 | 类型     | 说明                                                                                                    |
| ------------- | ---- | -------- | ------------------------------------------------------------------------------------------------------- |
| `nameOrId`    | 是   | `string` | 可传本地目录名或线上 ID；若本地 `schedule/<nameOrId>/config.json` 存在且带 `id`，优先使用其中 `id` 删除 |
| `projectPath` | 否   | `string` | 项目根目录，默认当前目录                                                                                |

示例（按名称）：

```bash
cloudcc delete timer DailySyncJob
```

示例（按 id）：

```bash
cloudcc delete timer a0Bxxxxxxxxxxxx
```

---

#### 8) 文档命令

命令：

```bash
cloudcc doc platform/timer <introduction|devguide>
```

参数：

| 参数          | 必填      | 类型 | 说明     |
| ------------- | --------- | ---- | -------- |
| `introduction | devguide` | 是   | `string` |

示例：

```bash
cloudcc doc platform/timer introduction
cloudcc doc platform/timer devguide
```
