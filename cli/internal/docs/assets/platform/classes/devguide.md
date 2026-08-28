# CloudCC 自定义类 AI 开发规范

## 1. 目的

本文档用于指导 AI 在 CloudCC
项目中编写自定义类时，如何做出稳定、可维护、可发布的实现。

适用范围：

- 新建自定义类
- 修改现有自定义类
- 为按钮、触发器、定时类、自定义页面、组件提供后端能力
- 编写通用服务类、工具类、集成类、流程编排类

本文档面向 AI，因此重点强调：

- 什么时候应该使用自定义类
- AI 写代码时必须遵守哪些硬规则
- SDK 该怎么用
- 代码结构应该怎么组织
- 哪些做法必须避免
- 修改类的时候，要将测试类也一起调整

### 外部 HTTP 调用门禁

自定义类需要主动调用外部 HTTP 服务时，优先使用平台接口注册器和 `CCRemoteClient`，并先读取 `platform/apiRegistrar devguide`。源码只使用注册项的 `apiCode`，不得硬编码远程 URL，也不得误用软件包标识 `apiName`。生成代码前必须确认注册项调试成功且为 `ACTIVE`；调用后必须检查 `CCRemoteResult` 的执行结果、HTTP 状态和错误信息，按需记录 `traceId`。接口修改会回到 `DRAFT`，必须重新调试后再投入运行。

## 2. 自包含编译与发布链路

CloudCC 自定义类不是普通 Maven 项目。技能已经把平台编译所需的 `cceg.jar`、cclib 和主要第三方依赖固化到 `assets/class-compiler/WEB-INF`，开发者不需要也不应获取 `setup-svc`、`main-svc` 或 CloudCC 平台源码。

`cloudcc validate classes` 的本地链路是：

1. 读取 SOURCE 标记内的 Java 源码。
2. 用内置 `com.cloudcc.core.cls.fag.FagTemplate` 生成 `com.cloudcc.core.fag.CLOCAL` 包装源码。
3. 用 JDK 21 和技能内置 classpath 编译；语法、SDK 类型、构造函数和策略错误会返回给调用方。

`cloudcc publish classes` 的发布链路是：

1. 先执行同 `cloudcc validate classes` 一致的本地 `FagTemplate` 编译验证。
2. 本地验证通过后，调用目标 setup-svc 的 `POST /api/ccfag/validate`，让目标环境按真实租户依赖校验待保存源码。
3. 远程 validate 通过后，才调用 class save/detail。
4. 保存后按 ID 读回源码并比较 SHA-256；一致才报告 `published_and_verified`。

目标平台内部如何路由到其部署服务不属于开发者依赖。技能不会读取服务源码、启动本地服务、探测本地端口或要求 main-svc binding。

## 3. 独立开发与发布环境

### 3.1 必需依赖

| 依赖 | 要求 | 来源 |
| --- | --- | --- |
| JDK | JDK 21，`java` 与 `javac` 均可用 | `JAVA_HOME`、`CLOUDCC_JAVA_HOME` 或 `classDev.javaHome` |
| 自包含编译器 | 30 个经过 SHA-256 清单校验的 `cceg.jar`、cclib 与编译依赖 | 技能包 `assets/class-compiler` |
| 发布入口 | 目标 CloudCC 公共网关 | `CloudCCDev`/`baseUrl` 自动解析，或 `classPublishUrl` |
| 发布认证 | 目标环境有效凭据 | 常规项目配置，或短期 `CLOUDCC_ACCESS_TOKEN` |

不需要平台源码、本机 Maven 仓库、Node/npm、setup-svc/main-svc 进程、runtime URL 或 binding。

### 3.2 项目配置

在 `cloudcc-cli.config.json` 当前环境中配置：

```json
{
  "use": "dev",
  "dev": {
    "safetyMark": "<mark>",
    "CloudCCDev": "<CloudCC developer configuration>",
    "classDev": {
      "javaHome": "/absolute/path/to/jdk-21"
    }
  }
}
```

认证信息优先通过进程环境变量注入，不要提交到项目：

```bash
export CLOUDCC_ACCESS_TOKEN='<temporary-token>'
```

私有部署如果标准配置不能推导目标网关，可在当前环境增加：

```json
"classPublishUrl": "https://example.cloudcc.cn/ccdomaingateway/setup"
```

### 3.3 初始化与环境诊断

```bash
cloudcc doctor classes <projectPath>
```

`doctor classes` 校验 JDK 21、内置 manifest、30 个 jar 的大小和 SHA-256，以及目标网关/认证配置。输出中的 `platformSourceRequired`、`setupSvcRequired`、`mainSvcRequired` 和 `mavenCacheRequired` 必须全部为 `false`。兼容命令 `prepare classes` 只做相同的只读校验，不再创建软链接或服务运行目录。

### 3.4 一次完成的标准工作流

```bash
cloudcc create classes OrderService
# 仅编辑 backend/classes/OrderService/OrderService.java 的 SOURCE 区域

cloudcc validate classes OrderService <projectPath>
cloudcc publish classes OrderService <projectPath>
```

- `validate` 使用技能内置的真实 `FagTemplate` 和依赖集编译，并把生成文件行号映射回 SOURCE 行号。
- `publish` 会先本地验证当前源码，再调用目标 setup-svc `POST /api/ccfag/validate`，最后才 save；若使用 `--validation-evidence`，证据中的类名、通过状态和 SHA-256 必须与当前源码一致。
- `publish` 失败时必须把本地编译诊断、远程 validate 的 `returnInfo`、`data.errors`、`data.warnings` 和原始 `responseBody` 返回给调用方，调用方应优先根据这些信息修正代码。
- 本地没有 ID 时，`publish` 先按名称精确查找已有类，以便从中断恢复并避免重复创建；歧义时 fail closed。
- 保存成功后按平台返回的 ID 调用 detail，比较读回源码 SHA-256，并把 ID 写入本地 `config.json`。

此技能的交付边界是本地编译与目标环境发布/读回，不以本地 main-svc 运行测试为前置条件。业务功能验收应在目标租户的正常用户入口、集成测试或客户验收流程中完成。

### 3.5 失败恢复原则

- 编译失败：修正诊断对应的 SOURCE 行，禁止靠重复发布试错。
- 保存响应丢失或 CLI 中断：重新执行 `publish`，它会先解析同名线上类 ID 后幂等更新。
- 读回哈希不一致：停止运行测试，确认服务端保存内容与本地源码，不继续盲目重发。
- 编译器完整性失败：重新安装对应版本技能包，禁止从本机服务目录或 Maven 缓存临时拼依赖。
- 业务验收失败：类已经上线但目标未完成，应继续修正业务实现并重新 validate/publish，不能把“发布成功”当成需求完成。

## 4. classes 模块支持的 CLI 命令总览

说明：

- 下文命令中的资源名使用 `classes`。
- `projectPath` 未传时，默认使用当前工作目录。

#### 1) 创建自定义类

命令：

```bash
cloudcc create classes <name>
```

参数：

| 参数   | 必填 | 类型     | 说明                                                  |
| ------ | ---- | -------- | ----------------------------------------------------- |
| `name` | 是   | `string` | 类名（同时作为目录名、Java 主类名），使用驼峰命名规则 |

#### 2) 发布自定义类

命令：

```bash
cloudcc publish classes <name> [projectPath] [--validation-evidence <file>]
```

参数：

| 参数   | 必填 | 类型     | 说明                          |
| ------ | ---- | -------- | ----------------------------- |
| `name` | 是   | `string` | 本地 `backend/classes/<name>/` 目录名 |
| `projectPath` | 否 | `string` | 项目根目录；未传时使用当前工作目录 |
| `--validation-evidence` | 否 | `string` | 复用同一源码 SHA-256 的本地验证证据；只跳过本地编译，不跳过远程 validate |

发布顺序固定为：

1. 本地编译验证，使用技能包内置 `FagTemplate` 和 classpath。
2. 远程 validate，调用 `POST /api/ccfag/validate`。
3. 保存，调用 `POST /api/ccfag/save`。
4. 按 ID 读回并校验源码哈希。

从 CLI/技能 `2.2.7` 开始，`publish classes` 要求 CLI 本地编译能力和目标 setup-svc validate 能力，目标 setup-svc 建议至少为 `19.3.R20`，因为旧版本 setup-svc 不提供 `/api/ccfag/validate`。高代码发布本身不要求 MetadataService 版本；setup-svc 分支版本只做初始化/doctor 提醒，不按字符串直接阻断。

`/api/ccfag/validate` 的实际入参类型是 `CCfagVo`，服务端实际读取并编译的是 `source`。CLI 发送的 `id`、`name`、`version`、`folderId` 是为了复用后续 save payload 和保留上下文；这些字段不参与 validate 编译判断。`source` 在远程 validate 和 save 中都会使用 URLDecoder-compatible 编码；源码中的字面量 `+` 会编码为 `%2B`，避免 Java URLDecoder 把加号解成空格。

#### 3) 拉取自定义类（按本地名称）

命令：

```bash
cloudcc pull classes <name>
```

参数：

| 参数   | 必填 | 类型     | 说明                                                                     |
| ------ | ---- | -------- | ------------------------------------------------------------------------ |
| `name` | 是   | `string` | 本地 `backend/classes/<name>/` 目录名；会读取该目录 `config.json` 中的 `id` 拉取 |

#### 4) 查询自定义类列表（支持条件查询）

命令：

```bash
cloudcc get classes [listQueryJson] [projectPath]
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

| 字段       | 类型     | 必填    | 说明                         |
| ---------- | -------- | ------- | ---------------------------- |
| `shownum`  | `number  | string` | 否                           |
| `showpage` | `number  | string` | 否                           |
| `sname`    | `string` | 否      | 名称筛选关键字，支持模糊查询 |

#### 5) 查看自定义类详情

命令：

```bash
cloudcc detail classes <name> <id>
```

参数规则（实现口径）：

| 参数   | 必填     | 类型     | 说明                                         |
| ------ | -------- | -------- | -------------------------------------------- |
| `name` | 条件必填 | `string` | 传 `name` 时优先查本地；本地不完整时再走线上 |
| `id`   | 条件必填 | `string` | 当 `name` 为空时，必须传 `id` 走线上查询     |

等价理解：`name` 与 `id` 至少传一个，优先使用 `name` 路径。

#### 6) 按 ID 拉取并落地到本地目录

命令：

```bash
cloudcc pullList classes <id> <projectPath>
```

参数：

| 参数          | 必填 | 类型     | 说明                                                 |
| ------------- | ---- | -------- | ---------------------------------------------------- |
| `id`          | 是   | `string` | 线上自定义类 ID                                      |
| `projectPath` | 是   | `string` | 项目根目录；会写入到 `<projectPath>/backend/classes/<name>/` |

#### 7) 删除自定义类

命令：

```bash
cloudcc delete classes <nameOrId> [projectPath]
```

参数规则：

| 参数          | 必填 | 类型     | 说明                                                                                                   |
| ------------- | ---- | -------- | ------------------------------------------------------------------------------------------------------ |
| `nameOrId`    | 是   | `string` | 可传本地目录名或线上 ID；若本地 `backend/classes/<nameOrId>/config.json` 存在且带 `id`，优先使用其中 `id` 删除 |
| `projectPath` | 否   | `string` | 项目根目录，默认当前目录                                                                               |

#### 8) 文档命令

命令：

```bash
cloudcc doc platform/classes <introduction|devguide>
```

参数：

| 参数          | 必填      | 类型 | 说明     |
| ------------- | --------- | ---- | -------- |
| `introduction | devguide` | 是   | `string`获取设计或是开发指导文档。 |

## 5. AI 必须遵守的硬规则

### 5.1 只能通过 cloudcc-cli 管理类目录

AI 不得手工创建或复制 `backend/classes/<类名>/` 目录，不得私自改动 `config.json` 中的
`id` 或版本信息。

### 5.2 只能在 SOURCE 区域内写业务代码

AI 修改现有类时，只能编辑主类中的：

```java
// @SOURCE_CONTENT_START
// @SOURCE_CONTENT_END
```

之间的内容。

不得破坏：

- 包路径
- 类目录结构
- `config.json`
- SOURCE 标记外的框架代码

### 5.3 必须保留 UserInfo 上下文

自定义类必须显式接收 `UserInfo`，并通过它构造平台能力对象。

标准写法：

```java
private UserInfo userInfo;
private CCService cs;

