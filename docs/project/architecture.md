# 架构

## 现状

当前是项目骨架：后端只有配置、数据库连接、日志、统一响应与健康检查；前端是 uni-app 工程，只有调用健康检查的工作台页。业务组件按 `fen.md` 逐个补齐，不预留空目录或占位实现。

## 后端

请求路径固定为：

```text
Gin 路由 → component/<domain>/api → component/<domain>/service → component/<domain>/store → infra/rdb → 数据库
```

各层职责：

- `api`：解析 query/body，调用 model 的 `Serialize`/`Check`，调用 service，写统一响应。
- `service`：业务编排、跨资源聚合、错误包装与日志；禁止直接访问 DB。
- `store`：单一 data model 的持久化方法；禁止业务规则与业务日志。
- `infra/model`：GORM 实体、写入口 Param、读出口 Response 的字段契约与生命周期方法。
- `infra`：`consts`、`logger`、`rdb`、`response`、`filter`、`utils` 等跨业务基础设施。

已有组件：

| 组件 | 路由 | 作用 |
| --- | --- | --- |
| `config` | 无 | YAML + 环境变量配置加载与校验 |
| `health` | `GET /api/v1/health` | 检查数据库连通性并返回服务版本 |

装配统一在 `cmd/harmony/main.go`：先打开数据库，再构造 store、service、api，最后注册路由。

## 配置与数据库

配置默认值在 `component/config`，本机值在 `configs/config-local.yaml`，可用 `SERVER_PORT`、`SERVER_MODE`、`DATABASE_DRIVER`、`DATABASE_DSN` 覆盖。YAML 使用严格模式，未知键会阻止启动。

本机默认使用 MySQL（OrbStack 容器 `mysql`，库 `harmony`，root/root）。`infra/rdb` 同时支持 sqlite 与 postgres，新增业务表时在 `backend/migrate/mysql/` 维护全量建表快照；服务启动不自动迁移。表结构设计见 `docs/01-数据库设计.md`。

## 日志与响应

日志使用 `infra/logger`，每个需要日志的 struct 在构造时固定 module 与 subModule，调用时传 `ctx`。日志消息用英文，字段用结构化值。

HTTP 响应统一使用 `infra/response` 信封：传输层恒为 HTTP 200，业务状态在 `code`；单条用 `WithItem`，列表用 `WithItems` 并附带分页字段。

## 前端

uni-app（Vue 3 + TypeScript + Vite）+ wot-design-uni，一套代码产出 H5（Web 端）与微信小程序。启动链路：

```text
index.html → src/main.ts（createSSRApp + Pinia）→ src/pages.json 声明的页面 → 页面根节点 wd-config-provider 注入主题
```

约定：

- 页面在 `src/pages.json` 中声明；主包只放 tab 页，其余页面放分包。
- API 调用在 `src/api/<领域>.ts`，统一经过 `src/api/http.ts`（基于 `uni.request`，H5 走 Vite 代理，小程序直连）。
- 平台差异只用条件编译表达；图片识别相关代码只出现在 `#ifdef H5` 分支，小程序产物中不得包含。
- 颜色、字号、间距只引用 `src/styles/tokens.scss` 的 token，主题变量在 `src/styles/theme.ts`；桌面浏览器中 H5 居中为 430px 画布。
- 交互与视觉规范见 `docs/03-前端交互与UI规范.md`，原型见 `docs/project/index.html`。

## 后续业务模块

| 模块 | 后端组件 | 前端页面 | 对应需求 |
| --- | --- | --- | --- |
| 本子 | `component/book` | `pages/book`、`sub-book/*` | 本子增删改查、合并 |
| 人 | `component/person` | `pages/person`、`sub-person/*` | 人增删改查、跨本子时间线 |
| 事项 | `component/event` | `sub-event/*` | 事项增删改查、按人聚合 |
| 记录与还礼 | `component/record` | `sub-record/*` | 记账、连续录入、还礼关联 |
| 待还与统计 | `component/ledger` | `sub-misc/pending`、`sub-misc/stats` | 待还清单、按年/人/事项/本子统计 |
| 图片识别 | `component/recognition` | `sub-recognize/*`（仅 H5） | 上传、识别、核对、批量写入 |

新增业务时先建 model 与建表 SQL，再按 store、service、api、前端页面的顺序落地。
