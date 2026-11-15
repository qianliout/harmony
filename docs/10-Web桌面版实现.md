# Web 桌面版实现

本文档把 `docs/project/web-prototype.html` 落成可执行的实现方案。原型是视觉与交互的唯一依据；业务规则仍以 `docs/project/fen.md` 为准，接口契约以 `docs/02-后端设计.md` 为准。

本文档描述目标方案与落地顺序，不代表代码已经完成。当前仓库仍是手机页面（H5 与小程序共用 430px 画布）加旧识别宽屏页的结构，按第 11 节分阶段改造。

## 1. 结论与边界

### 1.1 结论

1. **H5 改为独立的桌面表现层**：左侧导航 + 顶栏 + 内容区，页面用表格、抽屉、对话框和全局搜索承载。小程序继续使用现有手机页面，不改交互。
2. **只做一套业务核心**：`api/`、`api/types.ts`、`stores/`、`utils/`、`styles/tokens.scss` 两端共用；后端只有一处数据源。
3. **H5 页面与小程序页面分离，但同在一个 uni-app 工程**：新增 `frontend/src/sub-web/`，用条件编译只进入 H5 产物；小程序页面保持原样。
4. **后端只做加法**：补 5 处查询与聚合字段，不改表结构、不改既有响应字段含义。详见第 10 节。

### 1.2 不在本次范围

- 不修改任何业务规则、金额口径、回礼规则、合并规则、识别规则。
- 不新建独立后端，不引入第二份数据。
- 不做离线账本、不做多用户协作。
- 不重写小程序的页面与交互。
- 不为了桌面版引入 Vue Router、Element Plus、Ant Design 等第二套路由或组件库。

### 1.3 与现有文档的冲突

- `03-前端交互与UI规范.md` 中"H5 在桌面浏览器中显示为 430px 手机画布"的规定，被本文档取代；其余视觉 token、金额/日期/状态规则仍然有效。
- `08-Web端能力与宽屏布局.md` 中"只有识别页宽屏"的布局结论被本文档取代；识别核对、上传增强、识别历史、键盘等细节继续有效。
- `fen.md` 第 7.1 节"页面按手机屏幕尺寸设计，Web 端同样按手机尺寸布局"与最新原型冲突。本文档按原型实现桌面版，业务验收条目不变；定稿时需要同步修订该条需求描述。

## 2. 现有实现盘点

| 能力 | 现有代码 | Web 桌面版处理 |
| --- | --- | --- |
| 首页 | `frontend/src/pages/index/index.vue` | 小程序首页保持原样；H5 新增 `pages/web/index` 桌面工作台 |
| 记录列表 | `frontend/src/sub-record/list.vue` | 新增桌面表格与筛选栏，复用查询映射 |
| 亲友列表 / 详情 | `frontend/src/pages/person/list.vue`、`sub-person/detail.vue` | 新增桌面表格与时间线 |
| 礼簿列表 / 详情 | `frontend/src/pages/book/list.vue`、`sub-book/detail.vue` | 新增桌面表格、tabs、连续录入 |
| 事项 | 小程序在礼簿页内分段展示，详情在 `sub-event/detail.vue` | Web 增加独立事项列表页 |
| 回礼清单 | `frontend/src/sub-misc/pending.vue` | 新增桌面分组表格 |
| 统计 | `frontend/src/sub-misc/stats.vue` | 新增桌面卡片、柱条与按月图 |
| 整理纸本标注 | `frontend/src/sub-record/notes.vue` | 新增桌面批量表格 |
| 按页核对 | `frontend/src/sub-book/pages.vue` | 新增跨礼簿汇总表 |
| 导入导出 | `frontend/src/sub-misc/transfer.vue` | 复用 `api/transfer.ts` 与校验流程 |
| 设置 | `frontend/src/sub-misc/settings.vue` | 新增桌面分区卡片 |
| 识别 | `frontend/src/sub-recognize/index.vue` | 复用 `useReviewState`、`ImageViewer` 等，新增桌面核对视图 |
| 登录 | `frontend/src/sub-auth/login.vue` | 新增独立全屏登录页，复用扫码/令牌接口 |
| 样式 token | `frontend/src/styles/tokens.scss`、`palette.ts` | 追加桌面布局 token，不新增色值体系 |
| 基础组件 | `Money`、`EventSeal`、`PersonAvatar`、`DirTag`、`StatusPill`、`EmptyState`、`Skeleton` | 直接复用；桌面表格另建 `WebTable` 等 |

现有后端已覆盖原型绝大多数查询与写入。下一节列出的差异需要在后端补字段或补参数，均不涉及建表。

## 3. 原型与现状的差异

| 原型能力 | 现状 | 处理 |
| --- | --- | --- |
| 左侧导航、顶栏、面包屑 | 不存在 | 新增 `WebShell` |
| 记录表格底部"收/送/未回礼"汇总 | `/records` 只返回当前页数据与总数 | 后端返回筛选结果汇总 `custom.summary` |
| 亲友文本筛选（姓名/拼音首字母） | `/records` 只支持 `personId` | 后端增加 `personKeyword` |
| 事项列表独立页 | 小程序合并在礼簿页 | 新增 `sub-web/events`，复用 `/events` |
| 事项详情"我回了"分段 | `/views/event-summary` 没有回礼标记 | 后端补 `repayCount`、`repayCent`、`isRepay` |
| 统计"全部"年份 | `/stats/year` 只支持单年 | 后端支持 `year=all` |
| 统计按事项类型收/送双柱 | `EventTypeStat` 只有合计 | 后端补 `inCent`、`outCent` |
| 统计"往来最多的人"最近往来 | `PersonStat` 没有最近日期 | 后端补 `lastDate` |
| 回礼清单每人明细 | `PendingPerson` 只有合计 | 后端补 `records` 或前端按人懒加载 |
| 记录列表导出当前结果 | `/export` 只支持 book/event/person | 已扩展为接受记录筛选参数，导出当前结果 |
| 桌面表格、抽屉、命令面板、快捷键 | 不存在 | 新增 H5 组件与 shell |

## 4. 架构

### 4.1 共享核心 + 双表现层

```text
┌─────────────────────────────────────┐
│ H5 桌面表现层                        │
│ pages/web/index (工作台)             │
│ sub-web/* (记录、亲友、礼簿、识别…)   │
│ WebShell：侧栏、顶栏、抽屉、命令面板   │
└──────────────────┬──────────────────┘
                   │
┌──────────────────▼──────────────────┐
│ 共享核心                             │
│ api/*、api/types.ts、stores/*        │
│ utils/*、styles/tokens.scss、基础组件 │
└──────────────────┬──────────────────┘
                   │ /api/v1
┌──────────────────▼──────────────────┐
│ Go 后端 + SQLite 单文件              │
└──────────────────▲──────────────────┘
                   │
┌──────────────────┴──────────────────┐
│ 微信小程序表现层                     │
│ pages/*、sub-*/*（保持现状）          │
└─────────────────────────────────────┘
```

