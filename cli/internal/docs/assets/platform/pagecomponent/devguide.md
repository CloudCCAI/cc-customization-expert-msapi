# CloudCC 自定义组件开发指南

> 目标：指导开发者从 0 到 1
> 创建、调试、发布组件，并掌握常用能力（启动型组件、防止样式污染、ECharts、调用自定义类、引入第三方库/图片、日志上报、扩展属性与事件等）。

---

## CLI 与本地文件（必读）

`frontend/pagecomponents/<组件名>/` 目录、`config.json` 以及首次发布写回的 `id` 均与
**cloudcc-cli** 的创建、发布、拉取逻辑绑定。**必须通过下列命令**
完成新建组件目录、与云端同步、发布与删除；不要手工整包复制其他项目的
`frontend/pagecomponents/`、不要私自篡改 `config.json` 中的 `id`
或破坏元数据，否则容易导致编译入口错误、发布失败或与云端不一致。

**允许的做法**：在 CLI 已生成的目录内用 IDE 编辑 `.vue`、子组件与 `utils/`
等源码；与云端列表、详情、拉取、删除、发布相关的操作一律走命令。

执行命令前请确认：已完成 `cloudcc doc platform/project devguide`
中的环境初始化，项目根目录配置可用且具备 `accessToken`（发布流程还会使用
`pluginToken` / 账号密钥等，以你项目配置为准）。

业务代码只能在创建的组件中开发，不要将业务功能写入其他文件，特别是App.vue。

而且要将业务代码写入创建组件的components内的模块代码中，不要都写在组件的出口文件中。

### 命令总览（以代码实现为准）

```bash
cloudcc create pagecomponent <name>
cloudcc package pagecomponent <name> [projectPath] --dry-run
cloudcc publish pagecomponent <name>
cloudcc bind pagecomponent <projectPath> <pageApi> <componentIdOrName> [--embedded true|false] [--workspace-url <url>] [--dry-run]
cloudcc get pagecomponent [projectPath]
cloudcc detail pagecomponent <pageComponentName> [pageComponentId] [projectPath]
cloudcc detail pagecomponent "" <pageComponentId> [projectPath]
cloudcc pull pagecomponent <pageComponentNameOrId> [projectPath]
cloudcc delete pagecomponent <pageComponentNameOrId> [projectPath]
cloudcc doc platform/pagecomponent <introduction|devguide>
```

参数约定：

- `name` / `pageComponentName`：组件目录名，与 `frontend/pagecomponents/<name>/` 一致。
- `projectPath`：项目根路径；不传则使用当前工作目录。`get` 的第一个可选参数即为
  `projectPath`。
- `pageComponentNameOrId`：本地目录名，或云端组件 ID；若本地存在 `config.json` 且含
  `id`，`pull` / `delete` 会优先使用该 `id`。
- `pageComponentId`：`detail` 仅查云端时，将 `pageComponentName`
  置空：`cloudcc detail pagecomponent "" <pageComponentId>`。

### 命令作用摘要

| 命令      | 作用                                                                              |
| --------- | --------------------------------------------------------------------------------- |
| `create`  | 在 `frontend/pagecomponents/<name>/` 生成入口 `.vue`、`components/`、`config.json` 等模板         |
| `package` | 预览本次发布会采集的安全文件清单和 UMD bundle 路径，不调用远端接口 |
| `publish` | 上传已存在的 UMD bundle；首次成功且接口返回 `id` 时可写回 `config.json` |
| `bind`    | 将已有 customPage 的 `pageContent.comId` 和 `compList` 绑定到目标 pagecomponent，支持 `--dry-run` 只预览不保存 |
| `get`     | 拉取云端组件列表，标准输出为 JSON                                                 |
| `detail`  | 按目录名读本地；或按 `pageComponentId` 读云端（不传 `pageComponentName` 时）                    |
| `pull`    | 按本地 `id` 或按传入的 ID 从云端还原文件到本地 `frontend/pagecomponents/` 等路径                  |
| `delete`  | 调用接口删除云端组件                                                              |
| `doc`     | 输出 `introduction` 或 `devguide`（`devguide` 含附录 CCDK SDK 速查）              |

### 推荐操作顺序

