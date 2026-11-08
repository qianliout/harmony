# 人情往来账本

本机与线上统一使用 SQLite 的人情往来账本。后端 Go + Gin + GORM，前端 uni-app（一套代码产出 H5 与微信小程序）；识别走阿里云百炼通义千问视觉模型（当前可用模型 qwen3-vl-plus，可通过配置切换）。

需求见 [docs/project/fen.md](docs/project/fen.md)，技术方案见 [docs/](docs/README.md)，上线步骤见 [docs/09-部署上线.md](docs/09-部署上线.md)。

## 已实现

- 礼簿、亲友、事项、记录 CRUD；跨礼簿按人/事项聚合；回礼关联（多对多）、解除关联、删除保护。
- 回礼状态按方向显示：收礼“未回礼/已回礼/不用回”，主动送礼“对方未回/对方已回/不用回”，回礼生成的送礼显示“回礼”。
- 主维度筛选：礼簿类别、事项类型、人、时间；辅以具体礼簿、事项、名目、方向、状态。
- 纸本标注解析、纸本回礼补录、整理标注、按页核对、礼簿合并。
- 全局/范围内搜索（文字、金额、区间、年份、纸本页码）、按年/事项/人/礼簿统计。
- 识别（仅 H5、仅管理员）：千问视觉识别、后台任务、核对、批量写入、识别历史、签名原图 URL。
- 登录鉴权：小程序微信登录、网页小程序码/访问令牌登录、令牌续期、登录设备、管理员、数据隔离。
- 导入导出：CSV（带 BOM）/ JSON、CSV 预览与导入。
- 部署：单二进制内置 H5、SQLite 备份、migrate/backup/token 子命令、systemd/Caddy/deploy 脚本。

## 环境

- Go 1.25+；SQLite 使用纯 Go 驱动，构建不需要 CGO。
- pnpm 12+；Node 由 pnpm 按 `frontend/package.json` 的 `devEngines.runtime` 自动下载。
- 本机默认：后端 `127.0.0.1:8080`，H5 开发 `http://localhost:5173`。

## 初始化与启动

```bash
# 1. 建库并写入种子（可重复执行）
make -C backend migrate

# 2. 启动后端
make -C backend run

# 3. 启动 H5（/api 代理到 8080）
cd frontend && pnpm install && pnpm dev:h5
```

H5 需要登录：开发阶段可签发访问令牌，在登录页底部“使用访问令牌登录”粘贴：

```bash
cd backend
go run ./cmd/harmony token issue -config configs/config-local.yaml -user 1 -days 14
```

首次用微信登录小程序时会认领 `app_user(1)` 成为管理员；其他微信用户得到独立空账本。

## 配置

`backend/configs/config-local.yaml` 只写非机密配置：

```yaml
server: { host: 127.0.0.1, port: 8080, mode: debug }
database: { path: data/harmony.db }
storage: { uploadDir: data/uploads }
recognize: { provider: qwen, model: qwen3-vl-plus, highResolution: true, dailyLimit: 100 }
auth: { adminUserIds: [1], mpTokenDays: 30, webTokenDays: 14 }
wechat: { appID: wx348e5ad03f9cd6ba, envVersion: develop }
backup: { dir: data/backups, keepDays: 7 }
```

机密只放环境变量：

```bash
export DASHSCOPE_API_KEY=sk-...          # 或 HARMONY_RECOGNIZE_API_KEY / ALI_BAILIAN_API_KEY
export HARMONY_WECHAT_APP_SECRET=...     # 小程序 code2Session 与小程序码
export HARMONY_AUTH_SECRET=...           # 原图签名，>=32 字符；debug 模式有开发默认值
```

## 交付与部署

```bash
make release              # 本机单二进制，内置 H5
./backend/bin/harmony -config backend/configs/config-local.yaml

make release-linux        # 产出 Linux 二进制
HOST=root@server make deploy   # 备份 → migrate → 切换 → 重启 → 健康检查/回滚
```

服务器使用 `deploy/harmony.service`、`deploy/Caddyfile`、`deploy/backup.sh`；备份通过 `VACUUM INTO`，每天 03:30 由 systemd timer 执行并同步到 OSS。

## 验证

```bash
cd backend && CGO_ENABLED=0 go build ./... && go vet ./... && go test ./test -count=1
cd frontend && pnpm build          # vue-tsc + H5 + 小程序
make release-mp                    # 小程序产物不得包含 recognize/识别
```

`backend/test` 使用临时 SQLite 文件，不需要 MySQL；覆盖跨礼簿聚合、口径一致、软确认、回礼、合并、纸本标注、按页核对、搜索意图、识别回滚、登录/扫码/隔离/签名、日期边界与并发回礼。

识别真实样张放在 `backend/test/testdata/recognize/`（不入库），用 `go test -tags live -run TestRecognizeLive -v ./test/` 评测五项指标。

## 目录

```text
backend/
├── cmd/harmony/          服务入口与 migrate/backup/token 子命令
├── component/<领域>/      api / service / store 三层
├── infra/                 logger、response、rdb、middleware、model、utils
├── migrate/sqlite/        SQLite 全量建表快照
├── configs/               本机配置
├── test/                  service 测试与真实样张目录
└── web/                   H5 构建产物嵌入目录
frontend/src/
├── pages/ sub-*/          页面（pages.json 声明）
├── components/            通用展示组件
├── api/                   请求封装与按领域接口
├── stores/                字典、记账上下文、登录态
├── composables/           分页、软确认、宽屏、快捷键
├── styles/                token、色板、全局样式
└── utils/                 金额、日期、中文大写、导航
deploy/                    systemd、Caddy、备份与发布脚本
```

## API 约定

- 前缀 `/api/v1`；除健康检查与登录入口外都需要 `Authorization: Bearer <token>`。
- 传输层恒为 HTTP 200，业务状态在 `code`；401 未登录、403 无权限、409 软确认/关联冲突。
- 金额以“分”为单位、字段以 `Cent` 结尾；日期 `YYYY-MM-DD`；时间戳 `int64` 毫秒。
- 识别接口额外要求 `X-Client-Platform: h5`，并且只对管理员开放；原图只通过签名 URL 访问。