约束：

- 业务校验、金额换算、软确认、回礼关联、纸本标注解析等逻辑只能有一份实现，放在 `composables/` 或 `utils/`，页面只做展示与编排。
- H5 特有代码必须在 `#ifdef H5` 内；小程序产物中不得出现 `sub-web`、`WebShell`、`window`、`document`。
- 两端页面可以长得不同，但读写同一批接口，统计结果必须一致。

### 4.2 为什么不是另外两种方案

| 方案 | 不采用的原因 |
| --- | --- |
| 把现有手机页面全部改成响应式桌面布局 | 现有页面按 430px 与 `uni` 组件设计，改成表格 + 抽屉等于重写；且手机端的交互（底部操作栏、FAB、下拉刷新）会被破坏 |
| 新建 `frontend-web/` 独立 Vue 应用 | 会出现第二套 `api/`、`utils/`、token、构建与发布流程，两端功能对等只能靠人工同步，违反 `fen.md` 第 6 节第 16 条 |
| 用 iframe 把旧手机页面嵌进桌面壳 | 路由、登录、抽屉、剪贴板、识别页都难以协同；且视觉上仍是手机画布 |

采用同一 uni-app 工程内的 H5 专用页面，可以复用请求层与类型，同时让桌面 UI 不受小程序组件限制。

### 4.3 H5 启动与页面声明

1. 新增 `pages/web/index.vue` 作为 H5 首页，渲染 `<WebShell nav="home"><HomeView /></WebShell>`；`pages/index/index.vue` 保持小程序首页原样，不移动、不改生命周期。
   - `pages.json` 的 `pages` 数组按平台二选一：H5 只保留 `pages/web/index`，小程序保留 `pages/index/index`、`pages/person/list`、`pages/book/list`、`pages/mine/index`。
2. 新增 `frontend/src/sub-web/` 分包，只包含 H5 页面。除登录页外，每个页面是薄壳：

```vue
<script setup lang="ts">
import WebShell from '@/web/shell/WebShell.vue'
import RecordsView from '@/web/views/RecordsView.vue'
</script>

<template>
  <WebShell nav="records" title="记录" crumbs="全部记录">
    <RecordsView />
  </WebShell>
</template>
```

3. `frontend/src/pages.json` 用条件编译拆分页面：

```jsonc
{
  "pages": [
    // #ifdef H5
    { "path": "pages/web/index", "style": { "navigationStyle": "custom" } }
    // #endif
    // #ifndef H5
    { "path": "pages/index/index", "style": { "navigationBarTitleText": "人情往来账本" } },
    { "path": "pages/person/list", "style": { "navigationBarTitleText": "亲友" } },
    { "path": "pages/book/list", "style": { "navigationBarTitleText": "礼簿" } },
    { "path": "pages/mine/index", "style": { "navigationBarTitleText": "我的" } }
    // #endif
  ],
  "subPackages": [
    // #ifdef H5
    {
      "root": "sub-web",
      "pages": [
        { "path": "records", "style": { "navigationStyle": "custom" } },
        { "path": "people", "style": { "navigationStyle": "custom" } },
        { "path": "person", "style": { "navigationStyle": "custom" } },
        { "path": "events", "style": { "navigationStyle": "custom" } },
        { "path": "event", "style": { "navigationStyle": "custom" } },
        { "path": "books", "style": { "navigationStyle": "custom" } },
        { "path": "book", "style": { "navigationStyle": "custom" } },
        { "path": "repay", "style": { "navigationStyle": "custom" } },
        { "path": "stats", "style": { "navigationStyle": "custom" } },
        { "path": "notes", "style": { "navigationStyle": "custom" } },
        { "path": "pages", "style": { "navigationStyle": "custom" } },
        { "path": "settings", "style": { "navigationStyle": "custom" } },
        { "path": "recognize", "style": { "navigationStyle": "custom" } },
        { "path": "recognize-history", "style": { "navigationStyle": "custom" } },
        { "path": "login", "style": { "navigationStyle": "custom" } }
      ]
    }
    // #endif
    // #ifndef H5
    {
      "root": "sub-person",
      "pages": [
        { "path": "detail", "style": { "navigationBarTitleText": "亲友详情" } },
        { "path": "edit", "style": { "navigationBarTitleText": "亲友" } }
      ]
    },
    // …sub-book、sub-event、sub-record、sub-misc、sub-auth(confirm) 原样保留
    // #endif
  ]
  // #ifndef H5
  ,
  "tabBar": { /* 现有小程序 tabBar 原样保留 */ }
  // #endif
}
```

互斥的 `#ifdef H5` / `#ifndef H5` 块之间不要写逗号，否则预处理后会出现前导逗号或尾随逗号。每个块内部自己保证 JSON 逗号正确；改完必须分别构建 H5 与小程序验证。

4. `App.vue`：
   - 删除 H5 的 430px 居中规则与 `html.layout-wide` 规则；H5 根节点铺满视口。
   - 保留 `page` 背景与字体规则。
   - H5 `onLaunch` 继续调用 `auth.loadMe()`；401 跳转目标从 `/sub-auth/login` 改为 `/sub-web/login`。

5. 现有 `sub-auth/login.vue`、`sub-recognize/index.vue`、`sub-recognize/history.vue` 是旧 H5 页面：源码保留，但不再在 `pages.json` 声明，不进入 H5 产物；`sub-recognize/components` 与 `useReviewState` 继续被新桌面识别页复用。

### 4.4 路由

H5 使用 uni-app 自带 hash 路由，不引入 Vue Router。URL 形态：

```text
#/pages/web/index                  工作台
#/sub-web/records                  记录
#/sub-web/people                   亲友
#/sub-web/person?id=3              亲友详情
#/sub-web/events                   事项
#/sub-web/event?id=5               事项详情
#/sub-web/books                    礼簿
#/sub-web/book?id=2&tab=cont       礼簿详情
#/sub-web/repay                    回礼清单
#/sub-web/stats?year=2026          统计
#/sub-web/notes                    整理纸本标注
#/sub-web/pages                    按页核对
#/sub-web/settings                 设置
#/sub-web/recognize                识别
#/sub-web/recognize-history        识别历史
#/sub-web/login                    登录
```

导航规则：

- 侧栏切换一级页面用 `uni.redirectTo`，不堆页面栈；浏览器后退仍然可用。
- 列表进入详情、亲友、事项、礼簿用 `uni.navigateTo`，返回时保留上一页筛选与滚动位置。
- 详情页返回用 `uni.navigateBack`。
- H5 不显示 tabBar；小程序 tabBar 不进入 H5 产物。
- 页面查询参数只作为初始上下文：`records?personId=&bookId=&eventId=&bookCategory=&eventType=&direction=&status=&itemKind=&from=&to=&keyword=&page=`；`stats?year=`；`book?tab=`；`person?id=`；`event?id=`；`recognize?taskId=`。
- 抽屉、对话框、命令面板不占路由。需要从别处直接打开记录详情时，用 `records?recordId=`，列表加载完成后打开抽屉。

