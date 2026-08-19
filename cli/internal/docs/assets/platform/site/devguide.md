# CloudCC Site 开发规范

## 1. 目标与定位

本规范用于 CloudCC Site（完整站点）开发，目标是交付可上线、可维护、可扩展的前端应用，而不是仅可演示的页面。
需要在项目根目录创建 `site` 文件夹，再在其中创建具体 Site 项目文件夹进行开发。

## 2. 技术栈（强制）

Site 开发技术栈统一为：

- React
- TypeScript
- Vite
- Tailwind CSS
- shadcn/ui
- TanStack Query

不允许替换为其他同类主栈（如 Next.js、Vue、Ant Design、SWR）作为默认方案；如有特殊场景需先评审。

## 3. 设计与体验规范

### 3.1 先定设计方向，再写代码

在开始开发前，必须先明确：

- 业务目的：页面解决什么问题、服务谁。
- 视觉调性：明确选择一种风格方向（如极简、编辑感、工业感、科技感等），避免“默认模板风”。
- 差异化记忆点：必须至少有 1 个可感知的视觉或交互亮点。

### 3.1.1 设计产出标准（强制）

Site 方案必须满足以下产出标准：

- 有明确且大胆的视觉方向，不接受“通用后台模板风”。
- 字体与配色有辨识度，不能只用默认系统组合。
- 存在至少一处高质量视觉细节或动效亮点，且不影响性能与可用性。
- 实现必须结合业务上下文（目标用户、行业语境、场景约束），不做脱离场景的通用拼装。

### 3.2 视觉系统约束

- 字体、颜色、圆角、阴影、间距必须抽象为设计令牌（Design Tokens），统一在主题层管理。
- 不允许全站散落硬编码色值和随意字号。
- 页面布局要有层级和节奏，避免纯列表堆砌。

### 3.3 动效与交互

- 动效服务于信息表达，不做无意义炫技。
- 首屏允许有一处高质量入场动效；其余交互采用轻量微动效（hover/focus/loading/empty/success/error）。
- 必须保证动效可降级：低性能设备和弱网下不影响可用性。

## 4. 工程规范

### 4.1 React + TypeScript

- 全量 TypeScript，禁止新增 `any`（确有必要时需注释说明）。
- 组件职责单一，避免超大组件；通用能力沉淀到 `components`、`hooks`、`lib`。
- 组件对外 API（props）需稳定、可复用、可测试。

### 4.2 Vite

- 使用 Vite React + TS 模板初始化。
- 保持构建配置最小化，优先约定优于配置。
- 生产构建必须无报错、无阻塞警告。

### 4.3 Tailwind CSS

- 统一使用 Tailwind 工具类，不混用第二套原子样式方案。
- 禁止页面中大量重复长类名，公共模式需抽象为组件或样式变量。
- 主题色、间距、断点统一在 Tailwind 配置层维护。

### 4.4 shadcn/ui

- 基础 UI 组件优先复用 shadcn/ui，减少重复造轮子。
- 二次封装时保留可访问性语义（aria、键盘可达、焦点管理）。
- 不允许直接复制粘贴第三方组件后长期不维护。

### 4.5 TanStack Query

- 所有服务端状态统一由 TanStack Query 管理（查询、缓存、失效、重试）。
- 查询 key 必须稳定、可推导，禁止随意拼接导致缓存失效。
- 变更操作（mutation）后必须明确刷新策略（invalidate/update optimistic）。

## 5. 数据与资源规范

- JS、CSS、图片、字体等资源可上传 CloudCC 静态资源并通过静态资源 URL 引用。
- 静态资源命名需可读可追踪（模块名-用途-版本），禁止随机命名。
- 线上资源必须可回滚，禁止直接覆盖不可追溯文件。

## 6. 可用性与质量基线

- 必须覆盖加载态、空态、错误态、无权限态。
- 必须支持基础响应式（至少桌面端 + 常见移动宽度）。
- 关键交互需可键盘操作，表单项需有明确标签与错误提示。
- 首屏和关键列表页必须可观测（错误日志、请求失败可定位）。

## 7. 禁止项

- 禁止“脚手架默认样式”直接上线。
- 禁止在业务代码中散落 mock 常量且无清理。
- 禁止出现不可解释的魔法数字与硬编码接口路径。
- 禁止把组件文档与实现长期脱节。

## 8. 交付验收清单（上线前）

- 技术栈符合：React + TS + Vite + Tailwind + shadcn/ui + TanStack Query。
- 页面与组件通过基础自测：加载、空、错、权限、提交成功/失败。
- 构建通过且无阻塞问题。
- 静态资源路径、命名、版本可追溯。
- 关键页面视觉风格统一，存在明确记忆点，不是通用模板感。

