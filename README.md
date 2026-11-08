# 人情往来账本

本机 Web 端优先的人情往来账本。后端 Go，前端 uni-app（一套代码产出 H5 与微信小程序），数据存本机 MySQL。第一阶段完整业务需求见 [fen.md](docs/project/fen.md)。

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
│   ├── src/pages/            页面，在 src/pages.json 中声明
│   ├── src/api/              请求封装 http.ts 与按领域的接口函数
│   ├── src/styles/           设计 token 与 wot-design-uni 主题
│   ├── src/utils/            格式化等工具函数
│   └── package.json
├── docs/                     设计与架构文档
└── README.md
```

## 环境

- Go 1.25+；`infra/rdb` 同时编译了 sqlite 驱动，构建需要 CGO 与系统 C 编译器
- MySQL：本机 OrbStack 容器 `mysql`（`~/work/docker/mysql/`，root/root），库 `harmony`
- pnpm 12+；Node 不需要全局安装，pnpm 按 `frontend/package.json` 的 `devEngines.runtime` 自动下载 22.20.0（安装脚本白名单见 `frontend/pnpm-workspace.yaml`）
- 本机默认端口：后端 8080，前端 5173

## 启动后端

先确认 MySQL 可用并建库：

```bash
docker exec mysql mysqladmin ping -uroot -proot
docker exec mysql mysql -uroot -proot -e "CREATE DATABASE IF NOT EXISTS harmony DEFAULT CHARSET utf8mb4;"
```

```bash
cd backend
make run
```

默认读取 `configs/config-local.yaml`，连接本机 MySQL 的 `harmony` 库。健康检查：

```bash
curl http://127.0.0.1:8080/api/v1/health
```

响应：

```json
{"code":200,"msg":"success","err":"","data":{"status":"ok","database":"mysql","version":"v0.1.0","time":1794114963879},"totalItems":0,"itemsPerPage":0,"startIndex":0,"custom":null}
```

## 启动前端

```bash
cd frontend
pnpm install
pnpm dev:h5          # H5：打开 http://127.0.0.1:5173
pnpm dev:mp-weixin   # 小程序：用微信开发者工具打开 dist/dev/mp-weixin
```

H5 由 Vite 将 `/api` 代理到 `http://localhost:8080`，可用 `VITE_PROXY_TARGET` 覆盖；小程序直连 `VITE_MP_API_BASE`（默认 `http://127.0.0.1:8080/api/v1`），开发者工具中需勾选"不校验合法域名"。

前端按手机尺寸设计，桌面浏览器中显示为居中的 430px 画布。

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
cd frontend && pnpm build   # 类型检查 + H5 构建 + 小程序构建
```

## 文档

- [docs/README.md](docs/README.md)：文档索引
- [docs/project/architecture.md](docs/project/architecture.md)：骨架结构与扩展方式
- [docs/00-总览与架构.md](docs/00-总览与架构.md)：技术方案总览，01 到 04 为数据库、后端、前端、小程序设计