### 4.5 状态与数据流

Pinia 只存跨页面或跨视图共享的状态，列表数据仍由页面自己请求：

| store | 内容 |
| --- | --- |
| `auth` | 现有登录态、`isAdmin`、令牌 |
| `dict` | 礼簿类别、名目字典缓存 |
| `context` | 上次礼簿/事项、快捷金额、设置项 |
| `webUi`（`stores/web.ts`） | 当前命令面板、抽屉/对话框栈、toast、`revision` |

`webUi` 的职责：

```ts
openEntry(context?: { personId?: number; eventId?: number; bookId?: number }): void
openRecord(id: number): void
openRepay(originIds: number[]): void
openPaperAnno(id: number): void
openMerge(sourceId?: number): void
openSearch(): void
closeTop(): void
toast(message: string): void
markChanged(): void          // 写操作成功后 revision++
```

- 所有写操作成功后调用 `markChanged()`；当前视图 `watch(() => ui.revision, reload)`，避免各页面互相通知。
- 顶栏"记一笔"由 `WebShell` 直接调用 `openEntry()`。
- 页面跳转时 `WebShell` 的 `onShow` 关闭残留抽屉，避免跨页遮罩。
- 字典在应用启动时 `dictStore.ensure()` 加载一次；设置项在 shell 首次显示时 `contextStore.load()`。

数据请求统一走 `api/*.ts`，页面不直接调用 `http`。新增一个轻量组合式函数：

```ts
// web/composables/useAsync.ts
export function useAsync<T>(loader: () => Promise<T>, initial: T) {
  const data = ref(initial)
  const loading = ref(false)
  const error = ref('')
  async function reload() { /* 置 loading、catch ApiError、finally */ }
  return { data, loading, error, reload }
}
```

### 4.6 平台边界

- `sub-web/`、`web/` 目录中的文件允许使用标准 HTML 标签（`table`、`button`、`input`、`dialog`）与 `window`/`document`，因为这些文件只进入 H5 产物。
- 共享 `api/`、`utils/`、`stores/` 中不得出现 `window`/`document`；需要浏览器能力的代码用 `#ifdef H5` 包住。
- 小程序构建后检查：
  - `frontend/dist/build/mp-weixin` 中搜不到 `WebShell`、`sub-web`、`webUi`、`layout-wide`；
  - 搜不到识别相关入口与 `recognize` 页面。

## 5. 目录结构

```text
frontend/src/
├── pages/index/
│   ├── index.vue                 H5/Mp 平台分支
├── sub-web/                      H5 专用页面（薄壳）
│   ├── records.vue
│   ├── people.vue
│   ├── person.vue
│   ├── events.vue
│   ├── event.vue
│   ├── books.vue
│   ├── book.vue
│   ├── repay.vue
│   ├── stats.vue
│   ├── notes.vue
│   ├── pages.vue
│   ├── settings.vue
│   ├── recognize.vue
│   ├── recognize-history.vue
│   └── login.vue
├── web/
│   ├── shell/
│   │   ├── WebShell.vue
│   │   ├── SideNav.vue
│   │   ├── TopBar.vue
│   │   ├── CommandPalette.vue
│   │   ├── NotesDrawer.vue
│   │   └── ToastHost.vue
│   ├── views/                   与 sub-web 页面一一对应的内容组件
│   ├── overlays/
│   │   ├── EntryDrawer.vue
│   │   ├── RecordDrawer.vue
│   │   ├── RepayDrawer.vue
│   │   ├── PaperAnnoDrawer.vue
│   │   ├── MergeDialog.vue
│   │   └── ConfirmDialog.vue
│   ├── components/
│   │   ├── WebIcon.vue
│   │   ├── PageHead.vue
│   │   ├── SectionHead.vue
│   │   ├── WebTable.vue
│   │   ├── Pager.vue
│   │   ├── FilterSelect.vue
│   │   ├── Segmented.vue
│   │   ├── StatCard.vue
│   │   ├── BarRow.vue
│   │   ├── MonthChart.vue
│   │   └── PersonCell.vue
│   ├── composables/
│   │   ├── useWebHotkeys.ts
│   │   └── useFocusTrap.ts
│   ├── styles/web.scss           桌面布局、表格、抽屉基础样式
│   ├── icons.ts                  原型 SVG 路径表
│   └── notes.ts                  逐页"交互说明"文案
├── stores/
│   ├── web.ts                    useWebUiStore：命令面板、抽屉栈、toast、revision
│   ├── auth.ts                   现有
│   ├── dict.ts                   现有
│   └── context.ts                现有
├── composables/                  两端共享逻辑
│   ├── useRecordQuery.ts         新增，筛选映射
│   ├── useRecordSummary.ts       新增，记录汇总
│   ├── useEntryForm.ts           新增
│   ├── useContinuousEntry.ts     新增
│   ├── useRepayForm.ts           新增
│   ├── usePaperAnno.ts           新增
│   ├── useRecognizeFlow.ts       新增
│   ├── useConfirmSubmit.ts       现有
│   ├── useHotkeys.ts             现有，扩展为识别与全局快捷键
│   └── usePagedList.ts           现有
└── styles/tokens.scss            追加桌面 token
```

## 6. 设计 token 与布局

### 6.1 追加 token

在 `frontend/src/styles/tokens.scss` 的 `page, :root` 中追加；桌面新增值全部有出处，不在页面里写色值：

| token | 值 | 用途 |
| --- | --- | --- |
| `--c-side` | `#fbf8f4` | 侧栏底色 |
| `--c-hover` | `#fcfaf7` | 表格行、列表行悬停 |
| `--c-on-primary` | `#ffffff` | 朱砂卡文字 |
| `--c-on-primary-2` | `rgba(255, 255, 255, .8)` | 朱砂卡次要文字 |
| `--c-paper` | `#f3e9d6` | 纸本原图/标注底色 |
| `--c-paper-ink` | `#2b2420` | 纸本文字 |
| `--c-paper-line` | `#d9c7a6` | 纸本分隔线 |
| `--side-w` | `224px` | 侧栏宽 |
| `--top-h` | `64px` | 顶栏高 |
| `--gutter-x` | `32px` | 内容左右边距 |
| `--web-max` | `1280px` | 内容最大宽 |

`--c-none`、`--c-pending`、`--c-repaid`、`--c-in`、`--c-out` 已有，不重复定义。H5 的 `body` 使用 `--c-bg`，H5 根节点不再限 430px。

### 6.2 壳层尺寸

