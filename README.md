# 人情往来账本

本机 Web 端优先的人情往来账本。后端 Go + MySQL，前端 uni-app（一套代码产出 H5 与微信小程序）。业务规则见 [docs/project/fen.md](docs/project/fen.md)，实现约定见 [docs/](docs/README.md)。

## 已实现

- 礼簿：增删改、类别、默认事项、合并（含跨类别、二次合并）、已合并查看、按页核对。
- 亲友：增删改、同名软确认、拼音/首字母搜索、跨礼簿按事项聚合的时间线与汇总。
- 事项：增删改、按人聚合的来礼/回礼/未回、跨礼簿合计。
- 记录：收/送、名目推断、重复软确认、还礼多对多、解除关联、删除保护、纸本标注与按页核对。
- 纸本还礼：标注解析（年/月/节令/名目金额/多次还礼/只有记号）、补录还礼、整理标注。
- 搜索与统计：全局/范围内搜索（文字、金额、区间、年份、页码意图）、按年/事项/人/礼簿统计。
- 识别（仅 H5）：上传、异步识别、核对、批量写入、原始结果留痕；默认走 DeepSeek 的 OpenAI 兼容接口。
- 导入导出：CSV（带 BOM）/ JSON 导出、CSV 预览与导入。
- 小程序：同一套源码构建，识别相关页面与 API 不进入小程序产物。

## 环境

- Go 1.25+；`infra/rdb` 同时编译 sqlite 驱动，构建需要 CGO 与系统 C 编译器。
- MySQL：本机 OrbStack 容器 `mysql`（`~/work/docker/mysql/`，root/root）。
- pnpm 12+；Node 由 pnpm 按 `frontend/package.json` 的 `devEngines.runtime` 自动下载 22.20.0。

## 初始化数据库

```bash
docker exec mysql mysqladmin ping -uroot -proot
docker exec mysql mysql -uroot -proot -e \
  "CREATE DATABASE IF NOT EXISTS harmony DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;"
docker exec -i mysql mysql --default-character-set=utf8mb4 -uroot -proot harmony \
  < backend/migrate/mysql/schema.sql
```

建表快照包含种子：本机用户 `id=1`、本子类别与名目字典、占位事项“纸本未注明事项”。服务启动不自动迁移。

## 启动

```bash
# 后端 :8080
make -C backend run

# 前端 H5 :5173（/api 与 /uploads 代理到 8080）
cd frontend && pnpm install && pnpm dev:h5

# 小程序：用微信开发者工具打开 frontend/dist/dev/mp-weixin
cd frontend && pnpm dev:mp-weixin
```

## 识别配置

默认读取 `backend/configs/config-local.yaml` 的 `recognize` 段，API Key 通过环境变量注入，不写进文件：

```bash
export DEEPSEEK_API_KEY=sk-...          # 或 HARMONY_RECOGNIZE_API_KEY
# 可选覆盖
export HARMONY_RECOGNIZE_BASE_URL=https://api.deepseek.com/v1
export HARMONY_RECOGNIZE_MODEL=deepseek-chat
```

识别任务在后台调用模型；上传原图保存在 `backend/data/uploads/YYYYMM/`，通过 `/uploads/...` 访问。服务重启时未完成的识别任务会置为失败。

## 交付构建

```bash
make release
./backend/bin/harmony -config backend/configs/config-local.yaml
# 浏览器打开 http://localhost:8080
```

`make release` 依次构建 H5、复制到 `backend/web/dist`（`go:embed` 打进二进制）、构建后端。

小程序构建与边界检查：

```bash
make release-mp
```

## 验证

```bash
cd backend && CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go vet ./...
cd backend && CGO_ENABLED=1 go test ./test -count=1
cd frontend && pnpm build
```

`backend/test` 使用独立库 `harmony_test`，每次运行重建表并在事务中回滚；覆盖跨礼簿聚合、口径一致、软确认、还礼、合并、纸本标注、按页核对、搜索意图与识别回滚等条目。

## 目录

```text
backend/
├── cmd/harmony/          服务入口
├── component/<领域>/      api / service / store 三层
├── infra/                 logger、response、rdb、middleware、model、utils
├── migrate/mysql/         全量建表快照
├── configs/               本机配置
├── test/                  service 层测试
└── web/                   H5 构建产物嵌入目录（release 时生成）
frontend/src/
├── pages/ sub-*/          页面（pages.json 声明）
├── components/            通用展示组件
├── api/                   请求封装与按领域接口
├── stores/                字典、记账上下文
├── composables/           分页、软确认
├── styles/                token、色板、全局样式
└── utils/                 金额、日期、中文大写、导航
```

## API 约定

- 前缀 `/api/v1`；创建 `POST`、更新 `PUT ?id=`、删除 `DELETE ?id=`、查询 `GET`，资源 id 放在 query。
- 传输层恒为 HTTP 200，业务状态在 `code`；列表带 `totalItems`、`itemsPerPage`、`startIndex`。
- 金额以“分”为单位、字段以 `Cent` 结尾；日期 `YYYY-MM-DD`；时间戳 `int64` 毫秒；JSON 字段小驼峰。
- 软确认：`409 + custom.reason=need_confirm`，`data` 为确认信息；前端 `submitWithConfirm` 统一处理。
- 识别接口挂在 `PlatformGuard` 后，请求头 `X-Client-Platform` 必须为 `h5`。
