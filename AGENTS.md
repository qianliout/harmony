# AGENTS.md — harmony

## 项目

人情往来账本，第一阶段为本机 Web 端，后续增加微信小程序。前后端读写同一份数据。

- 后端：`backend/`，入口 `backend/cmd/harmony/main.go`，配置在 `backend/component/config`，本机配置 `backend/configs/config-local.yaml`。
- 前端：`frontend/`，Vue 3 + TypeScript + Vite + Element Plus。
- 需求：`fen.md`，是业务规则的唯一依据。
- 文档：`docs/`，索引见 `docs/README.md`。

## 开发规则

1. 不允许空目录或 stub 占位；建了目录就必须有真实内容，宁可先不建。
1. 后端 Go 改动先走 `go` skill，并按改动层加载 `go-api-layer`、`go-service-layer`、`go-query-dal`、`go-model-hierarchy`。
1. 后端分层固定：`api` 只做 HTTP 适配，`service` 只做业务编排，`store` 只做持久化，`infra/model` 管理字段生命周期；禁止跨层访问 DB、GORM 或 SQL。
1. 业务表落地时，建表 SQL 以全量快照维护在 `backend/migrate/<driver>/`，服务启动不自动迁移；改 schema 直接改快照。
1. API 前缀 `/api/v1`，传输层恒返回 HTTP 200，业务状态放 `code`；时间字段用 `int64` 毫秒，JSON 字段用小驼峰。
1. 前端使用 `<script setup lang="ts">` 与 Composition API；模块放 `src/modules/<name>/`，通过 `index.ts` 导出 `routes` 自动接入路由。
1. 页面按 480px 手机画布设计；视觉遵循 `frontend/src/styles/index.css` 中的白色界面、单一主蓝与冷灰文字层级。
1. 后端改动至少保证 `CGO_ENABLED=1 go build ./...` 与 `go vet ./...` 通过；前端改动至少保证 `pnpm build` 通过。