```text
┌──────────────┬──────────────────────────────────────────────┐
│ 侧栏 224      │ 顶栏 64：面包屑 / 搜索（⌘K） / 交互说明 / 记一笔 │
│ 品牌          ├──────────────────────────────────────────────┤
│ 导航          │ 内容区：左右 32，最大 1280，上下 24/80         │
│ 分组标题      │                                              │
│ 徽标          │                                              │
│ 本机用户      │                                              │
└──────────────┴──────────────────────────────────────────────┘
```

- 侧栏 `position: sticky; top: 0; height: 100vh; overflow-y: auto`。
- 顶栏 `position: sticky; top: 0; z-index: 20`，背景 `--c-bg`，底部 1px `--c-divider`。
- 内容区滚动由页面整体滚动承担，不做内部滚动容器，避免表格与吸底条冲突。
- `≤ 960px`：侧栏改为顶部横向导航，品牌文字、用户区、分组标题隐藏；内容单列；表格横向滚动；抽屉宽度 `100vw`；hero 快捷按钮换行。
- `≤ 520px`：筛选栏换行，统计卡两列，表格保留最小宽度并横向滚动。

### 6.3 字体与动效

- 正文 14/22，表格 13/20，页面标题 22/30，区块标题 17/24，统计数字 24/32 或 32/40，使用 `$font-num`。
- 金额、页码、笔数使用 `font-variant-numeric: tabular-nums`。
- 抽屉 260ms、遮罩 200ms、命令面板 180ms，缓动 `cubic-bezier(.2,.8,.2,1)`；`prefers-reduced-motion: reduce` 时全部取消。
- 所有可点击元素有 `:focus-visible` 描边，抽屉/对话框打开时焦点进入并循环，关闭后焦点回到触发元素。

## 7. 组件设计

### 7.1 壳层

| 组件 | 职责 |
| --- | --- |
| `WebShell` | 接收 `nav`、`title`、`crumbs`；渲染侧栏、顶栏、内容插槽、全局抽屉/对话框/toast；监听 `webUi` |
| `SideNav` | 导航分组、当前项高亮、待回礼/纸本标注徽标；点击 `redirectTo` |
| `TopBar` | 面包屑、搜索按钮（`⌘K`）、交互说明、记一笔 |
| `CommandPalette` | 全局搜索、分组结果、键盘上下选择、回车打开 |
| `NotesDrawer` | 当前页面交互说明与全局快捷键 |
| `ToastHost` | 3 秒自动消失的底部提示 |

导航项与原型一致：

```text
工作台 / 记录 / 亲友 / 事项 / 礼簿 / 回礼清单 / 统计
纸本录入：识别礼簿 / 整理纸本标注 / 按页核对
设置
```

徽标：

- 回礼清单：`GET /views/pending/owed` 的 `totalCount`。
- 整理纸本标注：`GET /paper-notes/pending` 的条数。
- 徽标在 shell 首次显示与 `revision` 变化时刷新，60 秒内不重复请求。

### 7.2 基础组件

| 组件 | 规格 |
| --- | --- |
| `WebIcon` | 18×18，线性图标，`stroke: currentColor`；图标路径来自原型的 symbol 表，放在 `web/icons.ts` |
| `PageHead` | 标题 22/30、元信息 13、右侧工具区；支持头像/印章插槽 |
| `SectionHead` | 标题 17、副标题、右侧工具 |
| `WebTable` | 列配置 `{ key, label, width, align, className }`；Sticky 表头；行悬停；点击行；空状态插槽；横向滚动；不实现排序（原型没有） |
| `Pager` | 上一页/下一页 + `当前 / 总页`；页大小固定 20 |
| `FilterSelect` / `Segmented` | 统一 32 高、token 边框与背景；方向段选中时收/送分别用 `--c-in`、`--c-out` |
| `StatCard` | 数值 24/32、标签 13；数值颜色只允许 in/out/pending/ink |
| `BarRow` | 同一刻度的收/送双条，右侧数值两行 |
| `MonthChart` | 内联 `<svg>`，12 个月双柱 + 网格 + 轴标签；不引入图表库 |
| `PersonCell` | 头像 + 姓名 + 关系/备注；点击进入亲友详情 |
| `RecordTable` | 基于 `WebTable` 的记录列组合；支持 `chk`、`date`、`person`、`event`、`book`、`cat`、`amount`、`status`、`page`、`anno`、`ops` 列 |
| 复用组件 | `Money`、`EventSeal`、`PersonAvatar`、`DirTag`、`StatusPill`、`EmptyState`、`Skeleton` 直接引用 |

金额统一走 `Money` 或 `formatCent`；日期统一走 `preciseDateLabel`；状态文字优先使用接口返回的 `statusLabel`，不要把方向换算散落在页面里。

### 7.3 浮层

| 组件 | 触发 | 关键行为 |
| --- | --- | --- |
| `EntryDrawer` | 顶栏、首页、列表、详情 | 新建/编辑记录；亲友可输入姓名或拼音，保存时 `personName` 自动新建；事项/礼簿下拉，事项末尾"＋ 新建事项"打开 `POST /events` 小表单；方向切换状态选项；金额大写；礼品与估价；重复软确认；保存并再记一笔 |
| `RecordDrawer` | 表格行、搜索结果 | 详情、关联回礼、编辑、删除；被回礼关联的收礼记录禁止删除；送礼记录删除时处理 `unlink_on_delete` 确认 |
| `RepayDrawer` | 亲友详情、回礼清单、记录详情 | 多笔原收礼一起回；对方事项可新建（姓名 + 事项名，`POST /records/repay` 的 `event.name` 自动查/建）；名目可多条；保存后原记录改已回礼并生成送礼记录 |
| `PaperAnnoDrawer` | 纸本标注表格、记录详情 | 原文只读；解析草稿可改日期精度/事项/名目/金额；一条标注可拆多条送礼 |
| `MergeDialog` | 礼簿页 | 选源礼簿与目标礼簿；调用 preview；跨类别提醒"名目不变"；确认后合并 |
| `ConfirmDialog` | 删除、状态变更 | 标题、正文、危险按钮；Esc 关闭 |
| `CommandPalette` | `⌘K`、`/`、顶栏搜索 | 防抖 250ms；分组显示亲友/事项/礼簿/记录；记录打开详情抽屉，其余跳详情页 |
| `NotesDrawer` | 顶栏"交互说明" | 按当前路由显示 `web/notes.ts` 中的说明与快捷键表 |
| `ToastHost` | 所有写操作 | 成功/失败提示，3 秒消失 |

## 8. 页面规格

### 8.1 工作台 `#/pages/web/index`

数据：

- `fetchYears()` + `fetchYearStats(year)`：默认当前年；当前年没有记录时取最近有记录的年份。
- `fetchPending('owed')`：全部未回礼。
- `fetchRecords({ pageSize: 7 })`：最近记录。
- `fetchRecentEvents(4)`：近期事项。