public XxxClass(UserInfo userInfo) {
    this.userInfo = userInfo;
    this.cs = new CCService(userInfo);
}
```

### 5.4 触发器、定时类、页面、按钮应做薄入口

AI 不能把大量核心逻辑直接堆在按钮入口、触发器入口、页面接口入口中。

推荐模式：

- 入口层只做参数接收、基础校验、结果返回
- 复杂逻辑沉到服务方法或独立服务类

### 5.5 必须显式处理异常

调用 `CCService` 等平台方法时，AI 不能默认“不会失败”。

必须：

- 显式 `throws Exception` 或 `try/catch`
- 在关键路径记录日志
- 保留失败原因
- 对外返回明确结果，而不是悄悄吞错

### 5.6 涉及时间必须使用 TimeUtil

AI 编写代码时，凡是时间写库、时间比较、格式化、Calendar
处理，都不能默认直接用本地时区对象。

必须优先使用：

- `TimeUtil.getNowDate(userInfo)`
- `TimeUtil.getUserTimeZone(userInfo)`
- `TimeUtil.getSimpleDateFormat(format, userInfo)`
- `TimeUtil.getCalendar(userInfo)`

禁止默认使用：

- `new Date()` 直接作为业务时间来源
- `Calendar.getInstance()` 不带用户时区
- `new SimpleDateFormat()` 后不设置时区

## 6. SDK 详细参考与使用规范

使用原则：

- 若某能力不在该公开文档中，不得编造参数或返回结构；可先搜本仓库同类实现再决定
- `ServiceResult`
  已文档化的用法：`isSuccess()`、`getMessage()`、`getId()`；其它键（如
  `insertNoException` 的 `resultlist`）以公开文档为准，未文档化的不得猜字段名
- 遇到未文档化能力时，先参考当前仓库相邻类的现有写法，再决定是否使用

### 6.1 CCObject 详细说明

`CCObject` 是 CloudCC 对象数据载体，继承自
`java.util.Map`，用于新增、更新、删除以及承载查询结果。

#### 6.1.1 构造方法

```java
public CCObject()
public CCObject(String ccobj)
public CCObject(String ccobj, String isShared)
```

参数说明：

| 方法                                      | 参数       | 类型     | 必填 | 说明                                        |
| ----------------------------------------- | ---------- | -------- | ---- | ------------------------------------------- |
| `CCObject(String ccobj)`                  | `ccobj`    | `String` | 是   | 对象 APIName                                |
| `CCObject(String ccobj, String isShared)` | `ccobj`    | `String` | 是   | 对象 APIName                                |
| `CCObject(String ccobj, String isShared)` | `isShared` | `String` | 是   | 共享标识，官方建议使用 `CCObject.IS_SHARED` |

#### 6.1.2 常量字段

| 常量                  | 默认值        | 说明         |
| --------------------- | ------------- | ------------ |
| `CCObject.OBJECT_API` | `CCObjectAPI` | 对象 API     |
| `CCObject.IS_SHARED`  | `isShared`    | 共享对象标识 |

#### 6.1.3 常用方法

```java
public String getObjectApiName()
public String put(String apiName, String value)
```

参数说明：

| 方法  | 参数      | 类型     | 必填 | 说明         |
| ----- | --------- | -------- | ---- | ------------ |
| `put` | `apiName` | `String` | 是   | 字段 APIName |
| `put` | `value`   | `String` | 是   | 字段值       |

AI 使用规则：

- 新增或更新对象时优先使用 `new CCObject("ObjectApiName")`
- 修改记录时必须写入 `id`
- 字段赋值一律使用字段 APIName，不用字段显示名
- 共享对象按官方方式传共享标识，不要自己拼魔法字符串
- 官方页示例把 `put` 记为 `String`
  参数；项目里也广泛传入数字、日期、布尔等对象。AI
  如果不确定类型，优先参考同对象的现有项目代码，不要自行猜测

### 6.2 UserInfo 详细说明

`UserInfo` 是当前登录用户上下文对象，是构造
`CCService`、`SendEmail`、`DevLogger`、`TimeUtil` 的基础。

#### 6.2.1 构造方法

```java
public UserInfo()
```

#### 6.2.2 官方页明确列出的常用方法

```java
public String getUserId()
public String getOrgId()
public String getRoleId()
```

参数说明：

| 方法        | 入参 | 返回     | 说明                             |
| ----------- | ---- | -------- | -------------------------------- |
| `getUserId` | 无   | `String` | 当前用户 ID，未设置时返回 `null` |
| `getOrgId`  | 无   | `String` | 当前组织 ID，未设置时返回 `null` |
| `getRoleId` | 无   | `String` | 当前角色 ID，未设置时返回 `null` |

AI 使用规则：

- 所有服务端类都应把 `UserInfo` 当作基础上下文
- 不要自己构造“当前用户”概念
- 不要使用硬编码用户替代 `UserInfo`
- 如果需要官方页未列出的 `UserInfo`
  方法，先去当前仓库搜索已有用法，再决定是否复用

### 6.3 CCService 详细说明
## 第1章 数据查询 API

---

### 1.1 cquery — 基础查询

**功能介绍**

查询指定对象下的记录列表，不校验用户角色权限，直接走数据库查询。内部根据系统配置自动决定是否翻译多语言字段。

**适用场景**

触发器内部逻辑、后台计算、系统级数据同步等**不需要权限控制**的内部调用场景，性能优先。

**方法签名**

```java
List<CCObject> cquery(String objectApiName, String expression)
List<CCObject> cquery(String objectApiName, String expression, String ordings)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是 | 对象 API 名称，如 `"Account"`、`"Contact"`，标准对象 `User` 自动映射为 `ccuser`，`Case` 自动映射为 `cloudcccase` |
| `expression` | String | 是 | 查询条件，SQL WHERE 子句格式，是数据库侧过滤边界，不是 Java 侧事后过滤的候选集描述。业务代码必须传入可收敛的过滤条件；`"1=1"` 只允许用于明确的后台诊断/导出场景，禁止用于代码兜底查重、存在性判断或幂等判断。普通自动编号应使用平台自动编号字段，不应通过查询已有记录后自行生成。`cquery*` 系列方法有平台返回条数上限，默认通常为 5000 且可配置，因此 `1=1` 后循环比对既慢又可能漏判。支持 `=`、`!=`、`>`、`<`、`LIKE`、`IN`、`AND`、`OR` 等标准 SQL 运算符 |
| `ordings` | String | 否 | 排序子句，格式为 `ORDER BY 字段名 ASC\|DESC`，支持多字段排序 |

**返回值**

`List<CCObject>`：匹配的记录列表，无结果时返回空列表，不返回 `null`。

**示例**

```java
// 查询状态为"启用"的客户
List<CCObject> activeAccounts = ccService.cquery("Account", "status__c = '启用'");

// 查询并按创建时间倒序
List<CCObject> sortedAccounts = ccService.cquery(
    "Account",
    "status__c = '启用'",
    "ORDER BY createdate DESC"
);

// 读取字段值
for (CCObject obj : sortedAccounts) {
    String id   = (String) obj.get("id");
    String name = (String) obj.get("name__c");
}
```

---

### 1.2 cqueryByFields — 指定字段查询

**功能介绍**

在条件查询基础上，限定只返回指定字段，减少数据传输量，提升查询性能。

**适用场景**

明确知道需要哪些字段时，优先使用此方法替代全字段 `cquery`，避免全字段返回造成的性能浪费。

**方法签名**

```java
List<CCObject> cqueryByFields(String objectApiName, String expression, String fields)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是 | 对象 API 名称 |
| `expression` | String | 是 | 查询条件，是数据库侧过滤边界。业务代码必须传入对象、业务键、状态、时间范围等边界条件；不要用 `"1=1"` 做查重或存在性判断。`cquery*` 结果有平台上限，默认通常为 5000 且可配置，不能依赖全量返回后再循环判断 |
| `fields` | String | 是 | 逗号分隔的字段 API 名列表，`id` 字段始终会返回 |

**返回值**

`List<CCObject>`：只包含指定字段的记录列表。

**示例**

```java
// 只返回 id、名称、电话三个字段
List<CCObject> list = ccService.cqueryByFields(
    "Account",
    "status__c = '启用'",
    "id,name__c,phone__c"
);

for (CCObject obj : list) {
    System.out.println(obj.get("name__c") + " - " + obj.get("phone__c"));
}
```

---

### 1.3 cqueryNoLang — 不翻译多语言查询

**功能介绍**

查询时跳过多语言字段翻译，直接返回字段存储值（如选项列表返回原始 key 而非翻译后的文本）。

**适用场景**

需要获取字段原始值（如数据迁移、后台逻辑比对、条件判断等），不希望被多语言翻译影响结果时使用。

**方法签名**

```java
List<CCObject> cqueryNoLang(String objectApiName, String expression, String fields, String ordings)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是 | 对象 API 名称 |
| `expression` | String | 否 | 查询条件，传 `null` 则查全部。业务代码不得为了查重、存在性、幂等或编号前置判断传 `null`/`1=1` 后循环比对；`cquery*` 结果有平台上限，默认通常为 5000 且可配置，可能导致漏判 |
| `fields` | String | 否 | 指定返回字段，传 `null` 返回全部字段 |
| `ordings` | String | 否 | 排序子句，传 `null` 不排序 |

**返回值**

`List<CCObject>`：不含多语言翻译的原始数据列表。

**示例**

```java
// 不翻译多语言，只返回 id 和 status__c 的原始值
List<CCObject> list = ccService.cqueryNoLang(
    "Account",
    "id = 'a001000001AbCdE'",
    "id,status__c",
    null
);

// status__c 返回原始 key，如 "Enable" 而非 "启用"
String rawStatus = (String) list.get(0).get("status__c");
```

---

### 1.4 cqueryWithRoleRight — 带权限查询

**功能介绍**

在查询前自动校验当前用户的对象权限、字段权限及记录级共享规则权限，只返回用户有权查看的数据。

**适用场景**

对外暴露的接口、用户主动触发的查询操作，凡是需要遵守数据权限的场景，**必须使用此系列方法**。

**方法签名**

