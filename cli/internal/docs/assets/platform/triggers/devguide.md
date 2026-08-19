# CloudCC 触发器开发规范

## 1. 核心原则

触发器只做事件入口，复杂业务逻辑统一写入自定义类。

推荐职责划分：

- 触发器：获取参数、判断条件、调用自定义类、必要时报错
- 自定义类：承载业务规则、聚合其他类、负责编排逻辑

## 2. 开发规范

- 一个对象的一个触发时机，只创建一个触发器
- 创建本地触发器骨架使用：`cloudcc create trigger <objectApi/TriggerName> [projectPath]`。
- 直接保存线上元数据使用：`cloudcc create trigger <projectPath> <triggerJson|@file>`。
- 尽量不要在触发器中直接开发业务逻辑
- 需要复用、编排、扩展的逻辑，统一下沉到自定义类

## 3. 触发时间

- `beforeInsert`：当新建、复制记录行满足触发器条件，执行触发器操作
- `beforeUpdate`：编辑记录行满足触发器条件，执行触发器操作
- `beforeUpsert`：新建和编辑、复制记录行满足触发器条件，执行触发器操作
- `beforeDelete`：删除记录行满足触发器条件，执行触发器操作
- `afterInsert`：新建、复制记录行满足触发器条件，执行触发器操作
- `afterUpdate`：修改记录行满足触发器条件，执行触发器操作
- `afterUpsert`：新建或编辑、复制记录行满足触发器条件，执行触发器操作
- `afterDelete`：删除记录行满足触发器条件，执行触发器操作
- `afterInsertCommit`：插入提交后，报错不会回滚
- `afterUpdateCommit`：更新提交后，报错不会回滚
- `afterUpsertCommit`：保存提交后，报错不会回滚
- `afterDeleteCommit`：删除提交后，报错不会回滚
- `approval`：审批

## 4. 可直接引用的参数

- `(UserInfo)userInfo`
- `(Map<String, Object> )record_old`
- `(Map<String, Object> )record_new`

取记录值时，直接使用字段 API 名：

```java
record_old.get("name");
record_new.get("name");
```

## 5. 保存前提示

在记录保存前可使用：

```java
trigger.addErrorMessage("提示内容");
```

用于向用户提示并阻断保存。

## 8. 推荐写法

触发器中只保留入口逻辑：

```java
public class MyTriggerService extends CCTrigger {
    public MyTriggerService() {
        super(userInfo);
        // @SOURCE_CONTENT_START
        try {
            MyTriggerService service = new MyTriggerService(userInfo,(CCService)this);
            service.execute(record_old, record_new);
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
        // @SOURCE_CONTENT_END
    }
}
```

自定义类中承载业务实现：

```java
import java.util.Map;
// @SOURCE_CONTENT_START
public class MyTriggerService {
    private UserInfo userInfo;
    private CCService cs;

    public MyTriggerService(UserInfo userInfo) {
        this(userInfo, new CCService(userInfo));
    }

    public MyTriggerService(UserInfo userInfo, CCService cs) {
        this.userInfo = userInfo;
        this.cs = cs;
    }

    public void execute(Map<String, Object> record_old, Map<String, Object> record_new) {
            // 在这里编写业务逻辑
    }
}
// @SOURCE_CONTENT_END
```

## 9. 触发器元数据与 CLI 契约

触发器的真实元数据表是 `tp_sys_trigger`。它属于 CloudCC 原生高代码资源，不属于 MetadataService 低代码账本。资源名 `trigger` 与 `triggers` 完全等价；命令参数统一为“项目路径优先”。

不能根据 REST 命名习惯猜测 `/api/trigger/list`、`/api/trigger/detail` 或 `/api/trigger/delete`。真实 setup-svc 契约如下：

| 操作 | 真实端点 | 关键请求 |
| --- | --- | --- |
| 全局列表 | `POST /api/triggerSetup/getTriggerByCondition` | `shownum`、`showpage`、`sname`、`objId`、`fid`、`rptcond`、`rptorder` |
| 详情 | `POST /api/trigger/newobjtrigger` | `{ "id": "<triggerId>" }` |
| 发布前校验 | `POST /api/trigger/validate` | 当前触发器源码和元数据 |
| 创建/更新 | `POST /api/triggerSetup/saveTrigger` | `TpSysTriggerVO` 字段 |
| 删除 | `POST /api/triggerSetup/deleteTrigger` | `{ "id": "<triggerId>" }` |

`POST /api/trigger/queryTriggerList` 是按 `objid` 查询的对象级接口，不是全局列表。没有 `objid` 时即使返回成功和空列表，也不能据此判断租户没有触发器。

### 9.1 查询全局列表

```bash
cloudcc get trigger <projectPath> [nameOrQueryJson]
cloudcc get triggers <projectPath> [nameOrQueryJson]
```

不传查询条件时，CLI 使用：

```json
{
  "shownum": "2000",
  "showpage": "1",
  "sname": "",
  "objId": "",
  "fid": "",
  "rptcond": "lastmodifydate",
  "rptorder": "desc"
}
```

第二个参数可以是名称字符串，也可以是 JSON/`@file` 覆盖上述查询字段。

### 9.2 查询详情与删除

```bash
cloudcc detail trigger <projectPath> <id|name|apiName>
cloudcc delete trigger <projectPath> <id|name|apiName>
```

CLI 会先从全局列表按 `id`、`name`、`apiname` 或 `apiName` 精确解析唯一 ID，再调用详情或删除端点。名称/API 名匹配到多条时必须失败并要求使用唯一 ID，禁止猜测后执行删除。