布局：

- 朱砂 hero：问候语、日期、当年收/送/净差额、全部未回礼胶囊、四个快捷入口（记一笔、连续录入、识别礼簿、导入导出）。
- 左栏"最近记录"表格：日期、亲友、事项、礼簿、金额、状态。
- 右栏"最久没回的"：按最早未回礼排序取 5 人；"近期事项"取 4 个。
- 快捷入口：记一笔打开 `EntryDrawer`；连续录入跳 `book?tab=cont`（上次礼簿，没有则礼簿列表）；识别跳 `recognize`，非管理员隐藏；导入导出跳 `settings`。

边界：

- 没有记录时 hero 显示 0，不显示空图表；最近记录显示空状态。
- 所有写操作完成后 `revision++`，本页重新请求四组数据。

### 8.2 记录 `#/sub-web/records`

数据：

- `fetchRecords`：`pageSize=20`，参数来自筛选状态。
- 响应 `custom.summary` 提供筛选结果全量的收、送、未回礼金额与笔数。
- 字典来自 `dictStore`：礼簿类别、名目；事项类型来自 `eventTypes`。

筛选映射：

| 控件 | 接口参数 | 规则 |
| --- | --- | --- |
| 礼簿类别 | `bookCategory` | 字典 code |
| 事项类型 | `eventType` | 枚举 code |
| 亲友 | `personKeyword` | 姓名包含、全拼前缀、首字母前缀；250ms 防抖 |
| 时间 | `from`/`to` | 年份转 `YYYY-01-01` ~ `YYYY-12-31` |
| 收/送分段 | `direction` | 空 = 全部 |
| 回礼状态 | `status` | 文字随方向变化；只选状态不选方向时，按收礼处理 |
| 名目 | `itemKind` | 字典 code |
| 页码 | `page` | 任一筛选变化回到第 1 页 |

表格列：日期、亲友、事项、礼簿、名目、金额（方向标签 + 金额/礼品）、状态、纸本页码、纸本标注。行点击打开 `RecordDrawer`；亲友单元格跳详情；未回礼行左侧 3px 琥珀条。

底部汇总：`共 N 笔`、`收 ¥`、`送 ¥`、`其中未回礼 ¥`；右侧分页。

工具：导出当前结果（把当前筛选参数传给 `/export`，导出 CSV）、记一笔。

### 8.3 亲友 `#/sub-web/people`

数据：`fetchPersons({ keyword, relation, page, pageSize: 50 })`。接口已返回 `inCent`、`outCent`、`pendingCount`、`lastDate`，净差额在前端计算。

筛选：姓名/拼音输入 250ms 防抖；关系分段。

表格列：姓名（头像 + 备注）、关系、总收、总送、净差额、未回礼（笔数 + 金额）、最近往来。未回礼行左侧琥珀条；空结果显示"没有找到，试试拼音首字母"。

工具：新建亲友（姓名必填、关系、电话、备注；同名走后端软确认）。

### 8.4 亲友详情 `#/sub-web/person?id=`

数据：`fetchPersonDetail(id)`、`fetchPersonTimeline({ personId: id })`。

布局：

- 头部：头像、姓名、关系/电话/备注；编辑、回礼（选中后可用）、给此人记一笔。
- 四个统计卡：总收、总送、净差额、未回礼笔数与金额。
- 时间线：按事项分组；组头显示印章、事项名、日期、主办方、笔数、跨礼簿数；组内表格列：勾选、日期、礼簿、名目、金额、状态。
- 只有收方向且 `status=pending` 的记录可勾选；勾选集合跨事项保留；顶部"回礼 · 已选 N 笔"打开 `RepayDrawer`。
- 群组合计跨礼簿、跨名目；不把回礼生成的送礼记录误算成新的收礼。

### 8.5 事项 `#/sub-web/events`

数据：`fetchEvents({ page, pageSize: 50 })`，按日期倒序。

表格列：事项（印章 + 名称 + 类型/地点）、日期、主办方、人数、收、送、未回礼笔数。行点击进详情。

工具：新建事项（名称、类型、日期、地点、主办方、备注）；编辑；删除仅在无记录时可用（后端 409）。

### 8.6 事项详情 `#/sub-web/event?id=`

数据：`fetchEventDetail(id)`、`fetchEventSummary(id)`。

统计卡：总收入、总送出、参与人数、已回礼/未回礼笔数。

三个分段：

| 分段 | 过滤条件 | 金额列 |
| --- | --- | --- |
| 谁来了 | `inCent > 0` | 收合计 |
| 我回了 | `repayCount > 0` | 我回了金额 |
| 还有谁没回 | `pendingCount > 0` | 收合计 |

每个分段按人一行，明细列出礼簿 · 名目 · 金额；行点击进亲友详情；未回礼行左侧琥珀条。底部"在此事项记一笔"打开 `EntryDrawer({ eventId })`。

### 8.7 礼簿 `#/sub-web/books`

数据：`fetchBooks('normal')`、`fetchBooks('merged')`。

表格列：礼簿（印章 + 名称 + 描述）、类别、默认事项、记录数、收、送、并入目标。已合并列表只读，行首提示"并入《目标》"。

工具：合并礼簿（`MergeDialog`）、新建礼簿（名称、类别、默认事项、描述）。已合并礼簿不能编辑或删除。

### 8.8 礼簿详情 `#/sub-web/book?id=&tab=`

数据：

- `fetchBookDetail(id)`：记录数、收送、默认事项、类别。
- `fetchRecords({ bookId: id, page, pageSize: 20 })`：记录 tab。
- `fetchPaperPages(id)`：按页核对。
- `fetchPendingNotes(id)`：纸本标注。
- `fetchBooks()`：连续录入的礼簿选择（当前礼簿固定）。

tabs：

| tab | 内容 | 写入 |
| --- | --- | --- |
| 记录 | 记录表格：纸本页码、亲友、事项、名目、金额、状态、标注 | —— |
| 连续录入 | 上下文条 + 单行录入 + 本页合计 + 本次清单 | `createRecord`、`deleteRecord`、`setPageCheck` |
| 按页核对 | 页码、纸本合计、录入合计、差额、笔数 | 点击查看跳记录 tab |
| 整理纸本标注 | 标注原文、预填回礼、核对并转换 | `repayRecords` |

连续录入：

- 顶部固定事项、日期、纸本标记、本页合计；默认事项取礼簿 `defaultEventId`。
- 一行：姓名、金额、名目、标注原文；Enter 提交，焦点回到姓名。
- `createRecord` 失败返回 `need_confirm` 时，标记该行"可能重复"，不阻断；用户确认后带 `confirm` 重试。
- 每次提交后刷新本页录入合计；与纸本合计比较，差额实时显示。
- 撤销本次录入调用 `deleteRecord`；若已被回礼关联，按后端 409 提示。
- 合并到目标礼簿后，源礼簿只读，tab 中不显示写入按钮。