```bash
# 1) 查看云端已有组件（可选）
cloudcc get pagecomponent .

# 2) 仅通过 create 生成目录与模板
cloudcc create pagecomponent my_plugin

# 3) 外部构建产出 UMD bundle 后发布（勿跳过 CLI 发布流程）
cloudcc package pagecomponent my_plugin . --dry-run
cloudcc publish pagecomponent my_plugin

# 4) 若已有 customPage 引用旧组件 id，发布后显式绑定
cloudcc bind pagecomponent . customer_interaction_workbench component-my-plugin --embedded true --dry-run

# 5) 与云端对齐或迁移机器时拉取
cloudcc pull pagecomponent my_plugin
# 或已知云端 ID：cloudcc pull pagecomponent <id> .

# 6) 不再使用时删除云端组件
cloudcc delete pagecomponent my_plugin
```

---

## 1. 快速上手：发布第一个组件

### 1.1 准备预览环境

页面组件的本地预览属于 Go 技能运行时之外的前端流程。使用项目约定的外部前端预览方式确认模板项目能正常运行；Go CLI 不启动或管理该预览进程。

### 1.2 创建组件

- **使用本 CLI 时**：在项目根执行 `cloudcc create pagecomponent <name>`，会在
  `frontend/pagecomponents/<name>/` 生成标准模板（推荐）。
- 或在模板工程中手动新建入口文件，**组件名称必须满足 DOM
  命名规则**，例如：`cc-com-demo.vue`
  - 推荐遵循：以 `cc-` 开头，后续使用小写单词和 `-` 连接。
  - 创建命令会生成 `component-${name}` 作为 `componentInfo.component`，因此
    `<name>` 也必须符合命名规则：仅允许小写字母和
    `-`（不允许数字），且不能使用驼峰（如 `scheduleWorkbench`）。

### 1.3 在 `App` 中引入组件

在模板项目的入口（通常是 `App.vue`
或特定页面组件）中，引入你创建的组件并挂载到页面。

### 1.4 本地预览

使用项目约定的外部前端预览流程确认组件渲染正常。

### 1.5 发布组件

**推荐（与仓库、CI 一致）**：在项目根目录执行：

```bash
cloudcc publish pagecomponent <与目录名一致的 name>
```

CLI 不会在 Go 技能内执行 Node/npm 构建；它会读取已存在的 UMD bundle 并上传。默认读取
`frontend/build/<component>.umd.min.js` 或 `frontend/build/<component>.umd.js`；也可在组件
`config.json` 中设置 `bundlePath`、`prebuiltBundlePath`、`prebuiltBundle` 或 `jsBundlePath`
指向预构建文件。首次成功且接口返回 `id` 时可能写回
`frontend/pagecomponents/<name>/config.json`。发布成功后，输出会保留发布接口响应，并附加
`customPageReferences`：当已有 customPage 仍指向旧 `pageContent.comId` 时会标记
`stale_component_reference`；当没有页面引用该组件时会标记 `not_referenced`。该检查失败不会
回滚组件发布，失败原因会写入 `customPageReferenceCheckError`。

发布依赖采集默认只包含 `frontend/pagecomponents/<name>/` 下的组件源码依赖和组件
`config.json`。根目录 `cloudcc-cli.config.json`、`.env`、token cache、`.claw-local`
等项目配置或凭据不会进入 `compContentVue` / `dependencies`。发布前可运行：

```bash
cloudcc package pagecomponent <name> . --dry-run
```

查看将被采集的文件和 bundle 路径。

注意：发布 pagecomponent 会提示 customPage 引用状态，但不会自动更新已有 customPage 的
`pageContent.comId`。若发布返回了新的组件 id，应显式执行：

```bash
cloudcc bind pagecomponent . <pageApi> <componentIdOrName> --embedded true --workspace-url <url> --dry-run
```

确认预览 payload 后，去掉 `--dry-run` 才会保存 customPage 并生成新的 renderVersion。
保存前 CLI 会校验 `pageContent[].comId`、`pageContent[].componentInfo.id` 与 `compList[].id`
一致；保存失败时错误会附带 devconsole `responseBody=<json>`，便于定位底层协议拒绝原因。

发布后验收建议显式校验最新组件 id：