`pull` 和 `pullList` 当前与 `detail` 使用同一只读详情契约并输出服务端响应，不宣称已经写入本地目录。

### 9.3 创建、更新与发布

直接保存元数据：

```bash
cloudcc create trigger <projectPath> <triggerJson|@file>
cloudcc update trigger <projectPath> <triggerJson|@file>
cloudcc save trigger <projectPath> <triggerJson|@file>
```

`update` 必须包含 `id`。常用字段包括 `id`、`name`、`apiname`、`apiName`、`isactive`、`folderid`、`version`、`triggerTime`、`targetObjectId`、`remark` 和 `triggerSource`。可以用 `sourceFile` 指向带 SOURCE 标记的本地 Java 文件；CLI 会读取标记内内容。

创建本地骨架与发布：

```bash
cloudcc create trigger <objectApi/TriggerName> [projectPath]
cloudcc publish trigger <objectApi/TriggerName> [projectPath]
```

`publish trigger` 的发布顺序固定为：

1. 远程 validate，调用 `POST /api/trigger/validate`。
2. 保存，调用 `POST /api/triggerSetup/saveTrigger`。


远程 validate 失败时，CLI 必须返回 setup-svc 的 `returnInfo`、`data.errors`、`data.warnings` 和原始 `responseBody`，并且不能继续 save。

从 CLI/技能 `2.2.7` 开始，`publish trigger` 要求目标 setup-svc 至少为 `19.3.R20`，因为旧版本 setup-svc 不提供 `/api/trigger/validate`。

`/api/trigger/validate` 的实际入参类型是 `TriggerVo`。服务端实际读取：

| 字段 | validate 中的作用 | 是否必须 |
| --- | --- | --- |
| `triggerSource` | 待编译触发器源码 | 必须，不能为空 |
| `apiname` | 编译时使用的触发器 API 名；为空时服务端默认 `TriggerFunctionImpl` | 建议传，避免类名/诊断上下文退化为默认值 |
| `triggerTime` | 判断是否 batch trigger：`beforeBatch`、`afterBatch`、`commitBatch` 会按 batch 模板编译 | batch 触发器必须传；普通触发器建议传 |
| `version` | 传给编译器的触发器版本 | 建议传；CLI 默认 `2` |

CLI 还会随 validate body 带上 `id`、`name`、`isactive`、`targetObjectId`、`remark`、`folderid`、`apiName` 等字段，这是为了和后续 `/api/triggerSetup/saveTrigger` payload 保持一致；这些字段不参与 validate 编译判断。

trigger 的 validate 和 save 编码规则不同：`/api/trigger/validate` 不做 URLDecoder 解码，所以 CLI 直接传原始 `triggerSource`；`/api/triggerSetup/saveTrigger` 会用 Java `URLDecoder` 解码 `triggerSource`，CLI 必须使用 query-component 兼容编码，确保源码里的字面量 `+` 编码为 `%2B`，不能使用会保留 `+` 的 path escaping。

业务响应出现 `result=false` 或失败 `returnCode` 时，CLI 必须返回错误并保留原始响应体，不能把 HTTP 200 当作保存成功。

### 9.4 文档命令

```bash
cloudcc doc platform/triggers introduction
cloudcc doc platform/triggers devguide
```

## 10. 当前项目中触发器的真实约束

这是 AI 最容易忽视、但必须先接受的现实约束。

### 10.1 触发器不是普通 Java 类容器

根据当前项目的实际文件结构，触发器目录位于：

- `triggers/<对象 API 名小写>/<触发器名>/`

主类通常形如：

- `package triggers.<对象小写>.<触发器名>;`
- `public class Xxx extends CCTrigger`
- SOURCE 区域位于构造函数中

这意味着：

- 触发器天然是“薄入口”
- 不适合承载超长业务逻辑
- 更不适合在触发器里构建一整套复杂服务

## 11. AI 必须遵守的硬规则

### 11.1 只能通过 cloudcc-cli 管理触发器目录

不得手工创建或复制 `triggers/...` 目录，不得私自构造 `config.json`。

### 11.2 只能在 SOURCE 区域内写业务逻辑

AI 修改已有触发器时，只能修改：

```java
// @SOURCE_CONTENT_START
// @SOURCE_CONTENT_END
```

之间的内容。

不得破坏：

- 包路径
- 类名
- 目录结构
- `config.json`
- SOURCE 标记外的框架代码

### 11.3 不得私改 `config.json` 的身份字段

尤其不要私自改动：

- `id`
- `apiname`
- `targetObjectId`
- `schemetableName`
- `triggerTime`

这些字段应通过创建、发布、拉取、云端同步保持一致。

### 11.4 必须保留 `userInfo` 上下文

所有 SDK 能力都应围绕当前上下文用户执行。

不得：

- 硬编码用户
- 绕过 `userInfo` 伪造“当前用户”
- 用固定时区或固定组织代替上下文

### 11.5 触发器里必须显式考虑递归风险

凡是触发器里再去更新当前对象、父对象、子对象、共享对象，都要先判断是否会再次触发相关逻辑。

AI 不能默认“更新一下没事”。

### 11.6 触发器里必须显式考虑幂等性

尤其是以下动作：

- 自动生成记录
- 自动发待办
- 自动发邮件
- 自动推送外部系统
- 自动创建共享

必须先判断是否已执行过，否则很容易重复生成、重复发送、重复推送。

### 11.7 涉及时间必须优先使用 `TimeUtil`

只要出现以下需求，AI 默认当成“有时区风险”处理：

- 当前时间
- 到期日比较
- 提醒时间
- 审批时间
- 统计周期
- 格式化时间字符串