### 8.9 回礼清单 `#/sub-web/repay`

数据：`fetchPending('owed')`、`fetchPending('owing')`。后端补 `records` 后直接渲染每人明细；未补前先只显示合计，展开时按人请求。

两个分段：

| 分段 | 数据 | 操作 |
| --- | --- | --- |
| 我该回的 | `owed` | 每人一行，回礼打开 `RepayDrawer`，默认带上此人的全部未回礼记录，可删减 |
| 该回我的 | `owing` | 每人一行，标为对方已回（`markRecordRepaid` 或批量） |

表格列：亲友、明细（事项 · 礼簿 · 名目 · 金额）、合计、最早一笔、操作。按最早一笔升序。

### 8.10 统计 `#/sub-web/stats?year=`

数据：`fetchYears()`、`fetchYearStats(year)`；年份含"全部"（`year=all`）。

内容：

- 五个统计卡：总收、总送、净差额、记录数、涉及人数。
- 单年显示按月双柱 SVG；全部年份隐藏按月图。
- 按礼簿类别、按事项类型两组双条柱，使用同一刻度。
- 往来最多的人表格：总收、总送、净差额、笔数、最近往来。

口径：合计 = 现金 + 礼品估价；同一笔只计一次；合并礼簿仍按每条记录的名目拆分。

### 8.11 整理纸本标注 `#/sub-web/notes`

数据：`fetchPendingNotes(0)`。

表格列：亲友、原收礼（金额/礼品）、标注原文、预填回礼、操作。预填回礼来自 `POST /paper-notes/parse`，日期精度、事项、名目、金额可改。

操作：

- "核对并转换"打开 `PaperAnnoDrawer`，保存后调用 `POST /records/repay`，原记录改已回礼，并保留原始标注。
- "全部按预填转换"逐条提交，遇到 `onlyMark`（只有已还记号）调用 `markRecordRepaid`；失败时停止并报告第几条。
- 转换成功刷新列表与徽标。

### 8.12 按页核对 `#/sub-web/pages`

数据：`fetchBooks('normal')` 后并发 `fetchPaperPages(bookId)`，按礼簿分组汇总。

表格列：礼簿、纸本标记、笔数、纸本合计、录入合计、差额。差额为 0 显示绿色"对上了"，否则红色带符号金额。点击"查看"跳对应礼簿记录 tab。

礼簿数量超过 20 时改为只加载当前筛选礼簿；第一版默认全量并发，接口调用数受礼簿数量限制。

### 8.13 设置 `#/sub-web/settings`

四个分区：

1. 礼簿整理：合并礼簿、已合并礼簿、整理纸本标注、按页核对。
2. 导入导出：复用 `exportRecords`、`previewImportFile`、`commitImport`；CSV 带 BOM；导入先预览逐行错误，确认后写入。
3. 名目与类别：`fetchDicts`、`createDict`、`updateDict`、`deleteDict`；内置项不可删，被引用时显示 409 提示。
4. 账号与登录设备：`fetchSessions`、`deleteSession`；本机开发免登录时显示"开发免登录"。

设置项 `showOwing`、快捷金额等使用 `contextStore.save`。

### 8.14 识别 `#/sub-web/recognize`

仅管理员可见。四个步骤：上传 → 识别中 → 核对 → 完成；历史页 `recognize-history`。

- 上传：点击选择、拖放、`Ctrl/⌘+V` 粘贴；jpg/png，单张 ≤ 5MB；客户端先校验，再调 `uploadRecognizeImage`。
- 识别：轮询 `fetchRecognizeTask`，1.5 秒一次，100 秒超时；显示已用时间与取消。
- 核对：左侧原图（`ImageViewer`：缩放、拖动、旋转、适应），右侧表格；顶部上下文（礼簿、事项、日期、纸本标记、纸本合计 vs 识别合计）。
- 表格列：序号、姓名、大写、小写、名目、回礼标注、提示、确认、删除；标黄行（同名、大小写不一致、低置信度）必须勾选确认才能写入。
- 同名候选必须选择已有亲友或新建；缺姓名/金额的行必须补全或删除。
- 底部："添加一行"、"重新识别"、"写入 N 条"；不可写入时显示原因。
- 写入调用 `commitRecognizeTask`；成功显示写入条数、回礼条数、礼簿链接。
- 历史：卡片网格/列表，状态、条数、合计；待核对任务可继续，已写入只读。
- 键盘：复用 `useHotkeys` 的识别快捷键；只在核对步骤启用。

### 8.15 登录 `#/sub-web/login`

独立全屏页面，不渲染侧栏。

- 左侧朱砂品牌区，右侧表单。
- 分段：小程序扫码 / 访问令牌。
- 扫码：`createWebTicket` → 轮询 `pollWebTicket`；2 分钟过期可刷新；小程序确认后写入令牌并进工作台。
- 令牌：`auth.loginWithToken(token)`；失败显示后端消息。
- 开发免登录：`auth.loadMe()` 成功且后端返回开发用户时，顶部提示"本机开发模式已免登录"，可直接进入。
- 登录成功回到 `redirect` 查询参数指向的页面，没有则工作台。
- 登录页不显示侧栏与识别入口。

### 8.16 全局搜索与快捷键

- `⌘K` 或 `/` 打开命令面板；Esc 关闭；输入 250ms 防抖后调用 `searchAll(q, 5)`。
- 分组：亲友、事项、礼簿、记录；显示总数；记录结果显示金额、礼簿、页码、日期与 `matchField` 高亮。
- 上下键移动选中项，Enter 打开第一项；点击亲友/事项/礼簿跳详情；点击记录打开 `RecordDrawer`。
- `N` 在非输入状态下打开记一笔；Drawer/Dialog 打开时 `N` 不触发。
- Esc 依次关闭命令面板、抽屉、对话框。
- 所有快捷键只在 H5 注册，页面卸载时注销。

## 9. 共享逻辑抽取

桌面版不能复制现有页面里的校验与编排。按下面顺序抽取，先让小程序页面改为使用同一份逻辑，再写 Web 视图：