```java
// 重载1：基础带权限查询
List<CCObject> cqueryWithRoleRight(String objectApiName, String expression)

// 重载2：控制是否返回已删除记录
List<CCObject> cqueryWithRoleRight(String objectApiName, String expression, String isAddDeleteHttp)

// 重载3：指定返回字段
List<CCObject> cqueryWithRoleRight(String objectApiName, String expression, String isAddDeleteHttp, String fields)

// 重载4：完整参数（含多货币 + 多语言）
List<CCObject> cqueryWithRoleRight(String objectApiName, String expression, String isAddDeleteHttp, String fields, String iscurrency, String islang)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是 | 对象 API 名称 |
| `expression` | String | 是 | 查询条件，是权限过滤前的数据库侧业务边界。对外或权限相关业务也必须传入可收敛条件；不要用 `"1=1"` 做查重、存在性判断或幂等判断。`cquery*` 结果有平台上限，默认通常为 5000 且可配置，不能替代唯一性判断 |
| `isAddDeleteHttp` | String | 否 | `"true"` 返回结果中包含删除权限标记；`"false"` 不包含，默认 `"false"` |
| `fields` | String | 否 | 指定字段，`null` 返回全部有权限的字段 |
| `iscurrency` | String | 否 | `"true"` 启用多货币 |
| `islang` | String | 否 | 语言码 |

**返回值**

`List<CCObject>`：仅包含当前用户有权查看的记录，无权限时返回空列表。

> 此方法会抛出 `Exception`，调用方需 try-catch 或声明 throws。

**示例**

```java
try {
    // 带权限查询客户列表
    List<CCObject> list = ccService.cqueryWithRoleRight(
        "Account", "status__c = '启用'"
    );

    // 带权限 + 指定字段
    List<CCObject> list2 = ccService.cqueryWithRoleRight(
        "Account", "status__c = '启用'",
        "false", "id,name__c,phone__c"
    );
} catch (Exception e) {
    // 处理权限异常
    e.printStackTrace();
}
```

---

### 1.5 cqueryWithRoleRightNoLang — 带权限不翻译多语言查询

**功能介绍**

在带权限查询基础上，跳过多语言翻译，返回字段原始存储值。

**适用场景**

需要带权限控制且需要拿到字段原始值（如选项值比对、数据导出）的场景。

**方法签名**

```java
List<CCObject> cqueryWithRoleRightNoLang(String objectApiName, String expression, String fields)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是 | 对象 API 名称 |
| `expression` | String | 是 | 查询条件，必须下推业务键、状态或时间范围等边界；不要用 `"1=1"`/`null` 后在 Java 中循环做查重、存在性或幂等判断 |
| `fields` | String | 否 | 指定返回字段，`null` 返回全部 |

**示例**

```java
List<CCObject> list = ccService.cqueryWithRoleRightNoLang(
    "Account", "id = 'a001000001AbCdE'", "id,status__c,type__c"
);
```

---

### 1.6 cqlQuery — CQL 语句查询

**功能介绍**

CloudCC Query Language（CQL）查询，类似 SQL，直接传入查询语句执行。支持单对象查询（返回 `List<CCObject>`）和多对象联查（返回 `List<Map>`）。

**适用场景**

复杂联查、需要精确控制 SQL 语句的场景，如跨对象关联查询等。业务查询默认只读取未逻辑删除数据。

**方法签名**

```java
// 单对象查询，返回 CCObject 列表
List<CCObject> cqlQuery(String objectApiName, String cql)

// 多对象联查，返回 Map 列表
List<Map> cqlQuery(String cql)

// 抛出异常版本（需要精确错误信息时）
List<CCObject> cqlQueryThrowException(String objectApiName, String cql)

// 含日志级别控制
List<CCObject> cqlQueryWithLogInfo(String objectApiName, String cql, String logLevel)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是（联查时不需要） | 主对象 API 名称，多对象联查时传逗号分隔的对象名，如 `"Account,Contact"` |
| `cql` | String | 是 | CQL 查询语句，语法参考 SQL SELECT，字段名使用对象字段 API 名。平台业务数据删除默认是逻辑删除，AI 生成业务查询时必须默认为每个业务对象拼接逻辑删除过滤，常见字段为 `is_deleted = '0'`；联查时按对象别名拼接，如 `a.is_deleted = '0'`、`c.is_deleted = '0'`。若目标对象实际字段为 `is_delete` 或其他名称，必须按实际字段拼接。只有用户明确要求查询回收站/已删除数据、删除审计或删除状态对比时，才允许不加该条件 |
| `logLevel` | String | 否 | 日志级别，`"DEBUG"`、`"INFO"`、`"ERROR"` |

**返回值**

- 单对象：`List<CCObject>`
- 联查：`List<Map>`，Map 的 key 为查询别名

> 注意：CQL 语句会经过合法性校验，禁止 DDL 操作（DROP、CREATE、ALTER 等），违反则返回 `null` 或抛出异常。

> 注意：`cqlQuery` 不要假设平台会自动排除逻辑删除数据。生成自定义类、触发器或定时类中的业务 CQL 时，默认必须把平台逻辑删除条件写入每个业务对象的 `WHERE` 条件，常见写法是 `is_deleted = '0'`；如果目标对象使用别名或不同逻辑删除字段名，例如 `is_delete`，必须按实际字段和别名补齐。

**示例**

```java
// 单对象 CQL 查询
List<CCObject> list = ccService.cqlQuery(
    "Account",
    "SELECT id, name__c, phone__c FROM Account " +
    "WHERE is_deleted = '0' AND status__c = '启用'"
);

// 多对象联查（联系人关联客户）
List<Map> list2 = ccService.cqlQuery(
    "SELECT a.id, a.name__c, c.name__c AS contactName " +
    "FROM Account a, Contact c " +
    "WHERE a.is_deleted = '0' AND c.is_deleted = '0' " +
    "AND a.id = c.accountid__c AND a.status__c = '启用'"
);

// 需要抛出异常时使用
try {
    List<CCObject> list3 = ccService.cqlQueryThrowException(
        "Account",
        "SELECT id, name__c FROM Account WHERE is_deleted = '0' AND id = 'xxx'"
    );
} catch (Exception e) {
    System.out.println("CQL 执行失败：" + e.getMessage());
}
```

错误写法：

```java
// 缺少 is_deleted = '0'，会把逻辑删除数据也纳入业务判断
List<CCObject> list = ccService.cqlQuery(
    "Account",
    "SELECT id, name__c FROM Account WHERE status__c = '启用'"
);
```

如果需求明确要看已删除数据，应在代码注释或方法名中说明业务原因，例如回收站查询、删除审计或删除状态对比；普通查重、存在性判断、同步读取、报表辅助查询、触发器/定时类业务处理都必须默认排除已删除数据。

---

### 1.7 pagedQuery — 分页查询

**功能介绍**

按页码和每页大小进行分页查询，不校验权限。

**适用场景**

列表页面的后端分页加载、批量数据处理分批读取等场景。

**方法签名**

```java
// 基础分页
List<CCObject> pagedQuery(String objectApiName, String expression, String pageNUM, String pageSize)

// 带删除权限标记
List<CCObject> pagedQuery(String objectApiName, String expression, String pageNUM, String pageSize, String isAddDelete)

// 指定字段
List<CCObject> pagedQuery(String objectApiName, String expression, String pageNUM, String pageSize, String isAddDelete, String fields)

// 完整参数（含多语言）
List<CCObject> pagedQuery(String objectApiName, String expression, String pageNUM, String pageSize, String isAddDelete, String fields, String islang)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是 | 对象 API 名称 |
| `expression` | String | 是 | 查询条件。分页查询也必须有业务边界；后台诊断以外不要用 `"1=1"` 扫描大对象。代码兜底查重或存在性判断使用 `pageNUM="1"`、`pageSize="1"`、`fields="id"`，把业务键条件直接写入 expression；普通保存查重优先使用 `dupeCatcher` |
| `pageNUM` | String | 是 | 当前页码，**从 `"1"` 开始**，传 String 类型 |
| `pageSize` | String | 是 | 每页记录数，如 `"20"`、`"50"`，传 String 类型 |
| `isAddDelete` | String | 否 | `"true"` 附带删除权限标记 |
| `fields` | String | 否 | 指定返回字段 |
| `islang` | String | 否 | 语言码，如 `"zh_CN"` |

**返回值**

`List<CCObject>`：当前页的记录列表。

**示例**

```java
// 基础分页，第1页，每页20条
List<CCObject> page1 = ccService.pagedQuery(
    "Account", "status__c = '启用'", "1", "20"
);

// 第2页，带删除权限标记，指定字段
List<CCObject> page2 = ccService.pagedQuery(
    "Account", "status__c = '启用'",
    "2", "20",
    "true",
    "id,name__c,phone__c"
);
```

---

### 1.8 pagedQueryWithRoleRight — 带权限分页查询

**功能介绍**

分页查询 + 权限校验，只返回当前用户有权查看的分页数据。

**适用场景**

列表页面的分页加载，需要遵守数据权限（共享规则、角色层级）的场景。

**方法签名**

```java
List<CCObject> pagedQueryWithRoleRight(String objectApiName, String expression, String pageNUM, String pageSize)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是 | 对象 API 名称 |
| `expression` | String | 是 | 查询条件，必须有业务边界；权限分页也不能用 `"1=1"` 后在 Java 中循环做查重、存在性或幂等判断 |
| `pageNUM` | String | 是 | 页码（从 `"1"` 开始） |
| `pageSize` | String | 是 | 每页记录数 |

**示例**

```java
List<CCObject> page = ccService.pagedQueryWithRoleRight(
    "Opportunity", "stage__c != '已关闭'", "1", "50"
);
```

---

### 1.9 getTotalRecordSize / pageQuery — 统计与简化分页

#### getTotalRecordSize — 获取记录总数

**功能介绍**

根据条件统计记录总数，用于分页时计算总页数。

**方法签名**

```java
// 基础统计（不含已删除记录）
long getTotalRecordSize(String objectApiName, String expression)

// 控制是否包含已删除记录
long getTotalRecordSize(String objectApiName, String expression, String isAddDelete)

// 带权限统计（只统计当前用户有权查看的记录）
long getTotalRecordSizeWithRoleRight(String objectApiName, String expression)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是 | 对象 API 名称 |
| `expression` | String | 是 | 查询条件 |
| `isAddDelete` | String | 否 | `"true"` 统计时包含已软删除的记录 |

**返回值**

`long`：满足条件的记录总数。

**示例**

```java
// 获取启用状态客户总数
long total = ccService.getTotalRecordSize("Account", "status__c = '启用'");

// 配合分页使用
int pageSize = 20;
long total2 = ccService.getTotalRecordSize("Account", "status__c = '启用'");
int totalPage = (int) Math.ceil((double) total2 / pageSize);

// 带权限统计
long myTotal = ccService.getTotalRecordSizeWithRoleRight(
    "Opportunity", "stage__c != '已关闭'"
);
```

#### pageQuery — 简化分页（返回 Map）

**功能介绍**

分页查询的简化版本，直接返回包含 `total`（总数）和 `list`（当前页数据）的 `Map`，免去单独调用 `getTotalRecordSize` 的步骤。

**方法签名**

```java
Map pageQuery(String objectApiName, String expression, String pageNUM, String pageSize)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是 | 对象 API 名称 |
| `expression` | String | 是 | 查询条件 |
| `pageNUM` | String | 是 | 页码（从 `"1"` 开始） |
| `pageSize` | String | 是 | 每页记录数 |

**返回值**

`Map`：包含以下两个 key：
- `total`（`long`）：满足条件的记录总数
- `list`（`List<CCObject>`）：当前页的记录列表

**示例**

```java
Map result = ccService.pageQuery("Account", "status__c = '启用'", "1", "20");

long total = (long) result.get("total");
List<CCObject> list = (List<CCObject>) result.get("list");

System.out.println("总记录数：" + total);
System.out.println("当前页记录数：" + list.size());
```

---

## 第2章 数据写入 API

---

### 2.1 insert — 基础新增

**功能介绍**

向指定对象新增一条记录。默认执行触发器（Before/After），不触发工作流，不校验权限，不执行校验规则和查重规则。

**适用场景**

触发器内部、后台系统级新增，性能优先且不需要权限检查的场景。

**方法签名**

```java
// 标准新增（执行触发器，不执行工作流，不校验权限）
ServiceResult insert(CCObject ccobj)

// 批量新增（事务处理）
ServiceResult insert(List<CCObject> ccobjs)

// 完整参数控制
ServiceResult insert(CCObject ccobj, boolean isRight, boolean isWorkFlow,
                     boolean isBeforeTrigger, boolean isAfterTrigger,
                     boolean isValidAndDuplication)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `ccobj` | `CCObject` | 是 | 待新增的对象实例，需先设置对象 API 名，再 `put` 字段值 |