```bash
cloudcc verify injectionPage . <pageApi> --expected-component <component-name> --expected-component-id <published-component-id>
```

如果 customPage 仍指向旧 `comId`，命令会返回 `stale_component_reference`；可用
`--stale-policy warning` 将 stale 结果降级为 warning 以兼容分阶段验收。

也可在 VS Code 中右键组件入口文件，选择「发布组件」（CloudCC
扩展），控制台提示成功即表示已上传至平台，可在页面编辑器中使用。

---

## 2. 启动型组件（loadModel=start）

> 通过设置
> `loadModel: "start"`，可以让组件在应用启动时加载，而不是等页面渲染时再加载，适合做布局调整、全局样式/脚本注入、全局监听等。

### 2.1 场景示例

示例：使用启动型组件隐藏详情页右侧区域（如联系人详情页右侧活动动态 Card）。

### 2.2 代码示例

```vue
<template>
  <div></div>
</template>

<script>
console.log("JS 脚本可以写在这里，会在组件加载时自动执行");

export default {
  data() {
    return {
      componentInfo: {
        // 组件唯一标识，全局唯一
        component: "cloudcc-demo-a",
        // 组件名称，在页面编辑器显示的名字
        compName: "组件名称，建议语意话，容易理解，6个字以内",
        // 组件描述信息
        compDesc: "组件描述信息",
        // 设置组件在应用启动的时候加载
        loadModel: "start"
      },
     
    };
  }
};
</script>

<style lang="scss">
// 下方样式选择器仅作示例，具体 DOM 结构请以实际页面为准
.detail001 {
  .main.left.scrollBoxUnique {
    width: 100% !important;
  }
  .layoutSwitchBox {
    display: none !important;
  }
  .right.scrollBoxUnique {
    display: none !important;
  }
}
</style>
```

说明：

- 示例中通过类名（如 `.detail001`）与子选择器控制布局与显示/隐藏。
- DOM
  结构可能随版本调整，**不要依赖不稳定选择器**，必要时配合「正确查找页面元素」最佳实践。

---

## 3. 避免样式污染（scoped）

> Vue 组件中不加 `scoped` 的 `style`
> 会导致全局样式污染。平台推荐所有组件样式默认使用 `scoped`。

### 3.1 最小示例

```vue
<template>
  <div></div>
</template>

<script>
export default {
  data() {
    return {
      componentInfo: {
        component: "cloudcc-demo-a",
        compName: "组件名称，建议语意话，容易理解，6个字以内",
        compDesc: "组件描述信息"
      },
     
    };
  }
};
</script>

<style lang="scss" scoped>
// 因为设置了 scoped，此标签下的样式仅作用于当前组件
</style>
```

要点：

- `lang="scss"`：推荐统一使用 SCSS
- `scoped`：保证样式只作用在当前组件，避免影响宿主页面与其他组件

---

## 4. 使用 ECharts 绘制图表

> 平台已集成 `echarts@5.3.2`，组件打包时会排除该库，发布到平台后会自动引用平台
> ECharts 资源。

### 4.1 代码示例（区域图）

```vue
<template>
  <div id="chartId" :style="{ width: '100%', height: '400px' }"></div>
</template>

<script>
import * as echarts from "echarts";

export default {
  data() {
    return {
      componentInfo: {
        component: "cloudcc-demo-a",
        compName: "组件名称，建议语意话，容易理解，6个字以内",
        compDesc: "组件描述信息"
      },
      myChart: null
    };
  },
  mounted() {
    this.myChart = echarts.init(document.getElementById("chartId"));
    this.myChart.setOption({
      // 此处为官方示例配置，可按需裁剪和修改
      color: ["#80FFA5", "#00DDFF", "#37A2FF", "#FF0087", "#FFBF00"],
      title: { text: "Gradient Stacked Area Chart" },
      tooltip: {
        trigger: "axis",
        axisPointer: {
          type: "cross",
          label: { backgroundColor: "#6a7985" }
        }
      },
      legend: { data: ["Line 1", "Line 2", "Line 3", "Line 4", "Line 5"] },
      toolbox: { feature: { saveAsImage: {} } },
      grid: {
        left: "3%",
        right: "4%",
        bottom: "3%",
        containLabel: true
      },
      xAxis: [{ type: "category", boundaryGap: false, data: ["Mon","Tue","Wed","Thu","Fri","Sat","Sun"] }],
      yAxis: [{ type: "value" }],
      series: [
        // 省略部分 series 配置，可按官方示例使用
      ]
    });
  }
};
</script>

<style lang="scss" scoped>
.cc-container {
  text-align: center;
  padding: 8px;
  background: goldenrod;
}
</style>
```