| 模块 | 来源 | 抽出的内容 | 使用方 |
| --- | --- | --- | --- |
| `useRecordQuery` | `sub-record/list.vue` | 筛选状态、方向/状态联动、年份转 from/to、分页重置、参数序列化 | Web 记录页、亲友时间线、礼簿记录页 |
| `useEntryForm` | `sub-record/form.vue` | 表单状态、名目推断、金额校验、软确认、保存/编辑/删除 | Web `EntryDrawer`、小程序记一笔 |
| `useContinuousEntry` | `sub-record/continuous.vue` | 上下文、单行提交、重复标记、本页合计、撤销、页码递增 | Web 连续录入、小程序连续录入 |
| `useRepayForm` | `sub-record/form.vue` 回礼模式、`sub-misc/pending.vue` | 原记录选择、金额/名目/事项、`markRepaid`、提交 | Web `RepayDrawer`、小程序回礼 |
| `usePaperAnno` | `sub-record/notes.vue`、`PaperRepaySheet.vue` | 解析、草稿编辑、单条/批量转换、仅记号处理 | Web 纸本标注、小程序标注 |
| `useRecognizeFlow` | `sub-recognize/index.vue` | 上传、轮询、取消、恢复任务、写入 | Web 识别页、旧识别页过渡期 |
| `useRecordSummary` | 新增 | 读取 `/records` 的 `custom.summary`，统一类型 | Web 记录页、小程序可选用 |

规则：

- 页面不得自行实现"金额与礼品至少一项""同名软确认""合并后只读""已回礼不能删收礼"等规则。
- 共享 composable 不依赖 `window`/`document`；需要浏览器能力时在调用方用 `#ifdef H5` 处理。
- 抽取时先补测试或至少补一条真实页面回归，再切换调用方。

## 10. 后端改动

所有改动都是加法，响应新增字段、查询新增可选参数，不改既有字段含义，不需要改表。

### 10.1 记录列表汇总

**问题**：桌面记录页底部要显示筛选结果全量的收、送、未回礼，现有 `/records` 只有当前页与总数。

**改动**：

- `backend/infra/model/record/record.go` 增加：

```go
type Summary struct {
    Count       int64 `json:"count"`
    InCent      int64 `json:"inCent"`
    OutCent     int64 `json:"outCent"`
    PendingCent int64 `json:"pendingCent"`
}
```

- `record/store` 增加 `Summary(db, ownerID, q)`，复用现有 `filtered`，只替换 `Select`：

```sql
SELECT COUNT(*) AS count,
       COALESCE(SUM(CASE WHEN direction = 'in'  THEN total_cent ELSE 0 END), 0) AS in_cent,
       COALESCE(SUM(CASE WHEN direction = 'out' THEN total_cent ELSE 0 END), 0) AS out_cent,
       COALESCE(SUM(CASE WHEN direction = 'in' AND status = 'pending' THEN total_cent ELSE 0 END), 0) AS pending_cent
FROM record
WHERE …与列表完全相同的筛选…
```

- `record/service.List` 在返回列表的同时计算汇总；`record/api.List` 追加：

```go
response.WithCustomField("summary", summary)
```

- 前端 `api/types.ts`：`Page<T>` 增加 `custom?: Record<string, unknown>`；新增：

```ts
export interface RecordSummary {
  count: number
  inCent: number
  outCent: number
  pendingCent: number
}
```

- `fetchRecords` 返回 `RecordPage`，把 `body.custom?.summary` 读成 `summary`；现有调用方仍可使用 `items/total`。

### 10.2 亲友文本筛选

**问题**：原型筛选栏支持姓名/拼音首字母，`/records` 只有 `personId`。

**改动**：

- `record.Query` 增加 `PersonKeyword string \`json:"personKeyword"\``。
- `record/api.List` 读取 `personKeyword`。
- `record/store.filtered` 增加：

```go
if q.PersonKeyword != "" {
    like := "%" + q.PersonKeyword + "%"
    prefix := q.PersonKeyword + "%"
    query = query.Where(`person_id IN (
        SELECT id FROM person WHERE owner_id = ? AND
        (name LIKE ? OR name_pinyin LIKE ? OR name_initials LIKE ?)
    )`, ownerID, like, prefix, prefix)
}
```

- `/search/records` 复用同一查询时同步增加参数；`SearchRecordParams` 增加 `personKeyword`。
- 索引已存在：`idx_person_owner_name`、`idx_person_owner_initials`。

### 10.3 回礼清单明细

**问题**：原型每人行下直接列出记录；`PendingPerson` 只有合计。

**改动**：

- `backend/infra/model/view/view.go`：

```go
type PendingPerson struct {
    Person    person.Ref    `json:"person"`
    Count     int64         `json:"count"`
    TotalCent int64         `json:"totalCent"`
    Earliest  string        `json:"earliest"`
    Records   []record.Resp `json:"records"`
}
```

- `ViewSrv.Pending` 在拿到人列表后，用一次 `record.Store.List` 查出对应方向、`status=pending` 的记录，经 `recordSrv.Serialize` 后按 `personId` 分组填入。避免逐人查询。
- 记录数超过 500 时只返回前 500 条并在 `custom.truncated=true` 标记；桌面展开时再按人分页。正常情况下不会触发。
- 前端 `PendingPerson` 增加 `records: RecordItem[]`。

### 10.4 统计

**改动一：按事项类型收/送拆分**

- `stats.EventTypeRow` 增加 `InCent`、`OutCent`。
- `stats.Store.ByEventType` 的 SQL 增加：

```sql
COALESCE(SUM(CASE WHEN r.direction = 'in'  THEN r.total_cent ELSE 0 END), 0) AS in_cent,
COALESCE(SUM(CASE WHEN r.direction = 'out' THEN r.total_cent ELSE 0 END), 0) AS out_cent
```

- `stats.EventTypeStat` 增加 `InCent`、`OutCent`，`cent` 保留兼容。
- 前端 `EventTypeStat` 增加 `inCent`、`outCent`。

**改动二：往来最多的人的最近往来**

- `stats.PersonRow` / `stats.PersonStat` 增加 `LastDate`，SQL 增加 `MAX(r.occurred_on) AS last_date`。
- 前端 `PersonStat` 增加 `lastDate`。

**改动三：全部年份**

- `/stats/year` 接受 `year=all`（或新增 `all=true`，二者只保留一种，实现时选 `year=all` 更直观）。API 层要先用 `GetStrFromQuery` 读原始值，等于 `all` 时把 `all=true` 传给 service；不要直接用 `GetInt64FromQuery`，否则 `all` 会被解析成 0。
- `StatsSrv.Year` 收到 `all` 时使用 `0001-01-01` ~ `9999-12-31`；月度数据仍返回 12 个月合计，前端在"全部"时不渲染按月图。
- 前端 `fetchYearStats(year: number | 'all')`，`stats` 页年份分段增加"全部"。

### 10.5 事项详情的回礼标记

**问题**：原型"我回了"分段需要知道哪些送礼记录是回礼生成的。

**改动**：

- `view.PersonDetailLine` 增加 `IsRepay bool`。
- `view.EventPerson` 增加 `RepayCount int64`、`RepayCent int64`。
- `ViewSrv.EventSummary` 的循环中：

```go
if item.Direction == consts.DirectionOut && item.IsRepay {
    entry.RepayCount++
    entry.RepayCent += item.TotalCent
}
```

- 明细分组 key 加上 `item.IsRepay`，避免同一礼簿、同一名目的普通送礼与回礼被合并：