| `ccobjs` | `List<CCObject>` | 是（批量时） | 待新增的对象列表，在同一事务中批量执行 |
| `isRight` | boolean | 是 | `true` 校验用户操作权限；`false` 跳过权限校验 |
| `isWorkFlow` | boolean | 是 | `true` 新增后触发工作流；`false` 不触发 |
| `isBeforeTrigger` | boolean | 是 | `true` 执行 Before 触发器；`false` 跳过 |
| `isAfterTrigger` | boolean | 是 | `true` 执行 After 触发器；`false` 跳过 |
| `isValidAndDuplication` | boolean | 是 | `true` 执行校验规则和查重规则；`false` 跳过 |

**返回值**

`ServiceResult`：操作结果，调用 `result.isSuccess()` 判断是否成功，`result.getId()` 获取新记录 ID。

**示例**

```java
// 构建对象
CCObject account = new CCObject("Account");
account.put("name__c", "测试公司");
account.put("phone__c", "13800000000");
account.put("status__c", "启用");

// 标准新增
ServiceResult result = ccService.insert(account);
if (result.isSuccess()) {
    String newId = result.getId();
    System.out.println("新增成功，ID：" + newId);
} else {
    System.out.println("新增失败：" + result.getMessage());
}

// 完整控制参数（触发器中防递归：关闭所有开关）
ServiceResult result2 = ccService.insert(account,
    false,   // 不校验权限
    false,   // 不触发工作流
    false,   // 不执行 Before 触发器
    false,   // 不执行 After 触发器
    false    // 不执行校验规则
);
```

---

### 2.2 insertWithRoleRight — 带权限新增

**功能介绍**

新增前校验当前用户是否有对该对象的创建权限，执行触发器，不触发工作流。

**适用场景**

对外接口中用户主动创建记录的场景，需要遵守系统权限配置。

**方法签名**

```java
ServiceResult insertWithRoleRight(CCObject ccobj)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `ccobj` | `CCObject` | 是 | 待新增的对象实例 |

**示例**

```java
CCObject contact = new CCObject("Contact");
contact.put("firstname__c", "张");
contact.put("lastname__c", "三");
contact.put("accountid__c", "001000001AbCdEfGh");

ServiceResult result = ccService.insertWithRoleRight(contact);
if (result == null || !result.isSuccess()) {
    throw new BusiException("创建联系人失败：" +
        (result != null ? result.getMessage() : "未知错误"));
}
String contactId = result.getId();
```

---

### 2.3 insertWithWorkflow — 带工作流新增

**功能介绍**

新增记录后触发工作流规则，不校验权限，执行触发器。

**适用场景**

需要新增后自动触发工作流（如发送通知、创建任务、字段更新）的场景。

**方法签名**

```java
ServiceResult insertWithWorkflow(CCObject ccobj)
```

**示例**

```java
CCObject lead = new CCObject("Lead");
lead.put("firstname__c", "李");
lead.put("lastname__c", "四");
lead.put("leadsource__c", "网络");

ServiceResult result = ccService.insertWithWorkflow(lead);
```

---

### 2.4 insertLt — 轻量新增

**功能介绍**

跳过触发器（Before/After）和工作流的轻量级新增，不校验权限，不校验规则，直接写库。

**适用场景**

大批量数据导入、初始化数据填充等对性能要求极高且确定不需要触发器的场景。

**方法签名**

```java
ServiceResult insertLt(CCObject ccobj)
```

> 注意：此方法跳过所有联动逻辑，慎用。若数据完整性依赖触发器，请使用 `insert()` 而非此方法。

**示例**

```java
CCObject obj = new CCObject("Product");
obj.put("name__c", "产品A");
obj.put("price__c", 100.00);

// 轻量写入，不触发任何联动
ServiceResult result = ccService.insertLt(obj);
```

---

### 2.5 insertWithValidationAndDuplication — 带校验规则+查重新增

**功能介绍**

新增时同时执行校验规则（Validation Rule）和查重规则（Duplication Rule），校验通过后才写库。

**适用场景**

业务数据录入时需要保证数据质量，执行平台配置的校验和查重规则。

**方法签名**

```java
ServiceResult insertWithValidationAndDuplication(CCObject ccobj)
```

**示例**

```java
CCObject account = new CCObject("Account");
account.put("name__c", "新公司");
account.put("phone__c", "400-123-4567");

ServiceResult result = ccService.insertWithValidationAndDuplication(account);
if (!result.isSuccess()) {
    // 校验失败或查重命中
    System.out.println("提示：" + result.getMessage());
}
```

---

### 2.6 insertNoException — 批量新增不抛异常

**功能介绍**

批量新增时，单条记录失败不中断整个批次，每条记录独立返回结果。

**适用场景**

导入场景中允许部分失败、需要逐条返回成功/失败结果的批量写入。

**方法签名**

```java
ServiceResult insertNoException(List<CCObject> ccobjs)
```

**返回值**

`ServiceResult`：`result.get("resultlist")` 为 `List<Map>`，每条 Map 包含：
- `isSuccess`（boolean）：该条是否成功
- `id`（String）：成功时的新记录 ID
- `errormessage`（String）：失败时的错误信息

**示例**

```java
List<CCObject> objs = new ArrayList<>();
for (int i = 0; i < 100; i++) {
    CCObject obj = new CCObject("Account");
    obj.put("name__c", "公司" + i);
    objs.add(obj);
}

ServiceResult result = ccService.insertNoException(objs);
List<Map> resultList = (List<Map>) result.get("resultlist");

long successCount = resultList.stream()
    .filter(m -> Boolean.TRUE.equals(m.get("isSuccess")))
    .count();
System.out.println("成功：" + successCount + " / " + objs.size());
```

---

### 2.7 upsert — 新增/更新二合一

**功能介绍**

根据 `CCObject` 中是否包含 `id` 字段自动路由：有 `id` 则更新，无 `id` 则新增。

**适用场景**

同步数据时不确定记录是否已存在，使用 `upsert` 简化逻辑判断。

**方法签名**

```java
ServiceResult upsert(CCObject ccobj)            // 标准版
ServiceResult upsertLt(CCObject ccobj)           // 轻量版（跳过联动）
ServiceResult upsertWithRoleRight(CCObject ccobj) // 带权限版
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `ccobj` | `CCObject` | 是 | 若含 `id` 字段则更新该记录，否则新增 |

**示例**

```java
CCObject account = new CCObject("Account");

// 场景1：更新（有 id）
account.put("id", "001000001AbCdEfGh");
account.put("name__c", "更新后的公司名");

// 场景2：新增（无 id）
// account.put("name__c", "全新公司");

ServiceResult result = ccService.upsert(account);
if (result.isSuccess()) {
    String id = result.getId(); // 新增时返回新 ID，更新时返回原 ID
}
```

---

### 2.8 update — 基础更新

**功能介绍**

更新指定记录，`CCObject` 中必须包含 `id` 字段。默认执行触发器，不触发工作流，不校验权限。

**方法签名**

```java
// 标准更新
ServiceResult update(CCObject ccobj)

// 完整控制参数
ServiceResult update(CCObject ccobj, boolean isRight, boolean isWorkFlow,
                     boolean isBeforeTrigger, boolean isAfterTrigger)

// 带校验规则 + 查重
ServiceResult update(CCObject ccobj, boolean isRight, boolean isWorkFlow,
                     boolean isBeforeTrigger, boolean isAfterTrigger,
                     boolean isValidAndDuplication)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `ccobj` | `CCObject` | 是 | 必须包含 `id` 字段，其余字段为待更新值 |
| `isRight` | boolean | 是（完整版） | 是否校验权限 |
| `isWorkFlow` | boolean | 是（完整版） | 是否触发工作流 |
| `isBeforeTrigger` | boolean | 是（完整版） | 是否执行 Before 触发器 |
| `isAfterTrigger` | boolean | 是（完整版） | 是否执行 After 触发器 |
| `isValidAndDuplication` | boolean | 否（完整版） | 是否执行校验规则和查重规则 |

**示例**

```java
CCObject account = new CCObject("Account");
account.put("id", "001000001AbCdEfGh");
account.put("name__c", "新公司名称");
account.put("status__c", "禁用");

// 标准更新
ServiceResult result = ccService.update(account);

// 触发器中更新其他对象，关闭触发器防递归
CCObject related = new CCObject("Opportunity");
related.put("id", "006000001XyZaBc");
related.put("syncstatus__c", "已同步");
ccService.update(related, false, false, false, false);
```

---

### 2.9 updateWithRoleRight — 带权限更新

**功能介绍**

更新前校验当前用户是否有对该记录的编辑权限（包含对象权限和记录级权限），通过后执行更新。

**方法签名**

```java
String updateWithRoleRight(CCObject ccobj)
```

**返回值**

`String`：`"success"` 表示成功，`"failure:错误信息"` 表示失败。

**示例**

```java
CCObject obj = new CCObject("Contact");
obj.put("id", "003000001AbCdE");
obj.put("title__c", "技术总监");

String result = ccService.updateWithRoleRight(obj);
if (!"success".equals(result)) {
    System.out.println("更新失败：" + result);
}
```

---

### 2.10 updateWithWorkFlow — 带工作流更新

**功能介绍**

更新记录后触发工作流规则，不校验权限，执行触发器。

**方法签名**

```java
ServiceResult updateWithWorkFlow(CCObject ccobj)
```

**示例**

```java
CCObject opp = new CCObject("Opportunity");
opp.put("id", "006000001AbCdE");
opp.put("stage__c", "已赢单");

ServiceResult result = ccService.updateWithWorkFlow(opp);
```

---

### 2.11 updateWithValidationAndDuplication — 带校验规则+查重更新

**功能介绍**

更新时执行校验规则和查重规则，不通过则返回失败信息。

**方法签名**

```java
ServiceResult updateWithValidationAndDuplication(CCObject ccobj)
```

**示例**

```java
CCObject account = new CCObject("Account");
account.put("id", "001000001AbCdE");
account.put("name__c", "可能重复的公司名");

ServiceResult result = ccService.updateWithValidationAndDuplication(account);
if (!result.isSuccess()) {
    System.out.println("校验失败：" + result.getMessage());
}
```

---

### 2.12 delete — 删除

**功能介绍**

删除指定记录（软删除，移入回收站）。`CCObject` 中必须包含 `id` 字段。

**方法签名**

```java
// 标准删除（不校验权限）
ServiceResult delete(CCObject ccobj)

// 带权限删除（返回 String）
String deleteWithRoleRight(CCObject ccobj)

// 批量删除（返回 boolean）
boolean delete(List<CCObject> ccobjs)

// 按条件删除共享记录
void deleteShareObjectBySql(String objectApiName, String expression)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `ccobj` | `CCObject` | 是 | 必须包含 `id` 字段 |
| `ccobjs` | `List<CCObject>` | 是（批量时） | 批量待删除对象列表，每条必须有 `id` |

**返回值**

- `ServiceResult`：操作结果
- `String`（带权限版）：`"success"` 或错误信息
- `boolean`（批量版）：是否全部成功

**示例**

```java
// 删除单条记录
CCObject toDelete = new CCObject("Account");
toDelete.put("id", "001000001AbCdE");
ServiceResult result = ccService.delete(toDelete);

// 带权限删除
String delResult = ccService.deleteWithRoleRight(toDelete);
if (!"success".equals(delResult)) {
    System.out.println("删除失败：" + delResult);
}

// 批量删除
List<CCObject> toDeleteList = new ArrayList<>();
for (String id : idList) {
    CCObject obj = new CCObject("Contact");
    obj.put("id", id);
    toDeleteList.add(obj);
}
boolean success = ccService.delete(toDeleteList);
```