建议：

- 在 `beforeDestroy`/`unmounted` 生命周期中调用 `this.myChart.dispose()`
  释放资源，避免内存泄露。

---

## 5. 在组件中调用自定义类（后端接口）

> 自定义组件调用后台自定义类使用ccdk中的
> `$CCDK.CCCommon.post`，不需要自行创建其他方式。

### 5.1 代码示例

```vue
<template>
  <div class="cc-container" @click="getInfo">Hello World</div>
</template>

<script>
export default {
  data() {
    return {
      componentInfo: {
        component: "cloudcc-demo-a",
        compName: "组件名称，建议语意话，容易理解，6个字以内",
        compDesc: "组件描述信息"
      },
    };
  },
  methods: {
    getInfo() {
      const className = "AccountClass";
      const methodName = "selectAccount";
      const params = [
        { argType: "java.lang.String", argValue: "hello" },
        { argType: "java.lang.String", argValue: "world" }
      ];

      window.$CCDK.CCCommon.post(className, methodName, params)
        .then(res => {
          console.log(res);
        })
        .catch(error => {
          console.error(error);
        });
    }
  }
};
</script>

<style lang="scss" scoped>
.cc-container {
  text-align: center;
  padding: 8px;
  background: goldenrod;
}
</style>
```

要点：

- `className`、`methodName` 必须与 CloudCC 后端自定义类和方法保持一致
- `params` 为按顺序传入的参数数组，每项包括 `argType`（Java 类型）与 `argValue`

---

## 6. 引入第三方库与静态资源

### 6.1 通过 CCDK 注入第三方 JS（示例：jQuery）

> 平台提供统一 JS 注入能力 `CCDK.CCLoad.loadJs`，无需将第三方库打包进组件。

```vue
<template>
  <div></div>
</template>

<script>
// 在组件加载前引入 jQuery
window.$CCDK.CCLoad.loadJs(
  "https://cdn.bootcdn.net/ajax/libs/jquery/3.6.4/jquery.js"
);

export default {
  data() {
    return {
      componentInfo: {
        component: "cloudcc-demo-a",
        compName: "组件名称，建议语意话，容易理解，6个字以内",
        compDesc: "组件描述信息",
        // 启动加载，确保第三方库在应用启动时就注入
        loadModel: "start"
      },
     
    };
  }
};
</script>
```

注意：

- 建议优先使用平台已集成的库；如需外部 CDN，要考虑可用性与合规性。

### 6.2 使用静态资源图片

1. 在「开发者控制台 → 静态资源」中新建并上传图片
2. 在列表中复制该资源的访问地址
3. 在组件中直接使用该 URL：

```vue
<template>
  <div>
    <img
      style="width: 800px; height: 800px"
      src="https://res.lightning.cloudcc.cn/staticResource/org08f84e9c0566eaf0c/202309/16945042351857583.png"
    />
  </div>
</template>

<script>
export default {
  data() {
    return {
      componentInfo: {
        component: "cloudcc-demo-a",
        compName: "组件名称，建议语意话，容易理解，6个字以内",
        compDesc: "组件描述信息",
        loadModel: "start"
      },
     
    };
  }
};
</script>
```

---

## 7. 日志上报：CCLog（Info / Error）

> 为方便排查线上问题，平台提供 `CCDK.CCLog` 上报日志到监控平台，支持 Info 和
> Error 等级。

### 7.1 上报 Info 日志