```go
detailKey := fmt.Sprintf("%d|%d|%s|%s|%t", item.PersonID, item.BookID, item.ItemKind, item.Direction, item.IsRepay)
```

- 前端 `EventPerson`、`EventPersonLine` 增加对应字段。

### 10.6 其它

- `/export` 已扩展为接受 `bookId`、`eventId`、`personId`、`bookCategory`、`eventType`、`itemKind`、`personKeyword`、`direction`、`status`、`from`、`to`；记录页"导出当前结果"直接复用 `record.Query` 的筛选。
- `/books/pages` 第一版由前端并发按礼簿请求；礼簿数量超过 20 再增加"全部礼簿分页"接口。

### 10.7 测试

后端在 `backend/test/` 增加：

1. `record_test.go`：`personKeyword` 命中姓名、拼音前缀、首字母前缀；与其他筛选组合；`summary` 与分页前后一致。
2. `aggregate_test.go`：事项类型收/送拆分；`year=all` 与单年合计关系；`lastDate`。
3. `view` 相关：回礼清单包含明细；事项汇总的 `repayCount` 只统计回礼生成的送礼记录。
4. 回归：既有响应字段不变；空结果 `data=[]`；无记录时汇总为 0。

## 11. 实施顺序

| 阶段 | 内容 | 完成标志 |
| --- | --- | --- |
| W0 基础 | token、H5 页面声明、`WebShell`、侧栏/顶栏、`webUi`、toast、`WebTable`、图标 | H5 打开工作台，侧栏可切换空页面；小程序构建不变 |
| W1 只读 | 工作台、记录、亲友、亲友详情 | 四页数据与现有小程序一致；筛选、分页、查找可用 |
| W2 只读 | 事项、事项详情、礼簿、礼簿详情记录 tab、回礼清单、统计 | 原型前 9 屏可浏览，金额与统计口径一致 |
| W3 写入 | `EntryDrawer`、`RecordDrawer`、`RepayDrawer`、删除、合并、新建/编辑亲友/事项/礼簿 | 记账、回礼、删除、合并全链路可用 |
| W4 纸本 | 连续录入、按页核对、整理纸本标注、设置、导入导出、命令面板 | 纸本录入与导入导出可用 |
| W5 识别/登录 | 识别页、识别历史、登录页、管理员限制 | 管理员可完成上传→核对→写入；令牌/扫码可登录 |
| W6 收尾 | 响应式、快捷键、焦点、空/错/骨架屏、旧 H5 页面退出声明、文档 | `pnpm build`、两端产物检查、验收清单通过 |

每个阶段结束后都跑一次 `pnpm build` 与后端测试，不把问题堆到最后。

## 12. 验收

### 12.1 构建与产物

```bash
cd backend
CGO_ENABLED=0 go build ./...
go vet ./...
go test ./test

cd ../frontend
pnpm build            # 类型检查 + H5 + 小程序
```

产物检查：

- 小程序产物中搜不到 `WebShell`、`sub-web`、`webUi`、`layout-wide`、`识别礼簿`。
- H5 产物包含 `sub-web` 页面；不再出现 430px 画布与 `pages/person/list` 等手机页面（条件编译排除）。

### 12.2 功能场景

1. 工作台四组数字与统计页同一年份一致；未回礼胶囊数量与回礼清单一致。
2. 记录页任意筛选组合：底部汇总等于全部匹配记录之和，不只是当前页；翻页后汇总不变。
3. 只选"未回礼"不选方向，不混入送礼记录；切到送达后状态文字变为"对方未回/对方已回/不用回"。
4. 亲友详情勾选两笔不同礼簿的未回礼，一次回礼生成两条送礼记录，原记录都变已回礼；净差额同步变化。
5. 事项详情"我回了"只显示回礼生成的送礼记录；"还有谁没回"与回礼清单一致。
6. 合并跨类别礼簿：记录名目不变，统计按名目拆分；源礼簿只读。
7. 连续录入 15 行：本页合计实时更新；差额为 0 显示"对上了"；重复行提示"可能重复"但不阻断。
8. 纸本标注"11年国庆 孙女 100元 火炮20元"拆成两条回礼（现金 100、火炮 20），日期精度为月。
9. 识别页标黄行未确认时写入按钮禁用；确认后可写入；回礼标注随收礼生成送礼记录。
10. 非管理员看不到识别入口，直接访问 `#/sub-web/recognize` 被拦回工作台。
11. 浏览器刷新、前进/后退、深链 `book?id=2&tab=cont` 均能恢复页面与 tab。
12. 1440、1024、960、375 宽各检查一次：侧栏/顶栏切换、表格横向滚动、抽屉宽度、按钮不重叠。

### 12.3 一致性口径

- 金额：接口分为单位，页面只通过 `formatCent`/`Money` 显示；合计 = 现金 + 礼品估价。
- 状态：只使用接口 `statusLabel` 或统一的 `record.StatusLabel` 规则；回礼记录显示"回礼"。
- 日期精度：日/月/年分别显示"约"；不要用创建时间代替发生日期。
- 合并：显示当前礼簿 + `originBookName` 追溯；统计按记录名目，不按礼簿类别重算。

## 13. 风险与后续

| 风险 | 处理 |
| --- | --- |
| 两端 UI 分叉导致业务逻辑漂移 | 第 9 节共享 composable 是硬约束；页面只做展示 |
| H5 每个页面都挂载 `WebShell`，切页时侧栏重建 | 页面过渡关闭；shell 保持轻量；筛选状态由页面栈/URL 保留 |
| `/records` 汇总查询增加一次聚合 | 复用同一筛选与索引；万级数据下实测；超过阈值再缓存 |
| `personKeyword` 子查询 | 已有姓名/首字母索引；组合筛选时看执行计划 |
| 回礼清单返回明细变大 | 500 条上限 + `truncated` 标记；超限按人懒加载 |
| 桌面表格在窄屏拥挤 | 960/520 两档断点；表格横向滚动；关键列固定 |
| 原型与 `fen.md` 的"手机尺寸"冲突 | 定稿前由需求方确认；本文档按最新原型执行，验收口径仍是 `fen.md` 第 9 节 |
| 旧 H5 识别/登录页残留 | 源码保留但不再声明；产物检查确保无引用，组件继续被桌面页复用 |

## 14. 参考

- `docs/project/web-prototype.html`：桌面原型。
- `docs/project/fen.md`：业务规则与验收清单。
- `docs/01-数据库设计.md`：表结构、筛选与统计口径。
- `docs/02-后端设计.md`：接口契约、软确认、回礼与识别事务。
- `docs/03-前端交互与UI规范.md`：颜色、字体、金额/日期/状态规则。
- `docs/07-登录与鉴权.md`：令牌、扫码登录、管理员、原图访问。
- `docs/08-Web端能力与宽屏布局.md`：识别宽屏、上传增强、识别历史。