---

## 第3章 批量操作 API

---

### 3.1 batchInsert — 批量新增

**功能介绍**

在单个事务中批量新增多条记录，性能优于循环调用 `insert()`，失败时整体回滚。

**方法签名**

```java
// 标准批量新增（不校验权限，执行触发器，不触发工作流）
ServiceResult batchInsert(List<CCObject> ccobjs)

// 完整控制参数
ServiceResult batchInsert(List<CCObject> ccobjs, boolean isRight, boolean isWorkFlow,
                           boolean isTrigger, boolean isValidAndDuplication)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `ccobjs` | `List<CCObject>` | 是 | 待新增的对象列表，同一对象类型 |
| `isRight` | boolean | 否 | 是否校验权限 |
| `isWorkFlow` | boolean | 否 | 是否触发工作流 |
| `isTrigger` | boolean | 否 | 是否执行触发器（Before + After） |
| `isValidAndDuplication` | boolean | 否 | 是否执行校验规则和查重规则 |

**返回值**

`ServiceResult`：整体操作结果，单条失败时整个事务回滚。

**示例**

```java
List<CCObject> products = new ArrayList<>();
for (int i = 1; i <= 1000; i++) {
    CCObject p = new CCObject("Product");
    p.put("name__c", "产品" + i);
    p.put("price__c", i * 10.0);
    products.add(p);
}

// 批量新增（推荐，一次事务）
ServiceResult result = ccService.batchInsert(products);
if (result.isSuccess()) {
    System.out.println("批量新增成功");
}
```

---

### 3.2 batchUpdate — 批量更新

**功能介绍**

在单个事务中批量更新多条记录，每条记录必须包含 `id` 字段。

**方法签名**

```java
ServiceResult batchUpdate(List<CCObject> ccobjs)
ServiceResult batchUpdate(List<CCObject> ccobjs, boolean isRight, boolean isWorkFlow,
                           boolean isTrigger, boolean isValidAndDuplication)
```

**入参说明**

同 `batchInsert`，每条 `CCObject` 必须含 `id` 字段。

**示例**

```java
List<CCObject> toUpdate = new ArrayList<>();
for (CCObject obj : queryResult) {
    CCObject update = new CCObject("Account");
    update.put("id", obj.get("id"));
    update.put("syncstatus__c", "已处理");
    toUpdate.add(update);
}

ServiceResult result = ccService.batchUpdate(toUpdate);
```

---

### 3.3 batchDelete — 批量删除

**功能介绍**

在单个事务中批量删除多条记录。

**方法签名**

```java
ServiceResult batchDelete(List<CCObject> ccobjs)
ServiceResult batchDelete(List<CCObject> ccobjs, boolean isRight)
```

**示例**

```java
List<CCObject> toDelete = objs.stream().map(obj -> {
    CCObject d = new CCObject("TempRecord");
    d.put("id", obj.get("id"));
    return d;
}).collect(Collectors.toList());

ServiceResult result = ccService.batchDelete(toDelete);
```

---

### 3.4 insertWithRoleRightForBatch — 带权限批量新增（逐条返回结果）

**功能介绍**

带权限校验的批量新增，每条记录独立处理并返回结果，单条失败不影响其他记录。

**方法签名**

```java
List<Map> insertWithRoleRightForBatch(List<CCObject> ccobjs)

// 完整控制
List<Map> insertForBatch(List<CCObject> ccobjs, boolean isRight, boolean isWorkFlow,
                          boolean isBeforeTrigger, boolean isAfterTrigger,
                          boolean isValidAndDuplication)
```

**返回值**

`List<Map>`：每条记录的处理结果，Map 包含：
- `isSuccess`（boolean）
- `id`（String）：成功时的新记录 ID
- `errormessage`（String）：失败时的错误信息

**示例**

```java
List<Map> results = ccService.insertWithRoleRightForBatch(objs);
results.forEach(r -> {
    if (Boolean.TRUE.equals(r.get("isSuccess"))) {
        System.out.println("成功，ID：" + r.get("id"));
    } else {
        System.out.println("失败：" + r.get("errormessage"));
    }
});
```

---

### 3.5 updateWithRoleRightForBatch — 带权限批量更新（逐条返回结果）

**方法签名**

```java
List<Map> updateWithRoleRightForBatch(List<CCObject> ccobjs)

List<Map> updateForBatch(List<CCObject> ccobjs, boolean isRight, boolean isWorkFlow,
                          boolean isBeforeTrigger, boolean isAfterTrigger,
                          boolean isValidAndDuplication)
```

**示例**

```java
List<Map> results = ccService.updateWithRoleRightForBatch(toUpdateList);
long failCount = results.stream()
    .filter(r -> !Boolean.TRUE.equals(r.get("isSuccess")))
    .count();
System.out.println("失败条数：" + failCount);
```

---

## 第4章 审批流程 API

---

### 4.1 submitForApproval — 提交审批

**功能介绍**

将指定记录提交到审批流程，触发审批流，向审批人发送待办通知。

**方法签名**

```java
String submitForApproval(String relateId, String fprId, String appPath)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `relateId` | String | 是 | 待审批记录的 ID |
| `fprId` | String | 是 | 提交人（发起人）的用户 ID，通常为 `userInfo.getUserId()` |
| `appPath` | String | 是 | 审批流程入口路径，用于生成审批链接，通常传系统配置值 |

**返回值**

`String`：`"success"` 或错误信息。

**示例**

```java
String recordId = "006000001AbCdEfGh";
String submitterId = userInfo.getUserId();
String appPath = "/approval/process";

String result = ccService.submitForApproval(recordId, submitterId, appPath);
```

---

### 4.2 process — 审批处理

**功能介绍**

处理审批工作项，支持通过、拒绝、转交等操作。

**方法签名**

```java
String process(String processType, String workItemId, String fprId,
               String comments, String appPath)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `processType` | String | 是 | 操作类型：`"approve"` 通过，`"reject"` 拒绝，`"transfer"` 转交 |
| `workItemId` | String | 是 | 审批工作项 ID（从待审批列表获取） |
| `fprId` | String | 是 | 操作人用户 ID |
| `comments` | String | 否 | 审批意见，可为 `null` |
| `appPath` | String | 是 | 审批流程路径 |

**示例**

```java
// 通过审批
String result = ccService.process(
    "approve",
    "workitem001AbCdE",
    userInfo.getUserId(),
    "同意，质量符合标准",
    "/approval/process"
);

// 拒绝审批
ccService.process("reject", workItemId, userId, "不符合要求，请修改后重新提交", appPath);
```

---

### 4.3 recall — 撤回审批

**功能介绍**

撤回已提交的审批申请，将审批状态重置为草稿。

**方法签名**

```java
String recall(String relateId, String comments)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `relateId` | String | 是 | 被撤回的记录 ID |
| `comments` | String | 否 | 撤回原因说明 |

**示例**

```java
String result = ccService.recall(
    "006000001AbCdEfGh",
    "发现数据填写有误，需要修改"
);
```

---

### 4.4 showPendingHis — 查询待审批列表

**功能介绍**

查询指定用户当前的待处理审批项目列表。

**方法签名**

```java
List<Map> showPendingHis(String userId)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `userId` | String | 是 | 待审批用户 ID，传 `null` 或空字符串则使用当前用户 |

**返回值**

`List<Map>`：每条 Map 包含 `objtype`（对象类型）、`objid`（记录 ID）、`objname`（记录名称）、`workItemid`（工作项 ID）、`comments`（摘要）、`actdate`（提交时间）等字段。

**示例**

```java
List<Map> pendingList = ccService.showPendingHis(userInfo.getUserId());

for (Map item : pendingList) {
    System.out.println("记录：" + item.get("objname"));
    System.out.println("工作项ID：" + item.get("workItemid"));
}
```

---

### 4.5 getApprovalPendingSize — 获取待审批数量

**功能介绍**

获取当前用户待处理的审批数量，通常用于首页待办角标。

**方法签名**

```java
String getApprovalPendingSize()
```

**返回值**

`String`：待审批数量字符串，如 `"5"`。

**示例**

```java
String count = ccService.getApprovalPendingSize();
System.out.println("待审批：" + count + " 条");
```

---

## 第5章 动态消息 Chatter API

---

### 5.1 addMicroPost — 发布动态

**功能介绍**

向 Chatter 动态流发布一条消息，支持纯文本、链接、附件、地理位置等多种内容类型。

**方法签名**

```java
// 基础发布
String addMicroPost(String feedType, String linkName, String linkValue,
                    String fileName, String fileType, InputStream fileStream,
                    File file, String bodyType, String body,
                    String targetType, String targetId, String recordId)

// 带地理位置
String addMicroPost(String feedType, String linkName, String linkValue,
                    String fileName, String fileType, InputStream fileStream,
                    File file, String bodyType, String body,
                    String targetType, String targetId, String recordId,
                    String longitude, String latitude, String address,
                    String taskIdOrEventId)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `feedType` | String | 是 | 动态类型，纯文字传 `"TextPost"` |
| `linkName` | String | 否 | 链接名称，无链接传 `null` |
| `linkValue` | String | 否 | 链接 URL，无链接传 `null` |
| `fileName` | String | 否 | 附件文件名，无附件传 `null` |
| `fileType` | String | 否 | 附件 MIME 类型，如 `"image/png"` |
| `fileStream` | InputStream | 否 | 附件文件流，无附件传 `null` |
| `file` | File | 否 | 附件 File 对象，无附件传 `null` |
| `bodyType` | String | 是 | 内容类型，传 `"text"` |
| `body` | String | 是 | 动态正文内容 |
| `targetType` | String | 是 | 目标类型：`"record"` 关联记录，`"user"` 关联用户 |
| `targetId` | String | 是 | 目标 ID（记录 ID 或用户 ID） |
| `recordId` | String | 否 | 关联记录 ID |
| `longitude` | String | 否 | 经度 |
| `latitude` | String | 否 | 纬度 |
| `address` | String | 否 | 地理位置文字描述 |
| `taskIdOrEventId` | String | 否 | 关联任务或事件 ID |

**返回值**

`String`：新动态的 Feed ID。

**示例**

```java
String feedId = ccService.addMicroPost(
    "TextPost",
    null, null,
    null, null, null, null,
    "text",
    "这个机会已完成初步沟通，客户反馈积极！",
    "record",
    "006000001AbCdE",
    "006000001AbCdE"
);

System.out.println("动态 ID：" + feedId);
```

---

### 5.2 addMicroComment — 发布评论

**功能介绍**

对已有动态发布评论，支持附带附件。

**方法签名**

```java
String addMicroComment(String feedid, String fileName, String fileType,
                        InputStream fileStream, String body)

String addMicroComment(String feedid, String fileName, String fileType,
                        InputStream fileStream, String body,
                        String longitude, String latitude, String address)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `feedid` | String | 是 | 被评论的动态 Feed ID |
| `fileName` | String | 否 | 附件文件名，无附件传 `null` |
| `fileType` | String | 否 | 附件类型 |
| `fileStream` | InputStream | 否 | 附件流 |
| `body` | String | 是 | 评论内容 |

**示例**

```java
String commentId = ccService.addMicroComment(
    feedId,
    null, null, null,
    "已安排会议，下周一上午10点"
);
```

---

### 5.3 getChatters — 查询动态列表

**功能介绍**

查询动态流列表，支持按类型、记录、用户等维度过滤。

**方法签名**

```java
List<Map> getChatters(String queryType, String userId, String feedid,
                       String recordId, String feedsort, String limit, String skip)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `queryType` | String | 是 | `"all"` 全部，`"self"` 当前用户发布的 |
