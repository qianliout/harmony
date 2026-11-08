# 架构

## 现状

当前是项目骨架：后端只有配置、数据库连接、日志、统一响应与健康检查；前端只有应用外壳与调用健康检查的工作台。业务组件按 `fen.md` 逐个补齐，不预留空目录或占位实现。

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

本机默认使用 SQLite 单文件库，DSN 为 `harmony.db`，文件落在启动目录。`infra/rdb` 同时支持 postgres 与 mysql，新增业务表时按目标数据库在 `backend/migrate/<driver>/` 维护全量建表快照；服务启动不自动迁移。

## 日志与响应

日志使用 `infra/logger`，每个需要日志的 struct 在构造时固定 module 与 subModule，调用时传 `ctx`。日志消息用英文，字段用结构化值。

HTTP 响应统一使用 `infra/response` 信封：传输层恒为 HTTP 200，业务状态在 `code`；单条用 `WithItem`，列表用 `WithItems` 并附带分页字段。

## 前端

启动链路：

```text
index.html → src/main.ts → router/index.ts → 自动收集 modules/*/index.ts 的 routes → AppLayout → 模块页面
```

模块约定：

- 页面路由在 `src/modules/<name>/router.ts`，`index.ts` 重新导出为 `routes`，由 `import.meta.glob` 自动收集。
- API 调用在 `src/modules/<name>/api/`，统一经过 `src/core/request`。
- 页面按手机尺寸设计，`AppLayout` 在桌面浏览器中居中为 480px 画布。
- 颜色、字号、圆角通过 `src/styles/index.css` 覆盖 Element Plus 变量；语义色只用于状态表达。

## 后续业务模块

| 模块 | 后端组件 | 前端模块 | 对应需求 |
| --- | --- | --- | --- |
| 本子 | `component/book` | `modules/books` | 本子增删改查、合并 |
| 人 | `component/person` | `modules/persons` | 人增删改查、跨本子时间线 |
| 事项 | `component/event` | `modules/events` | 事项增删改查、按人聚合 |
| 记录与还礼 | `component/record` | `modules/records` | 记账、连续录入、还礼关联 |
| 待还与统计 | `component/ledger` | `modules/ledger` | 待还清单、按年/人/事项/本子统计 |
| 图片识别 | `component/recognition` | `modules/recognition` | 上传、识别、核对、批量写入 |

新增业务时先建 model 与建表 SQL，再按 store、service、api、前端模块的顺序落地。