```vue
<template>
  <div>
    <el-button @click="reportInfo">上报 Info 日志</el-button>
  </div>
</template>

<script>
export default {
  data() {
    return {
      componentInfo: {
        component: "cloudcc-demo-a",
        compName: "组件名称，建议语意话，容易理解，6个字以内",
        compDesc: "组件描述信息",
        loadModel: "start"
      },
     
    };
  },
  methods: {
    reportInfo() {
      const logInfo = {
        infoType: "debug",     // 日志级别，如 debug/info 等
        serviceName: "my app", // 服务/组件名称
        infoMessage: "描述信息" // 描述内容
      };
      window.$CCDK.CCLog.reportInfoLog(logInfo);
    }
  }
};
</script>
```

### 7.2 上报 Error 日志

```vue
<template>
  <div>
    <el-button @click="reportError">上报 Error 日志</el-button>
  </div>
</template>

<script>
export default {
  data() {
    return {
      componentInfo: {
        component: "cloudcc-demo-a",
        compName: "组件名称，建议语意话，容易理解，6个字以内",
        compDesc: "组件描述信息",
        loadModel: "start"
      },
     
    };
  },
  methods: {
    reportError() {
      const logInfo = {
        serviceName: "my app",
        errorMessage: "错误描述信息",
        printStackTraceInfo: "堆栈信息或补充说明"
      };
      window.$CCDK.CCLog.reportErrorLog(logInfo);
    }
  }
};
</script>
```

### 7.3 查看上报日志

1. 登录 CloudCC 系统
2. 点击右上角头像 → 进入「开发者平台」
3. 进入对应日志菜单查看上报记录

---

## 8. 组件销毁逻辑与性能

> 由于自定义组件的特殊渲染逻辑，**必须控制组件销毁时机**，避免长时间使用导致内存不断上涨（OOM）。

### 8.1 默认销毁策略

- 组件进入后台（用户不可见）时开始倒计时
- 倒计时结束后组件被销毁，默认销毁时间约为 20 分钟（具体以平台当前版本为准）

### 8.2 自定义销毁时间

在安全信息的高级配置中设置
`destroyTimeout`，所有打包组件共享该配置（单位：毫秒）：

```json
"devConsoleConfig": {
  "destroyTimeout": 1200000
}
```

建议：

- 对于占用资源较高的组件（如大量 ECharts
  实例、轮询等），适当缩短销毁时间，并在组件卸载前主动释放资源。

---

## 9. 扩展属性（propObj / propOption）与动态配置

> 通过扩展属性，组件可以变成「可配置组件」，由页面编辑器在属性面板中采集配置数据，再由组件使用这些配置渲染不同效果。

### 9.1 基本结构

```js
data() {
  return {
    componentInfo: {
      component: "cloudcc-demo-a",
      compName: "组件名称，建议语意话，容易理解，6个字以内",
      compDesc: "组件描述信息",
      loadModel: "start"
    },
    // 扩展属性对象：存放实际属性值
    propObj: {
      // 自定义属性时，必须写入 id
      id: "",
      src: ""
    },
    // 扩展属性配置：定义属性在页面编辑器中的展示形式
    propOption: {
      src: {
        lable: "请输入图片地址",
        type: "input"
      }
    },
   
  };
}
```

### 9.2 在组件中接收扩展属性（elePropObj）

页面编辑器会将配置好的属性通过 `props.elePropObj` 传入组件：

```vue
<template>
  <div>
    <img style="width: 800px; height: 800px" :src="elePropObj.src" />
  </div>
</template>

<script>
export default {
  props: {
    // 自定义属性对象
    elePropObj: {
      type: Object,
      default: () => ({})
    }
  },
  data() {
    return {
      componentInfo: {
        component: "cloudcc-demo-a",
        compName: "组件名称，建议语意话，容易理解，6个字以内",
        compDesc: "组件描述信息",
        loadModel: "start"
      },
      propObj: {
        id: "",
        src: ""
      },
      propOption: {
        src: {
          lable: "请输入图片地址",
          type: "input"
        }
      },
     
    };
  }
};
</script>
```

### 9.3 `propOption` 常用类型一览

> 所有配置写在 `data` 的 `propOption` 中，平台会根据 `type`
> 决定渲染何种采集控件。

- **输入框**：

```js
propOption: {
  src: {
    lable: "请输入图片地址",
    type: "input"
  }
}
```

- **日期选择器**：

```js
propOption: {
  src: {
    lable: "日期选择器",
    type: "date"
  }
}
```