| `userId` | String | 否 | 过滤指定用户的动态 |
| `feedid` | String | 否 | 过滤指定 Feed（用于获取评论） |
| `recordId` | String | 否 | 过滤关联指定记录的动态 |
| `feedsort` | String | 否 | 排序方式，`"asc"` 或 `"desc"` |
| `limit` | String | 是 | 最多返回条数 |
| `skip` | String | 是 | 跳过条数（分页偏移） |

**示例**

```java
List<Map> chatters = ccService.getChatters(
    "all",
    null,
    null,
    "006000001AbCdE",
    "desc",
    "10",
    "0"
);
```

---

### 5.4 praiseFeed — 点赞/取消点赞

**方法签名**

```java
String praiseFeed(String feedid, String type, String iscomments)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `feedid` | String | 是 | 动态 Feed ID |
| `type` | String | 是 | `"praise"` 点赞，`"cancel"` 取消点赞 |
| `iscomments` | String | 是 | `"true"` 表示是对评论点赞，`"false"` 表示对动态点赞 |

**示例**

```java
ccService.praiseFeed(feedId, "praise", "false");
ccService.praiseFeed(feedId, "cancel", "false");
```

---

## 第6章 文件操作 API

---

### 6.1 uploadFile / insertFile — 上传文件

**功能介绍**

上传文件到平台文件存储，返回文件 ID 用于后续关联。

**方法签名**

```java
String uploadFile(Map fileInfo, InputStream inputStream)
String insertFile(Map fileInfo, InputStream inputStream)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `fileInfo` | Map | 是 | 文件元信息，包含 `name`（文件名）、`type`（MIME 类型）、`size`（文件大小，字节）、`relatedId`（关联记录 ID）等 |
| `inputStream` | InputStream | 是 | 文件输入流 |

**返回值**

`String`：上传成功后的文件 ID。

**示例**

```java
Map<String, Object> fileInfo = new HashMap<>();
fileInfo.put("name", "合同附件.pdf");
fileInfo.put("type", "application/pdf");
fileInfo.put("size", fileBytes.length);
fileInfo.put("relatedId", "001000001AbCdE");

InputStream is = new ByteArrayInputStream(fileBytes);
String fileId = ccService.uploadFile(fileInfo, is);
System.out.println("文件 ID：" + fileId);
```

---

### 6.2 downloadFile / getFile — 下载文件

**功能介绍**

根据文件 ID 下载文件，返回包含文件流和文件名的 Map。

**方法签名**

```java
Map downloadFile(String fileId)
Map getFile(String fileId)
```

**返回值**

`Map`：包含以下字段：
- `filename`（String）：文件名
- `stream`（InputStream）：文件内容流
- `type`（String）：MIME 类型
- `size`（Long）：文件大小

**示例**

```java
Map fileData = ccService.downloadFile("file_id_001AbCdE");

String filename = (String) fileData.get("filename");
InputStream stream = (InputStream) fileData.get("stream");

try (FileOutputStream fos = new FileOutputStream("/tmp/" + filename)) {
    byte[] buffer = new byte[4096];
    int len;
    while ((len = stream.read(buffer)) != -1) {
        fos.write(buffer, 0, len);
    }
}
```

---

### 6.3 queryFileList — 查询文件列表

**功能介绍**

按条件查询文件记录列表。

**方法签名**

```java
List<Map> queryFileList(Map conditionMap)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `conditionMap` | Map | 是 | 过滤条件 Map，如 `{"relatedId": "001xxx"}` |

**示例**

```java
Map conditions = new HashMap<>();
conditions.put("relatedId", "001000001AbCdE");

List<Map> files = ccService.queryFileList(conditions);
files.forEach(f -> System.out.println(f.get("name") + " - " + f.get("size")));
```

---

### 6.4 cqueryFileField — 查询文件字段

**功能介绍**

查询指定记录上某个文件字段包含的文件列表。

**方法签名**

```java
List<CCObject> cqueryFileField(String recordId, String fieldApiName)
```

**示例**

```java
List<CCObject> avatars = ccService.cqueryFileField(
    "003000001AbCdE",
    "avatar__c"
);
```

---

## 第7章 邮件 API

---

### 7.1 sendEmail — 发送邮件

**功能介绍**

发送邮件，支持模板邮件和自定义内容邮件，支持 CC/BCC/附件/开信追踪等能力。

**方法签名**

```java
// 模板邮件
String sendEmail(String templateId, String toaddress, String ccaddress,
                 String bcaddress, String relatedid)

// 自定义内容（无 CC）
String sendEmail(String name, String htmlbody, String toaddress, String relatedid)

// 带 CC
String sendEmailWithccaddress(String name, String htmlbody, String toaddress,
                               String relatedid, String ccaddress)

// 带附件
String sendEmailWithAttachment(String name, String htmlbody, String toaddress,
                                String relatedid, Object attachment)

// 完整参数
String sendEmail(String name, String htmlbody, String toaddress, String relatedid,
                 String ccaddress, String bccaddress, Object attachment,
                 String istrackopen, String singleSend)

// 带 File 附件
ServiceResult sendEmailWithFile(String content, String subject, String toaddress,
                                 String ccaddress, String bcaddress,
                                 List<File> fileList, List<InputStream> inputList,
                                 List<byte[]> byteList)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `templateId` | String | 是（模板版） | 邮件模板 ID |
| `name` | String | 是（自定义版） | 邮件主题 |
| `htmlbody` | String | 是（自定义版） | 邮件正文（支持 HTML） |
| `toaddress` | String | 是 | 收件人地址，多个用逗号分隔 |
| `ccaddress` | String | 否 | 抄送地址，多个用逗号分隔 |
| `bccaddress` / `bcaddress` | String | 否 | 密送地址 |
| `relatedid` | String | 否 | 关联记录 ID，邮件将归档到该记录 |
| `attachment` | Object | 否 | 附件对象 |
| `istrackopen` | String | 否 | `"true"` 追踪收件人是否开信 |
| `singleSend` | String | 否 | `"true"` 逐人单独发送（每封邮件只有一个收件人） |

**返回值**

`String`：`"success"` 或错误信息。

**示例**

```java
String result = ccService.sendEmail(
    "合同到期提醒",
    "<h2>您好</h2><p>您的合同将于30天后到期，请及时续签。</p>",
    "customer@example.com",
    "contract_id_001"
);

String result2 = ccService.sendEmail(
    "email_template_id_xxx",
    "to@example.com",
    "cc@example.com",
    "bcc@example.com",
    "lead_id_001"
);

String result3 = ccService.sendEmail(
    "季度报告",
    "<p>请查收本季度报告...</p>",
    "manager@example.com",
    "opportunity_id_001",
    null, null, null,
    "true",
    "true"
);
```

---

### 7.2 queryEmailIn / queryEmailOut — 查询邮件记录

**功能介绍**

查询收件箱或发件箱中的邮件记录。

**方法签名**

```java
List<Map> queryEmailIn(String startDate, String endDate)
List<Map> queryEmailIn(List<String> userIds, List<String> emails, String startDate, String endDate)
List<Map> queryEmailOut(String startDate, String endDate)
List<Map> queryEmail(String incoming, List<String> userIds, List<String> emails, String startDate, String endDate)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `startDate` | String | 是 | 开始时间，格式 `"yyyy-MM-dd"` |
| `endDate` | String | 是 | 结束时间，格式 `"yyyy-MM-dd"` |
| `userIds` | List<String> | 否 | 过滤指定用户 ID 列表 |
| `emails` | List<String> | 否 | 过滤指定邮件地址列表 |
| `incoming` | String | 是 | `"1"` 收件，`"0"` 发件 |

**示例**

```java
List<Map> inbox = ccService.queryEmailIn("2026-04-01", "2026-04-30");
List<Map> outbox = ccService.queryEmailOut("2026-04-01", "2026-04-30");
```

---

## 第8章 视图操作 API

---

### 8.1 getViewList — 获取视图列表

**功能介绍**

获取指定对象下当前用户有权限访问的视图列表。

**方法签名**

```java
List<Map> getViewList(String objectApiName)
List<Map> getViewListNew(String objectApiName)
```

**示例**

```java
List<Map> views = ccService.getViewList("Account");
views.forEach(v -> System.out.println(v.get("id") + " - " + v.get("name")));
```

---

### 8.2 queryByViewId — 按视图查询数据

**功能介绍**

按视图 ID 查询数据，自动应用视图的过滤条件、字段显示配置、排序设置。

**方法签名**

```java
Map queryByViewId(String viewId, String pages, String pageSize,
                   String sort, String dir, String keyWord, String fieldApis)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `viewId` | String | 是 | 视图 ID |
| `pages` | String | 是 | 页码（从 `"1"` 开始） |
| `pageSize` | String | 是 | 每页记录数 |
| `sort` | String | 否 | 排序字段 API 名 |
| `dir` | String | 否 | 排序方向：`"ASC"` 或 `"DESC"` |
| `keyWord` | String | 否 | 搜索关键词 |
| `fieldApis` | String | 否 | 指定返回字段，逗号分隔 |

**返回值**

`Map`：包含 `total`（总数）和 `list`（记录列表）。

**示例**

```java
Map result = ccService.queryByViewId(
    "view_id_001",
    "1", "20",
    "createdate", "DESC",
    null, null
);

long total = (long) result.get("total");
List<CCObject> list = (List<CCObject>) result.get("list");
```

---

### 8.3 searchByViewId — 视图关键词搜索

**功能介绍**

在视图基础条件上叠加关键词搜索和自定义过滤条件。

**方法签名**

```java
Map searchByViewId(String viewId, String pages, String pageSize, String sort,
                    String dir, String keyword, String fieldApis, String expression)

Map searchByViewId(String objid, String viewId, String pages, String pageSize,
                    String sort, String dir, String keyword, String fieldApis,
                    String expression)
```

**示例**

```java
Map result = ccService.searchByViewId(
    "view_id_001",
    "1", "20", "createdate", "DESC",
    "张三",
    "id,name__c",
    "status__c='启用'"
);
```

---

## 第9章 自定义设置 API

---

### 9.1 getListCustomSetting — 获取列表型自定义设置

**功能介绍**

获取在"自定义设置"（Custom Setting）中配置的参数值，支持列表型设置的全量获取和按 key 查询。

**方法签名**

```java
Map getListCustomSetting(String objectApiName)
Map getListCustomSetting(String objectApiName, String key)
Map getCustomSetting(String objectApiName, String id)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是 | 自定义设置对象的 API 名称 |
| `key` | String | 否 | 记录的名称（Name 字段值），不传则返回全部 |
| `id` | String | 是（按 ID 版） | 记录 ID |

**返回值**

`Map`：自定义设置的字段键值对。

**示例**

```java
Map allConfig = ccService.getListCustomSetting("SystemConfig__c");
Map smtpConfig = ccService.getListCustomSetting("EmailConfig__c", "SMTP_SERVER");
String smtpHost = (String) smtpConfig.get("host__c");
int smtpPort = Integer.parseInt((String) smtpConfig.get("port__c"));

System.out.println("SMTP 地址：" + smtpHost + ":" + smtpPort);
```

---

## 第10章 工具方法 API

---

### 10.1 getPicklistValue — 获取下拉选项值

**功能介绍**

获取指定字段的下拉选项（Picklist）值列表，支持按记录类型过滤。

**方法签名**

```java
List<Map> getPicklistValue(String objectApiName, String fieldApi, String recordtype)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `objectApiName` | String | 是 | 对象 API 名称 |
| `fieldApi` | String | 是 | 字段 API 名称 |
| `recordtype` | String | 否 | 记录类型 ID，`null` 返回所有选项 |

**返回值**

`List<Map>`：每条 Map 包含 `value`（选项值/key）和 `label`（显示文本）。

**示例**