## 9. 需求输入模板（给研发/AI）

为保证实现质量，需求描述至少包含以下信息：

- 页面类型：如仪表盘、落地页、设置页、数据管理页。
- 目标用户与业务目标：谁在用、要解决什么问题。
- 风格方向：如极简、科技感、工业感、杂志感等（必须明确一种主方向）。
- 技术约束：是否必须支持移动端、是否有性能预算、是否有无障碍要求。

示例：

- “创建一个面向销售团队的业绩仪表盘，风格偏科技感，桌面优先，首屏加载 2 秒内，必须支持键盘操作与错误态展示。”

## 10. AI 执行规范（强制）

- 不允许只基于文字规范给出“模板化”方案；必须结合示例图片中的风格特征给出可落地实现。
- 若当前环境无法读取图片，必须明确说明阻塞点，并先给出最短可执行替代路径（如让用户补齐图片或指定可访问路径）。

## 11. 附录：Frontend Aesthetics 可执行规范（内嵌版）

### 11.1 核心目标

- 默认生成容易趋向保守和模板化，需要通过明确提示引导出更有辨识度的页面。
- 目标不是“看起来像 AI 自动生成”，而是“可上线、可识别、有设计判断”的真实产品界面。

### 11.2 三个稳定有效的提示策略

1. 明确约束设计维度（字体、色彩、动效、背景）
2. 提供风格灵感来源（主题方向，不要过度限制）
3. 显式禁止默认套路（常见字体、配色、布局）

### 11.3 全量美学提示词（推荐基线）

```text
<frontend_aesthetics>
You tend to converge toward generic, "on distribution" outputs. In frontend design, this creates what users call the "AI slop" aesthetic. Avoid this: make creative, distinctive frontends that surprise and delight. Focus on:

Typography: Choose fonts that are beautiful, unique, and interesting. Avoid generic fonts like Arial and Inter; opt instead for distinctive choices that elevate the frontend's aesthetics.

Color & Theme: Commit to a cohesive aesthetic. Use CSS variables for consistency. Dominant colors with sharp accents outperform timid, evenly-distributed palettes. Draw from IDE themes and cultural aesthetics for inspiration.

Motion: Use animations for effects and micro-interactions. Prioritize CSS-only solutions for HTML. Use Motion library for React when available. Focus on high-impact moments: one well-orchestrated page load with staggered reveals (animation-delay) creates more delight than scattered micro-interactions.

Backgrounds: Create atmosphere and depth rather than defaulting to solid colors. Layer CSS gradients, use geometric patterns, or add contextual effects that match the overall aesthetic.

Avoid generic AI-generated aesthetics:
- Overused font families (Inter, Roboto, Arial, system fonts)
- Clichéd color schemes (particularly purple gradients on white backgrounds)
- Predictable layouts and component patterns
- Cookie-cutter design that lacks context-specific character

Interpret creatively and make unexpected choices that feel genuinely designed for the context. Vary between light and dark themes, different fonts, different aesthetics. You still tend to converge on common choices (Space Grotesk, for example) across generations. Avoid this: it is critical that you think outside the box!
</frontend_aesthetics>
```

#### A) Typography Only

```text
<use_interesting_fonts>
Typography instantly signals quality. Avoid using boring, generic fonts.

Never use: Inter, Roboto, Open Sans, Lato, default system fonts

Impact choices:
- Code aesthetic: JetBrains Mono, Fira Code, Space Grotesk
- Editorial: Playfair Display, Crimson Pro, Fraunces
- Startup: Clash Display, Satoshi, Cabinet Grotesk
- Technical: IBM Plex family, Source Sans 3
- Distinctive: Bricolage Grotesque, Obviously, Newsreader

Pairing principle: High contrast = interesting. Display + monospace, serif + geometric sans, variable font across weights.

Use extremes: 100/200 weight vs 800/900, not 400 vs 600. Size jumps of 3x+, not 1.5x.

Pick one distinctive font, use it decisively.
</use_interesting_fonts>
```

#### B) Theme Constraint（示例：Solarpunk）

```text
<always_use_solarpunk_theme>
Always design with Solarpunk aesthetic:
- Warm, optimistic color palettes (greens, golds, earth tones)
- Organic shapes mixed with technical elements
- Nature-inspired patterns and textures
- Bright, hopeful atmosphere
- Retro-futuristic typography
</always_use_solarpunk_theme>
```

### 11.6 标准执行顺序（强制）

1. 先定义页面目标、受众、主风格、记忆点
2. 选用“全量美学提示词”或“隔离提示词”
3. 按 React+TS+Vite+Tailwind+shadcn/ui+TanStack Query 落地
4. 补全状态页与可访问性
5. 对照验收清单做上线前检查
6. 不要使用暗色风格，使用清新自然的风格