- **时间选择器**：

```js
propOption: {
  src: {
    lable: "时间选择器",
    type: "time"
  }
}
```

- **图标选择器**：

```js
propOption: {
  src: {
    lable: "图标",
    type: "icon"
  }
}
```

- **颜色选择器**：

```js
propOption: {
  src: {
    lable: "颜色",
    type: "color"
  }
}
```

- **下拉框**：

```js
propOption: {
  src: {
    lable: "下拉框",
    type: "option",
    options: [
      { value: "fill", label: "fill" },
      { value: "contain", label: "contain" }
    ]
  }
}
```

- **切换器（单选按钮组）**：

```js
propOption: {
  src: {
    lable: "切换器",
    type: "radioButton",
    options: [
      { label: "default", value: "default" },
      { label: "button", value: "button" }
    ]
  }
}
```

- **开关**：

```js
propOption: {
  src: {
    lable: "开关",
    type: "switch"
  }
}
```

- **代码编辑框**：

```js
propOption: {
  src: {
    lable: "代码编辑框",
    type: "code"
  }
}
```

- **提示信息（带链接）**：

```js
propOption: {
  src: {
    lable: "提示信息",
    type: "word",
    link: "https://www.baidu.com"
  }
}
```

---

## 10. 自定义事件（events / eventsOption）

> 通过扩展事件属性，可以让页面配置层注入一段可执行代码，在组件生命周期中回调，适用于少量灵活逻辑（需非常注意安全性与维护成本）。

### 10.1 数据结构示例

```js
data() {
  return {
    componentInfo: {
      component: "test-plugin",
      compName: "test-plugin",
      compDesc: "Component description information"
    },
    events: {
      // 自定义事件：在 created 生命周期执行
      myCreated: `function created(self) {
        // self 即 this，为当前组件的 Vue 实例
      }`
    },
    eventsOption: {
      myCreated: {
        lable: "label.dev.created",
        type: "code"
      }
    },
   
  };
}
```

### 10.2 在生命周期中执行自定义事件

```js
created() {
  // 将字符串转为函数（eval 需谨慎使用，仅在可信环境启用）
  const myCreated = eval("(false || " + this.eleEventObj.myCreated + ")");
  myCreated(this);
}
```

注意：

- 此模式依赖 `eval`，有潜在安全风险，仅适用于内部受控场景
- 强烈建议对传入的代码做严格访问控制与审计

---

## 11. componentInfo 配置说明

### 11.1 标准配置

```js
componentInfo: {
  // 组件唯一标识，全局唯一：
  // 1. 必须使用小写英文
  // 2. 由两个及以上单词组成
  // 3. 使用 "-" 分隔
  component: "cloudcc-demo-01",

  // 组件显示名称，要精简易于理解，建议6个字以内
  compName: "演示使用组件",

  // 组件功能描述，建议尽量写清楚能力边界与主要用途，建议50字以内
  compDesc: "组件描述信息"
}
```

命名与描述规则（建议）：

- `compName`：组件显示名称，要精简易于理解，建议 **6 个字以内**。
- `compDesc`：组件功能描述，建议尽量写清楚能力边界与主要用途，建议 **50 字以内**。

### 11.2 高级配置：loadModel

```js
componentInfo: {
  // 加载模式，不写时默认 "lazy"
  // lazy：懒加载，仅在自定义页面实际加载组件时才加载
  // start：启动加载，在应用启动时即加载组件
  loadModel: "lazy";
}
```

推荐：

- 布局、全局脚本、埋点类组件可使用 `start`
- 普通展示组件默认使用 `lazy`

---

## 12. 进阶说明与架构提示

- 自定义组件的底层实现基于 Web Components，以支持跨框架组件开发与复用。
- 组件编译时，以组件入口文件作为构建入口，而非整个模板项目的 `main.js`：
  - `main.js` 作为本地运行模板项目的入口，可按需调整
  - 组件发布时，仅与组件入口及其依赖相关
- 引入第三方库时：
  - 优先使用平台统一注入方式（如 `CCDK.CCLoad.loadJs`）
  - 避免把大体积库直接打包进组件，减小发布包体积，提高加载性能

---