```java
List<Map> statusOptions = ccService.getPicklistValue(
    "Account", "status__c", null
);

statusOptions.forEach(opt ->
    System.out.println(opt.get("value") + " -> " + opt.get("label"))
);
```

---

### 10.2 getObjectRight — 获取对象操作权限

**功能介绍**

获取当前用户对指定对象的操作权限级别。

**方法签名**

```java
String getObjectRight(String objectApiName)
```

**返回值**

`String`：权限标识，如 `"Read"`（只读）、`"Edit"`（可编辑）、`"All"`（全权限）、`"none"`（无权限）。

**示例**

```java
String right = ccService.getObjectRight("Account");
if ("none".equals(right)) {
    throw new BusiException("无客户对象操作权限");
}
```

---

### 10.3 getAccessToken — 获取当前用户 AccessToken

**功能介绍**

获取当前登录用户的 AccessToken，可用于 API 调用认证。

**方法签名**

```java
String getAccessToken()
```

**示例**

```java
String token = ccService.getAccessToken();
```

---

### 10.4 getCurrencyRateInfos — 获取多货币汇率

**功能介绍**

获取平台配置的所有货币及其对公司本位币的汇率信息。

**方法签名**

```java
List<Map> getCurrencyRateInfos()
```

**返回值**

`List<Map>`：每条包含 `currencyCode`（货币代码如 `"USD"`）、`rate`（汇率）、`isActive`（是否启用）等。

**示例**

```java
List<Map> rates = ccService.getCurrencyRateInfos();
rates.forEach(r ->
    System.out.println(r.get("currencyCode") + " : " + r.get("rate"))
);
```

---

### 10.5 executeAsyncJob — 执行异步任务

**功能介绍**

通过 MQ/调度器触发后台异步任务，适合将耗时逻辑从同步接口中剥离出来，避免请求超时。
`executeAsyncJob` 主要用于常规异步方法调度；`executeAsyncCustomClass` 主要用于自定义类异步执行场景。

**方法签名**

```java
ServiceResult executeAsyncJob(Map paramMap)
ServiceResult executeAsyncCustomClass(Map paramMap)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `paramMap.className` | String | 是 | 需要执行的类名 |
| `paramMap.methodName` | String | 是 | 需要异步执行的方法名 |
| `paramMap.name` | String | 否 | 任务名称；不传默认 `className-methodName` |
| `paramMap.executeDate` | Date | 否 | 执行时间；不传表示立即执行，传入表示定时执行 |
| `paramMap.ismq` | String | 否 | 是否直接投递 MQ，支持 `"true"`/`"false"`；默认 `"false"` |
| `paramMap.queueName` | String | 否 | MQ 队列名；多队列隔离时建议指定 |
| `paramMap.<业务参数>` | Any | 否 | 业务自定义参数；会透传到异步执行上下文中 |

**执行行为说明**

- 未传 `executeDate`：立即执行。根据事务状态，任务可能先写入缓存，事务提交后再投递队列。
- 传入 `executeDate`：走调度中心注册任务，到达触发时间后执行。
- 使用 `executeAsyncCustomClass`：按定时任务方式注册，任务类型为自定义类异步任务。
- 环境需开启 `cloudcc.schedule.async.enable=true`，否则 `executeAsyncJob` 会返回失败。

**返回值**

`ServiceResult`：

- `success=true`：提交成功（表示任务已入队或已注册，不代表业务方法已执行完成）。
- `success=false`：提交失败，可通过错误信息定位配置或参数问题。
- 成功时可通过 `result.get("taskId")` 或 `result.getRtnInfo()` 获取任务标识。

**示例**

```java
Map params = new HashMap<>();
params.put("className", "AccountSyncJob");
params.put("methodName", "syncAccounts");
params.put("name", "Account增量同步");
params.put("ismq", "true");
params.put("queueName", "account.sync");
params.put("batchSize", 500);
params.put("operatorId", "005xxxxxxxxxxxx");

ServiceResult result = ccService.executeAsyncJob(params);
if (result.isSuccess()) {
    System.out.println("异步任务已提交, taskId=" + result.get("taskId"));
} else {
    System.out.println("提交失败: " + result.getErrorMessage());
}
```

**定时执行示例**

```java
Map params = new HashMap<>();
params.put("className", "BillingJob");
params.put("methodName", "generateMonthlyBill");
params.put("name", "月度账单生成");
params.put("executeDate", new java.util.Date(System.currentTimeMillis() + 5 * 60 * 1000));

ServiceResult result = ccService.executeAsyncJob(params);
```

**最佳实践**

- 异步方法需保证幂等（可按业务主键或 `taskId` 去重），避免重复消费造成脏数据。
- 大任务拆批执行，不要在单个异步任务中处理超大数据集。
- 关键链路建议记录 `taskId`，便于排障和日志追踪。
- 如果依赖数据库事务结果，优先在事务提交后再触发异步任务。

---

### 10.6 queryTianyancha — 天眼查企业信息

**功能介绍**

通过天眼查 API 查询企业工商信息，需要平台配置天眼查密钥。

**方法签名**

```java
ServiceResult queryTianyancha(String searchword)
ServiceResult tianYanChaDetailInfo(String keyword, String id)
```

**入参说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `searchword` | String | 是 | 企业名称或关键词 |
| `keyword` | String | 是 | 搜索关键词 |
| `id` | String | 是 | 天眼查企业 ID（从列表查询结果中获取） |

**示例**

```java
ServiceResult searchResult = ccService.queryTianyancha("阿里巴巴");
ServiceResult detail = ccService.tianYanChaDetailInfo("阿里巴巴", "tianyancha_id_xxx");
```

---

## 附录A 最佳实践

### A.1 触发器中防递归

在 After Trigger 中更新其他对象时，**必须关闭触发器开关**，防止触发再次调用触发器形成无限递归。

```java
ServiceResult result = ccService.update(relatedObj,
    false,
    false,
    false,
    false
);
```

---

### A.2 批量操作优先使用 batchInsert / batchUpdate

```java
List<CCObject> objs = buildDataList();
ccService.batchInsert(objs);
```

---

### A.3 查询只取需要的字段

```java
List<CCObject> list = ccService.cqueryByFields(
    "Account", "status__c='启用'", "id,name__c,phone__c"
);
```

---

### A.4 对外接口必须使用 WithRoleRight 版本

```java
List<CCObject> list = ccService.cqueryWithRoleRight("Account", expression);
ServiceResult result = ccService.insertWithRoleRight(obj);
```

---

### A.5 upsert 简化新增/更新逻辑

```java
CCObject obj = new CCObject("Account");
if (existingId != null && !existingId.isEmpty()) {
    obj.put("id", existingId);
}
obj.put("name__c", "公司名称");
obj.put("status__c", "启用");

ServiceResult result = ccService.upsert(obj);
```

---

### A.6 ServiceResult 标准判断写法

```java
ServiceResult result = ccService.insertWithRoleRight(obj);
if (result == null || !result.isSuccess()) {
    String msg = (result != null) ? result.getMessage() : "操作返回为空";
    throw new BusiException("保存失败：" + msg);
}
String newId = result.getId();
```

---

### A.7 CQL 查询安全注意事项

```java
String name = userInput;
List<CCObject> list = ccService.cquery(
    "Account",
    "name__c = '" + name.replace("'", "\\'") + "'"
);

List<CCObject> list2 = ccService.cqueryByFields(
    "Account", "name__c = '" + name + "'", "id,name__c"
);
```

## 7. 代码结构规范

### 7.1 方法粒度

AI 不应默认把所有逻辑塞进一个超长方法。

推荐拆分：

- `validateXxx`
- `queryXxx`
- `calculateXxx`
- `updateXxx`
- `sendXxx`
- `buildXxxResult`

原则：

- 一个方法只负责一类动作
- 复杂编排方法负责串联，不负责所有细节

### 7.1.1 单文件规模限制

AI 生成或改写自定义类时，单个 Java 源文件必须控制在 2000 行以下；预计超过 1500 行时，应先拆分设计，再继续写代码。

需要拆分的典型信号：

- 一个类同时处理多个业务对象或多个业务阶段
- 同一个 SOURCE 区域里同时包含查询、校验、编号、审批、通知、外部接口、批处理等多类职责
- 方法数量持续膨胀，入口方法只是在串联多个大块逻辑
- 同一类里出现多个可以独立测试或独立复用的业务能力

推荐拆分方式：

- 入口类只负责参数校验和编排
- 查询读取拆成 `XxxQueryService`
- 查重/校验拆成 `XxxValidator`
- 编号/计算拆成 `XxxCalculator`
- 写入动作拆成 `XxxWriter`
- 外部集成拆成 `XxxClient` 或 `XxxIntegrationService`

如果用户需求复杂，AI 应先给出类拆分清单，再分别生成多个 `backend/classes/<ClassName>/` 自定义类；不要把完整业务系统压进一个超长类。

### 7.2 返回值规范

AI 需要根据调用场景选择返回值类型：

- 页面/按钮/组件接口：优先 `Map<String,Object>`、`List<Map<String,Object>>` 或稳定 Java DTO
- 内部服务方法：优先返回明确业务结果
- 仅做写操作的方法：可返回 `void`，但必须通过异常或日志暴露失败

规则：

- 返回值结构要稳定
- 不要同一方法一会儿返回字符串、一会儿返回对象
- 对外接口最好带成功标识和错误信息
- 不推荐使用 `net.sf.json.JSONObject` / `JSONArray` 作为业务返回值或通用中间结构；该库效率低，且 null 值容易产生 `JSONNull` 相关运行时问题
- 如果历史方法签名或外部调用方已经固定为 `net.sf.json`，只能做局部兼容；新生成代码必须优先返回 `Map` / `List<Map>`，并显式处理 null

## 8. 数据与查询规范

### 8.1 字段名统一使用 API 名称

AI 不得使用字段显示名直接拼查询条件。

必须：

- 使用对象 APIName
- 使用字段 APIName

### 8.2 查询要收敛

先做能力分流：

- 所有需求都要先判断平台标准元数据能力是否可满足，不局限于某个低代码功能。对象/字段、页面布局、验证规则、查重过滤器、工作流/审批、共享/权限、公式/汇总、自动编号、查找筛选、相关列表等能满足时，优先用平台元数据实现；只有低代码能力无法表达时，才写自定义类代码。
- 普通自动编号、单据号、流水号、客户号、合同号等需求，优先使用平台自动编号字段（字段类型 `V`，通过 `platform/fields devguide` 的 MetadataService spec 生成字段行和 `tp_sys_autonum` 配置），不要在自定义类里用查询已有记录再 `+1` 的方式模拟编号。
- 保存时按对象字段和条件判断重复、提示或阻断重复记录，优先使用平台查重过滤器 `dupeCatcher`。只有查重过滤器无法表达的跨对象、动态外部校验、复杂聚合条件或后台治理场景，才进入自定义类代码实现。
- 能由字段、验证规则、查重过滤器、工作流、审批等平台元数据实现的需求，不要默认写 Java 代码。

AI 默认应避免：

- 无条件全表查询
- 无限制 `select *`
- 在循环里反复查同一批数据
- 在 `for` / `while` 循环内调用 `cquery`、`cqueryByFields`、`cqlQuery`、`pagedQuery` 等查询方法做逐条补字段、逐条查重或逐条存在性判断
- 为了判断“是否存在”“是否重复”“是否已处理”而先查出整张表再在 Java 中循环比对
- 把 `cquery*` 的 `expression` 当成“候选集描述”，再用 Java 判断真正条件

根因说明：

- `expression` 是数据库侧过滤条件，查重、存在性、幂等和编号前置判断必须把真正的业务键下推到这里
- `cquery*` 相关方法有平台返回条数上限，默认通常为 5000 且可配置；`1=1` 后循环最多只能判断返回窗口内的数据，大表上会出现假阴性
- 循环内查询会形成 N+1 查询：输入 200 条就可能打 201 次查询，触发器和定时类中会快速放大为超时、锁等待或接口限流
- 大数量查重、补齐引用对象字段、同步前置读取、列表组装等场景必须先收集 key，再批量查询和本地映射
- `cqlQuery` 是手写 CQL，不要假设平台自动过滤逻辑删除数据；默认必须为每个业务对象拼接 `is_deleted = '0'`
- 即使当前数据量小，生成代码也不能假设对象永远小；这类代码一旦进入触发器、定时类或复用自定义类，会随业务增长放大为性能和正确性问题

应优先：

- 一次查够
- 只查必要字段
- 把业务键、当前记录排除条件、状态、时间范围等边界写进查询条件
- 存在性判断只取 `id`，并用 `pagedQuery(..., "1", "1", ..., "id")` 或有明确条件的 `cqueryByFields`
- 使用 `cqlQuery` 联查时，为每个业务对象或别名补上逻辑删除条件，例如 `a.is_deleted = '0' AND c.is_deleted = '0'`
- 先遍历输入收集去重后的 ID / 业务键，再用 `IN` 条件或分页分批查询必要字段
- 提前构造 `Map<Id, CCObject>`、`Map<String, CCObject>` 或 `Map<String, Map<String,Object>>`，循环阶段只做内存查找和结果组装
- 当 `IN` 值较多时按固定批次分片查询，避免单条 CQL/SQL 过长
- 把批量处理合并到同一逻辑块

查重/存在性判断禁止写法：

```java
List accounts = cs.cqueryByFields("Account", "1=1", "id,tyshxydm,code");
for (int i = 0; i < accounts.size(); i++) {
    CCObject account = (CCObject) accounts.get(i);
    if (newTax.equals(account.get("tyshxydm"))) {
        duplicateTax = true;
    }
}
```

如果平台 `dupeCatcher` 无法覆盖，代码兜底写法才应把查重字段直接下推到查询条件，只返回必要字段：

```java
String escapedTax = newTax.replace("'", "\\'");
String expression = "tyshxydm = '" + escapedTax + "'";
if (currentId != null && currentId.trim().length() > 0) {
    expression += " and id != '" + currentId.replace("'", "\\'") + "'";
}

