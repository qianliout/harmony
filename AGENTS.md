# AGENTS.md — harmony

## 项目

人情往来账本，第一阶段为本机 Web 端，后续增加微信小程序。前后端读写同一份数据。

- 后端：`backend/`，入口 `backend/cmd/harmony/main.go`，配置在 `backend/component/config`，本机配置 `backend/configs/config-local.yaml`。
- 前端：`frontend/`，uni-app（Vue 3 + TypeScript + Vite）+ wot-design-uni，一套代码产出 H5（Web 端）与微信小程序；Node 由 pnpm 按 `devEngines.runtime` 自动下载。
- 数据库：SQLite 单文件 `backend/data/harmony.db`，纯 Go 驱动，不使用 MySQL。
- 需求：`docs/project/fen.md`，是业务规则的唯一依据。
- 文档：`docs/`，索引见 `docs/README.md`。

## 开发规则

1. 不允许空目录或 stub 占位；建了目录就必须有真实内容，宁可先不建。
1. 后端 Go 改动先走 `go` skill，并按改动层加载 `go-api-layer`、`go-service-layer`、`go-query-dal`、`go-model-hierarchy`。
1. 后端分层固定：`api` 只做 HTTP 适配，`service` 只做业务编排，`store` 只做持久化，`infra/model` 管理字段生命周期；禁止跨层访问 DB、GORM 或 SQL。
1. 业务表落地时，建表 SQL 以全量快照维护在 `backend/migrate/sqlite/`，服务启动不自动迁移；改 schema 直接改快照，并用 `harmony migrate` 补齐。
1. API 前缀 `/api/v1`，传输层恒返回 HTTP 200，业务状态放 `code`；时间字段用 `int64` 毫秒，JSON 字段用小驼峰。
1. 前端使用 `<script setup lang="ts">` 与 Composition API；页面在 `src/pages.json` 声明，主包只放 tab 页，其余页面放分包；接口调用放 `src/api/<领域>.ts`，统一经过 `src/api/http.ts`；平台差异只用条件编译（`#ifdef H5`），图片识别相关代码只能出现在 H5 分支。
1. 页面按手机画布设计（桌面浏览器中居中为 430px）；视觉遵循"朱砂礼簿"主题，颜色、字号、间距只引用 `frontend/src/styles/tokens.scss` 中的 token，规范见 `docs/03-前端交互与UI规范.md`，原型见 `docs/project/index.html`。
1. 后端改动至少保证 `CGO_ENABLED=0 go build ./...`、`go vet ./...`、`go test ./test` 通过；前端改动至少保证 `pnpm build`（类型检查 + H5 + 小程序构建）通过。
