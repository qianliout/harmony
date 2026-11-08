# 人情往来账本

本机 Web 端优先的人情往来账本。后端 Go，前端 Vue 3，数据存本机 SQLite。第一阶段完整业务需求见 [fen.md](fen.md)。

## 当前状态

骨架已就绪：后端可启动并提供健康检查，前端工作台可读取该接口。
业务模块（本子、人、事项、记录、还礼、统计、图片识别）尚未实现，按 fen.md 逐步补齐。

## 目录

```text
harmony/
├── backend/
│   ├── cmd/harmony/          服务入口
│   ├── component/            业务组件：config、health
│   ├── configs/              本地配置
│   ├── infra/                通用基础设施：consts、logger、rdb、response、model/filter、utils
│   ├── go.mod
│   └── Makefile
├── frontend/
│   ├── src/core/             请求、路由、布局、通用组件
│   ├── src/modules/          业务模块，按 modules/<name>/{api,views,router.ts,index.ts} 组织
│   ├── src/styles/           全局样式与 Element Plus token
│   └── package.json
├── docs/                     设计与架构文档
├── fen.md                    产品需求
└── README.md
```

## 环境

- Go 1.25+，SQLite 驱动需要 CGO 与系统 C 编译器
- Node.js 18+，包管理用 pnpm 12+（安装脚本白名单见 `frontend/pnpm-workspace.yaml`）
- 本机默认端口：后端 8080，前端 5173

## 启动后端

```bash
cd backend
make run
```

默认读取 `configs/config-local.yaml`，使用当前目录下的 `harmony.db`。健康检查：

```bash
curl http://127.0.0.1:8080/api/v1/health
```

响应：

```json
{"code":200,"msg":"success","err":"","data":{"status":"ok","database":"sqlite","version":"v0.1.0","time":1794114963879},"totalItems":0,"itemsPerPage":0,"startIndex":0,"custom":null}
```

## 启动前端

```bash
cd frontend
pnpm install
pnpm dev
```

打开 http://127.0.0.1:5173。Vite 将 `/api` 代理到 `http://localhost:8080`，可用 `VITE_PROXY_TARGET` 覆盖。

前端按手机尺寸设计，桌面浏览器中显示为居中的 480px 画布。

## API 约定

- REST 前缀 `/api/v1`，创建用 `POST`，更新用 `PUT ?id=`，查询用 `GET`。
- 传输层恒返回 HTTP 200，业务状态放在 `code`；`data` 为单条或数组，列表带 `totalItems`、`itemsPerPage`、`startIndex`。
- 时间字段一律 `int64` 毫秒时间戳。
- 请求与响应字段名使用小驼峰。

## 后端分层

- `api` 只做 HTTP 适配与参数装配。
- `service` 做业务编排与错误包装。
- `store` 只做持久化。
- `infra/model` 管理字段生命周期、校验与序列化。
- `infra` 只放跨业务复用的基础设施。

## 验证

```bash
cd backend && go build ./... && go vet ./...
cd frontend && pnpm build
```

## 文档

- [docs/README.md](docs/README.md)：文档索引
- [docs/project/architecture.md](docs/project/architecture.md)：骨架结构与扩展方式