List exists = cs.pagedQuery("Account", expression, "1", "1", "false", "id");
boolean duplicateTax = exists != null && exists.size() > 0;
```

循环内查询禁止写法：

```java
for (int i = 0; i < materials.size(); i++) {
    CCObject material = (CCObject) materials.get(i);
    String materialId = text(material.get("sku_id"));
    List products = cs.cqueryByFields(
        "Product",
        "id = '" + escape(materialId) + "'",
        "id,name,cpdm,kz"
    );
    // 每条 material 都查一次 Product，形成 N+1 查询
}
```

正确写法应先收集 key，再批量查询并构建映射：

```java
Set<String> materialIds = new HashSet<String>();
for (int i = 0; i < materials.size(); i++) {
    CCObject material = (CCObject) materials.get(i);
    String materialId = text(material.get("sku_id"));
    if (materialId.length() > 0) {
        materialIds.add(materialId);
    }
}

Map<String, CCObject> productById = new HashMap<String, CCObject>();
if (!materialIds.isEmpty()) {
    String expression = "id in (" + quoteForIn(materialIds) + ")";
    List products = cs.cqueryByFields("Product", expression, "id,name,cpdm,kz");
    if (products != null) {
        for (int i = 0; i < products.size(); i++) {
            CCObject product = (CCObject) products.get(i);
            productById.put(text(product.get("id")), product);
        }
    }
}

for (int i = 0; i < materials.size(); i++) {
    CCObject material = (CCObject) materials.get(i);
    CCObject product = productById.get(text(material.get("sku_id")));
    // 这里只做内存映射和结果组装
}
```

`quoteForIn` 代表本地辅助方法：必须对每个 ID/业务键做去空、去重、单引号转义，再拼成 `'id1','id2'` 形式；如果 key 数量很多，应按固定批次分片生成多个 `IN` 查询。

普通编号生成不能为了 `size()` 查全量，也不应通过 `max/count + 1` 在自定义类里模拟；必须优先交给平台自动编号字段。只有平台自动编号无法覆盖的特殊号段需求，才允许使用独立编号对象或受控代码实现，并且必须说明并发锁、幂等和回滚策略。

### 8.3 写操作要可回溯

涉及新增、更新、删除时：

- 要明确主键或条件
- 要记录关键业务键
- 要尽量保证幂等
- 要避免重复插入

## 9. 外部集成规范

AI 编写集成类时，必须把外部系统调用与业务逻辑适度隔离。

规则：

- 接口地址、密钥、系统标识优先配置化
- 不要在多个业务方法中重复拼接 URL
- 外部调用前后必须记录日志
- 调用失败要保留错误上下文
- 文件上传、下载、同步逻辑要单独封装

推荐模式：

- `Client` 负责调用外部接口
- `Service` 负责业务编排
- `Controller` 负责入口输出

## 10. 通知与协同规范

AI 编写通知能力时，应遵循：

- 先完成业务动作，再触发通知
- 通知内容应基于真实业务结果生成
- 收件人/待办对象来源应可追溯
- 同一动作不要重复发送

如果通知逻辑复杂：

- 单独提取通知方法
- 不要把大段 HTML 拼接散落到多处

## 11. 权限与共享规范

AI 处理权限相关需求时，应优先使用服务端判断，而不是把关键限制交给前端。

规则：

- 角色判断放服务端
- 审批权限放服务端
- 共享写入放服务端
- 删除共享关系要使用正确接口

当需求和共享、角色、负责人变化相关时，AI 应主动考虑：

- 是否要补共享
- 是否要删旧共享
- 是否要校验当前用户是否有权执行

## 12. 异常与日志规范

AI 写代码时必须假设以下事情都可能失败：

- 查询为空
- 字段为空
- 配置缺失
- 外部接口超时
- 文件不存在
- 写操作失败
- 状态不满足前置条件

因此必须：

- 对空值做保护
- 对异常做显式处理
- 对关键节点打日志
- 对外返回明确错误

不允许：

- 空 `catch`
- 吞异常后继续假装成功
- 失败时只返回 `false` 而不给任何上下文

## 13. 命名规范

优先遵循以下模式：

- `XxxController`：入口编排类
- `XxxService`：核心业务服务类
- `XxxUtil` / `XxxUtils`：通用工具类
- `XxxClient`：外部系统客户端
- `XxxHandler`：触发逻辑处理类

方法命名优先使用动词开头：

- `queryXxx`
- `createXxx`
- `updateXxx`
- `deleteXxx`
- `sendXxx`
- `calculateXxx`
- `buildXxx`

## 14. AI 禁止事项

AI 不得：

- 手工新建 `backend/classes/<类名>/` 目录
- 修改 `config.json` 的 `id` 或版本字段
- 在 SOURCE 区域外写业务代码
- 默认使用 `new Date()` 作为业务时间
- 使用字段显示名代替 API 名称
- 无条件全量查询大对象
- 在循环内执行查询造成 N+1 查询；必须先收集 key 批量读取
- 生成缺少 `is_deleted = '0'` 的业务 `cqlQuery`，除非需求明确包含已删除数据
- 生成超过 2000 行的单个 Java 源文件
- 将复杂业务需求全部塞进一个自定义类而不拆分职责
- 将关键逻辑全部堆在入口方法里
- 在多个地方复制同样的外部接口调用代码
- 把地址、角色、阈值、模板 ID 等全部硬编码
- 吞异常不报错
- 只写“成功/失败”而没有上下文日志

## 15. AI 生成代码前的检查清单

在开始写代码前，AI 应先确认：

1. 这是页面/按钮/触发器/定时类/组件中的哪一种入口
2. 这段逻辑是否真的应该放到自定义类
3. 会影响哪些对象
3. 是否需要外部系统调用
5. 是否需要附件或文件处理
6. 是否涉及时区
7. 是否涉及权限、共享或审批
8. 哪些参数适合配置化而不是硬编码

## 16. AI 交付前的自检清单

AI 完成代码后，必须自检：

1. 是否只修改了 SOURCE 区域
2. 是否保留了 `UserInfo` + `CCService`
3. 是否正确使用对象/字段 API 名称
3. 是否按场景选择了查询 API（内部无权限用 `cquery` / `cqueryByFields`
   等，对外或需数据权限用 `cqueryWithRoleRight`；复杂联查用 `cqlQuery`，分页用
   `pagedQuery` / `pageQuery` 等）
4. 是否所有查重、存在性判断、幂等判断都带业务键过滤，并且没有用 `1=1` 全量查询后循环比对
5. 是否没有在循环内执行查询；补齐引用字段、查重、存在性判断是否先收集 key 后批量查询并构建 Map
6. 是否所有业务 `cqlQuery` 都默认排除逻辑删除数据；多对象联查是否为每个对象别名加了 `is_deleted = '0'`
7. 是否对写操作做了失败处理
8. 是否对关键步骤加了日志
9. 是否避免了直接 `new Date()`
10. 是否把复杂逻辑拆成了可读的方法
11. 是否单个 Java 文件低于 2000 行；复杂需求是否拆分成多个自定义类
12. 是否避免了不必要硬编码
13. 是否让返回结果对调用方足够清晰

## 17. 推荐骨架

下面是一个适合 AI 生成新类时参考的通用骨架：


```java
import java.util.HashMap;
import java.util.List;
import java.util.Map;
// 这里包含了DevLogger，SendEmail等常用方法，不用再单独引入。
import com.cloudcc.core.*;

// @SOURCE_CONTENT_START
public class XxxService {
    private final UserInfo userInfo;
    private final CCService cs;
    private final DevLogger logger;

    public XxxService(UserInfo userInfo) {
        this.userInfo = userInfo;
        this.cs = new CCService(userInfo);
        this.logger = new DevLogger(userInfo);
    }

    public Map<String, Object> execute(String recordId) throws Exception {
        Map<String, Object> result = new HashMap<String, Object>();
        logger.devLogInfo("XxxService.execute start, recordId=" + recordId);
        try {
            validate(recordId);

            List<CCObject> records = queryData(recordId);
            Map<String, Object> calcResult = calculate(records);
            updateData(recordId, calcResult);

            result.put("success", true);
            result.put("data", calcResult);
            logger.devLogInfo("XxxService.execute success, recordId=" + recordId);
            return result;
        } catch (Exception e) {
            logger.devLogError("XxxService.execute failed, recordId=" + recordId, e);
            throw e;
        }
    }

    private void validate(String recordId) throws Exception {
        if (recordId == null || "".equals(recordId.trim())) {
            throw new Exception("recordId is required");
        }
    }

    private List<CCObject> queryData(String recordId) throws Exception {
        return cs.cquery("ObjectApiName", "id = '" + recordId + "'", null);
    }

    private Map<String, Object> calculate(List<CCObject> records) {
        Map<String, Object> result = new HashMap<String, Object>();
        result.put("count", records == null ? 0 : records.size());
        return result;
    }

    private void updateData(String recordId, Map<String, Object> calcResult) throws Exception {
        CCObject obj = new CCObject("ObjectApiName");
        obj.put("id", recordId);
        obj.put("last_execute_time", TimeUtil.getNowDate(userInfo));
        cs.update(obj);
    }
}
// @SOURCE_CONTENT_END
```

## 18. 一句话结论

AI 编写 CloudCC 自定义类时，最重要的不是“把需求翻译成几段 Java”，而是遵守
CloudCC 的类管理方式、正确使用
SDK、把复杂逻辑放在服务端可治理的位置，并让代码具备复用性、可追踪性和可发布性。
