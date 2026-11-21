# 人情往来账本

把多本纸质人情礼簿数字化成一份电子总账：能记、能查、能改。后端 Go + Gin + GORM + SQLite（纯 Go 驱动），前端 uni-app（Vue 3 + TypeScript），识别走阿里云百炼通义千问视觉模型（默认 qwen3-vl-plus）。

需求见 [docs/project/需求.md](docs/project/需求.md)，实施方案见 [docs/project/方案.md](docs/project/方案.md)。

## 已实现

- 礼簿：新建、编辑、删除（有记录时阻止），按名称、种类查询；种类预置人情礼、火炮、其他，可自定义。
- 亲友：姓名 + 备注全局唯一，同名的不同人用备注区分（显示为「张三 · 村东」）；按姓名、备注、拼音首字母查找；有记录时阻止删除。
- 记录：收挂礼簿、送记事情；记一笔时可按姓名匹配已有亲友，找不到自动新建。
- 总账：所有礼簿合并，按礼簿、方向、日期、关键词筛选，顶部显示总收、总送、笔数；礼簿详情只看本簿收礼，亲友详情看全部收送。
- 识别（仅 H5）：上传礼簿照片 → 后台识别 → 四列核对表（确认、姓名、金额、备注）→ 选礼簿和方向后批量写入。
- 本机单用户，无登录；所有数据归属 `owner_id = 1`。

## 环境

- Go 1.25+，构建不需要 CGO。
- pnpm 12+；Node 由 pnpm 按 `frontend/package.json` 的 `devEngines.runtime` 自动下载。
- 本机默认：后端 `127.0.0.1:8080`，H5 开发 `http://localhost:5173`。

## 初始化与启动

```bash
make -C backend migrate          # 按 backend/migrate/sqlite/schema.sql 建表，可重复执行
make -C backend run              # 启动后端
cd frontend && pnpm install && pnpm dev:h5   # 启动 H5，/api 代理到 8080
```

改表结构时直接改快照；字段变化大时备份后删除 `backend/data/harmony.db*` 重新 migrate。

## 配置

`backend/configs/config-local.yaml` 只写非机密配置（server、database、storage、recognize、backup）。识别的 API Key 只放环境变量：

```bash
export DASHSCOPE_API_KEY=sk-...   # 或 HARMONY_RECOGNIZE_API_KEY / ALI_BAILIAN_API_KEY
```

## 交付

```bash
make release                      # 本机单二进制，内置 H5
./backend/bin/harmony -config backend/configs/config-local.yaml
./backend/bin/harmony backup -config backend/configs/config-local.yaml   # VACUUM INTO 在线备份
```

## 验证

```bash
cd backend && CGO_ENABLED=0 go build ./... && go vet ./... && go test ./test -count=1
cd frontend && pnpm build         # vue-tsc + H5 + 小程序
```

`backend/test` 使用临时 SQLite 文件，覆盖礼簿、亲友（同名备注区分、歧义匹配）、记录（收送规则、筛选与合计口径一致）、识别批量写入（写入前校验、同批新姓名只建一次）。

## 目录

```text
backend/
├── cmd/harmony/          服务入口与 migrate/backup 子命令
├── component/<领域>/      api / service / store 三层：person、book、record、recognize、health
├── infra/                 consts、logger、response、rdb、middleware、model、utils
├── migrate/sqlite/        SQLite 全量建表快照
├── configs/               本机配置
├── test/                  service 集成测试
└── web/                   H5 构建产物嵌入目录
frontend/src/
├── pages/web/ sub-web/    H5 页面：总账、礼簿、亲友、识别
├── web/                   H5 外壳、视图、抽屉、组件
├── api/                   请求封装与按领域接口
├── components/            通用展示组件
├── stores/ styles/ utils/
deploy/                    systemd、Caddy、备份与发布脚本
```

## API 约定

- 前缀 `/api/v1`；传输层恒为 HTTP 200，业务状态在 `code`：400 参数错误、404 不存在、409 冲突（`custom.reason` 为 `duplicate_person`、`person_ambiguous`、`has_reference`、`state_conflict`）。
- 不使用路径参数：GET/DELETE 参数走 query，POST 走 JSON body，PUT 的 `id` 走 query、其余走 body。
- 金额以“分”为单位、字段以 `Cent` 结尾；日期 `YYYY-MM-DD`；时间戳 `int64` 毫秒；JSON 字段小驼峰。
- 列表响应带 `totalItems`、`itemsPerPage`、`startIndex`，分页参数 `page`、`pageSize`。