## 13. 自定义组件开发 Checklist

- [ ] 组件目录与云端同步仅通过 `cloudcc create` / `cloudcc publish` /
      `cloudcc pull` / `cloudcc delete` 等命令操作，不整包复制他人 `frontend/pagecomponents/`
      或手改 `config.json` 的 `id`
- [ ] 组件文件名与 `componentInfo.component` 命名符合规范（小写 + `-`
      分隔，全局唯一）
- [ ] 样式默认使用 `lang="scss" scoped`，避免样式污染
- [ ] 使用平台提供的 `CCDK` 能力调用后端、自定义类、日志上报等，不手写接口 URL
- [ ] 如需第三方库，优先使用 `CCLoad.loadJs` 或平台已集成的库
- [ ] 合理设置 `loadModel`（`lazy`/`start`），避免不必要的启动时加载
- [ ] 如使用 ECharts/定时器/监听等，注意在组件销毁时释放资源
- [ ] 合理使用 `propObj` / `propOption` / `elePropObj` 提升组件可配置性
- [ ] 如启用 `events` 动态代码，确保仅在可信场景使用，并有审计与限制
- [ ] 发布前已通过项目约定的外部前端预览流程完整验证主要功能与交互

---

## 14. 命令使用说明（与实现一致）

文首「CLI 与本地文件（必读）」为总览；本节按子命令列出行为细节。

### 14.1 create

```bash
cloudcc create pagecomponent <name>
```

- 创建 `frontend/pagecomponents/<name>/` 目录、入口 `<name>.vue`、`components/` 与 `config.json`
  模板。

### 14.2 publish

```bash
cloudcc publish pagecomponent <name>
```

- 读取已存在的 UMD bundle 并上传到云端；默认查找 `frontend/build/<component>.umd.min.js`
  和 `frontend/build/<component>.umd.js`。
- 如 bundle 位于其他路径，在 `frontend/pagecomponents/<name>/config.json` 中设置
  `bundlePath`、`prebuiltBundlePath`、`prebuiltBundle` 或 `jsBundlePath`。
- Go CLI 不调用 Node/npm；前端构建必须在 Go 技能运行前完成。
- 首次发布成功且响应包含 `id` 时，会写回 `frontend/pagecomponents/<name>/config.json`（若此前无
  `id`）。

### 14.3 get

```bash
cloudcc get pagecomponent [projectPath]
```

- 分页请求云端组件列表，标准输出为 JSON 数组。
- `projectPath` 可选，用于解析项目根下的 `cloudcc-cli` 配置。

### 14.4 detail

```bash
cloudcc detail pagecomponent <pageComponentName> [pageComponentId] [projectPath]
cloudcc detail pagecomponent "" <pageComponentId> [projectPath]
```

- 提供 `pageComponentName`（且非空）：读取本地 `frontend/pagecomponents/<pageComponentName>/` 的 `config.json`
  与入口 `.vue`。
- 将 `pageComponentName` 置为空并传 `pageComponentId`：仅从云端查询详情（`projectPath`
  仍用于配置）。
- 需要指定项目根且保留「仅本地名」时，可用占位：`cloudcc detail pagecomponent <pageComponentName> "" <projectPath>`（中间空串表示无
  `pageComponentId`）。

### 14.5 pull

```bash
cloudcc pull pagecomponent <pageComponentNameOrId> [projectPath]
```

- 若 `frontend/pagecomponents/<输入>/config.json` 存在且含 `id`，按该 `id` 拉取。
- 否则将输入视为云端组件 ID，拉取后在 `frontend/pagecomponents/<归一化名称>/` 等处落盘并更新
  `config.json`。

### 14.6 delete

```bash
cloudcc delete pagecomponent <pageComponentNameOrId> [projectPath]
```

- 若本地 `frontend/pagecomponents/<name>/config.json` 存在且含 `id`，优先按该 `id` 调删除接口。
- 否则将参数视为云端 ID 直接删除。

### 14.7 doc

```bash
cloudcc doc platform/pagecomponent introduction
cloudcc doc platform/pagecomponent devguide
```

- 仅支持 `introduction` 与 `devguide`；`devguide` 会在正文后拼接附录「CCDK SDK
  速查」。
