# SQLite 存储

本文档说明把数据库从 MySQL 整体切换到 SQLite 的方案。切换完成后，本机开发、测试、线上全部使用 SQLite，仓库里不再保留 MySQL 和 PostgreSQL 的驱动、建表快照和配置。

## 为什么可以用 SQLite

- 用户只有自己一个人，写入频率按"天"计，读写量远小于 SQLite 单文件的承载上限。
- 后端是单进程部署，只有这一个进程打开数据库文件，不存在多台机器共享文件的问题。
- 网页和小程序都只调用这一个后端，两端看到的数据天然一致，不需要任何同步机制。
- 备份就是复制一个文件，迁移服务器就是拷走一个目录。

需要接受的限制：同一时刻只能有一个写事务。对本应用来说，写操作都是毫秒级，排队不会被感知到。

## 驱动选型

使用纯 Go 实现的 `github.com/glebarez/sqlite`（GORM 驱动，底层是 `modernc.org/sqlite`），不使用依赖 CGO 的 `gorm.io/driver/sqlite`（`mattn/go-sqlite3`）。

| 对比项 | glebarez/sqlite | gorm.io/driver/sqlite |
| --- | --- | --- |
| CGO | 不需要 | 需要 |
| 在 Mac 上编译 Linux 二进制 | `GOOS=linux go build` 直接产出 | 需要 Linux 交叉编译工具链或在 Linux 上编译 |
| 性能 | 略慢，本应用感知不到 | 略快 |
| SQLite 版本 | 随 modernc 内置，3.4x | 随 mattn 内置，3.4x |

切换后所有构建命令改为 `CGO_ENABLED=0`，`backend/Makefile` 和 `AGENTS.md` 第 8 条同步修改。

## 连接参数

DSN 写成文件路径加 pragma：

```text
file:data/harmony.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(0)&_pragma=synchronous(NORMAL)&_txlock=immediate
```

| 参数 | 作用 |
| --- | --- |
| `journal_mode(WAL)` | 读写不互相阻塞；写事务进行中，其他请求仍可读 |
| `busy_timeout(5000)` | 遇到写锁时最多等 5 秒，而不是立即报 `SQLITE_BUSY` |
| `foreign_keys(0)` | 与现有约定一致，不使用物理外键 |
| `synchronous(NORMAL)` | WAL 模式下的推荐值，断电最多丢最后一个事务，不会损坏文件 |
| `_txlock=immediate` | 事务一开始就拿写锁，避免两个读事务同时升级为写事务时互相等待 |

连接池在 `rdb.Open` 中设置：`SetMaxOpenConns(4)`、`SetMaxIdleConns(4)`、`SetConnMaxLifetime(0)`。不要设成 1：现有代码里存在事务进行中再用 `rdb.Get()` 做查询的写法，单连接会死锁。

服务启动时 `rdb.Open` 之后立即执行一次 `PRAGMA journal_mode` 查询并打印结果，确认 WAL 已生效；数据库文件所在目录不存在时先创建。

## 配置变化

`backend/configs/config-local.yaml`：

```yaml
database:
  path: "data/harmony.db"
```

- `database.driver`、`database.dsn` 两个键删除，改为只配 `path`，pragma 由 `rdb` 固定拼接，不让配置文件改动。
- 环境变量 `DATABASE_DRIVER`、`DATABASE_DSN` 删除，新增 `HARMONY_DATABASE_PATH`。
- `consts.DBDriverMySQL`、`DBDriverPostgres`、`ValidDBDriver` 删除；`rdb.Open` 只保留 SQLite 分支。
- 健康检查返回的 `database` 字段固定为 `sqlite`，同时增加 `journalMode` 字段。

## 建表快照

新建 `backend/migrate/sqlite/schema.sql`，删除 `backend/migrate/mysql/`。快照仍然是全量、可重复执行（全部用 `IF NOT EXISTS`），服务启动不自动执行。

### 类型映射

| MySQL | SQLite | 说明 |
| --- | --- | --- |
| `BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY` | `INTEGER PRIMARY KEY AUTOINCREMENT` | 只有 `INTEGER PRIMARY KEY` 才是 rowid 别名 |
| `BIGINT UNSIGNED`、`INT`、`BIGINT` | `INTEGER` | |
| `TINYINT(1)` | `INTEGER` | 取值 0 或 1，GORM 的 bool 能直接读写 |
| `VARCHAR(n)`、`CHAR(n)` | `TEXT` | SQLite 不检查长度；长度校验已经在 param 层做了，保持不变 |
| `MEDIUMTEXT`、`JSON` | `TEXT` | JSON 内容由 Go 侧序列化 |
| `DATE` | `TEXT` | 固定格式 `YYYY-MM-DD`，见下文"日期列" |
| `DATETIME(3)` | `TEXT` | 由驱动写入，见下文"时间列" |
| `total_cent BIGINT AS (...) STORED` | `INTEGER GENERATED ALWAYS AS (amount_cent + gift_value_cent) STORED` | 需要 SQLite 3.31 以上，modernc 内置版本满足 |
| `CHECK (...)` | 原样保留 | |
| `KEY idx_x (...)` | 单独的 `CREATE INDEX IF NOT EXISTS idx_x ON t(...)` | SQLite 不支持在建表语句里声明普通索引 |
| `UNIQUE KEY uk_x (...)` | 单独的 `CREATE UNIQUE INDEX IF NOT EXISTS` | |
| `ENGINE=... CHARSET=... COLLATE=...` | 删除 | |

`record` 在 SQLite 里不是保留字，表名不再需要反引号，但保留双引号写法 `"record"` 也不影响。

### 示例

```sql
CREATE TABLE IF NOT EXISTS record (
  id                INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id          INTEGER NOT NULL,
  person_id         INTEGER NOT NULL,
  event_id          INTEGER NOT NULL,
  book_id           INTEGER NOT NULL,
  origin_book_id    INTEGER NOT NULL,
  item_kind         TEXT    NOT NULL,
  direction         TEXT    NOT NULL,
  amount_cent       INTEGER NOT NULL DEFAULT 0,
  gift_name         TEXT    NOT NULL DEFAULT '',
  gift_value_cent   INTEGER NOT NULL DEFAULT 0,
  total_cent        INTEGER GENERATED ALWAYS AS (amount_cent + gift_value_cent) STORED,
  occurred_on       TEXT    NOT NULL,
  date_precision    TEXT    NOT NULL DEFAULT 'day',
  status            TEXT    NOT NULL,
  remark            TEXT    NOT NULL DEFAULT '',
  paper_note        TEXT    NOT NULL DEFAULT '',
  paper_page        TEXT    NOT NULL DEFAULT '',
  paper_seq         INTEGER NOT NULL DEFAULT 0,
  source            TEXT    NOT NULL DEFAULT 'manual',
  recognize_task_id INTEGER NULL,
  created_at        TEXT    NOT NULL,
  updated_at        TEXT    NOT NULL,
  CONSTRAINT ck_amount_nonneg CHECK (amount_cent >= 0 AND gift_value_cent >= 0),
  CONSTRAINT ck_amount_or_gift CHECK (amount_cent > 0 OR gift_name <> '')
);
CREATE INDEX IF NOT EXISTS idx_record_owner_person_date ON record(owner_id, person_id, occurred_on);
CREATE INDEX IF NOT EXISTS idx_record_owner_event       ON record(owner_id, event_id);
CREATE INDEX IF NOT EXISTS idx_record_owner_book        ON record(owner_id, book_id);
CREATE INDEX IF NOT EXISTS idx_record_owner_dir_status  ON record(owner_id, direction, status, occurred_on);
CREATE INDEX IF NOT EXISTS idx_record_owner_date        ON record(owner_id, occurred_on);
CREATE INDEX IF NOT EXISTS idx_record_dup               ON record(owner_id, person_id, event_id, book_id, item_kind);
CREATE INDEX IF NOT EXISTS idx_record_owner_book_page   ON record(owner_id, book_id, paper_page, paper_seq);
```

SQLite 的索引名在整个库内唯一，不像 MySQL 只在表内唯一，所以索引名统一加上表名前缀 `idx_<表>_...`。

### 种子数据

| MySQL 写法 | SQLite 写法 |
| --- | --- |
| `NOW(3)` | `strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime')` |
| `ON DUPLICATE KEY UPDATE a = VALUES(a)` | `ON CONFLICT(唯一列) DO UPDATE SET a = excluded.a` |
| `FROM DUAL WHERE NOT EXISTS (...)` | 去掉 `FROM DUAL`，`SELECT ... WHERE NOT EXISTS (...)` |

种子数据内容不变：`app_user(1)`、内置 `dict_item`、占位事项"纸本未注明事项"。第二阶段接入登录后种子用户的处理见 `07-登录与鉴权.md`。

### 执行方式

```bash
mkdir -p backend/data
sqlite3 backend/data/harmony.db < backend/migrate/sqlite/schema.sql
```

本机需要 `sqlite3` 命令行（macOS 自带）。另外提供 `harmony migrate` 子命令，用同一个驱动执行快照，服务器上不安装 `sqlite3` 也能建表：

```bash
./harmony migrate -config /etc/harmony/config.yaml
```

`migrate` 子命令通过 `//go:embed` 把 `schema.sql` 打进二进制，只执行快照，不做差异比对。快照用 `IF NOT EXISTS` 和 `ON CONFLICT` 写成，重复执行不会破坏已有数据。

## 日期与时间列

这是 SQLite 切换里最容易出错的地方。SQLite 没有日期类型，比较全靠字符串字典序，所以存进去的格式必须固定。

### 日期列

涉及 `record.occurred_on`、`event.event_date`。

现状：model 里是 `time.Time` 加 `type:date` 标签。MySQL 驱动会存成 `2011-10-01`；SQLite 驱动会存成 `2011-10-01 00:00:00+08:00`。后者与查询参数 `'2011-10-31'` 做 `<=` 比较时，`2011-10-31 00:00:00+08:00` 大于 `2011-10-31`，当天的记录会被漏掉。

方案：在 `infra/utils` 新增 `Date` 类型，所有日期列改用它。

```go
// Date 是只精确到日的日期，数据库里固定存成 YYYY-MM-DD 文本，字典序即时间序。
type Date struct {
	time.Time
}

func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.Format(time.DateOnly), nil
}

func (d *Date) Scan(src any) error {
	// 兼容 string 与 []byte；空值置零；格式只接受 YYYY-MM-DD
}
```

- `record.OccurredOn` 改为 `utils.Date`，`event.EventDate` 改为 `*utils.Date`。
- store 里所有日期条件的参数一律传 `utils.DateString(t)`（已有），不要传 `time.Time`。
- `MIN(occurred_on)`、`MAX(occurred_on)` 扫描到 `string`，再用 `utils.ParseDate` 转换。

### 时间列

涉及 `created_at`、`updated_at`、`merged_at`、`finished_at`。

继续用 `time.Time`，由 GORM 维护。要求进程时区固定：服务器 systemd 单元设置 `TZ=Asia/Shanghai`，本机开发同样是东八区。时间列只用于排序和转成毫秒输出，不参与按日期范围的查询，字典序在同一时区下与时间序一致。

如果以后需要按时间列做范围查询，再统一改成 `INTEGER` 毫秒存储，本次不改。

## SQL 方言改动

全仓搜索到 6 处 MySQL 专用写法，逐一替换：

| 位置 | 现写法 | 改为 |
| --- | --- | --- |
| `stats/store/store.go` 的 `Years` | `DISTINCT YEAR(occurred_on)` | `DISTINCT CAST(substr(occurred_on, 1, 4) AS INTEGER)` |
| `stats/store/store.go` 按月汇总 | `MONTH(occurred_on)` | `CAST(substr(occurred_on, 6, 2) AS INTEGER)` |
| `book/store/store.go` 的 `MovePageChecks` | `INSERT ... SELECT ... ON DUPLICATE KEY UPDATE` | `INSERT ... SELECT ... WHERE true ON CONFLICT(owner_id, book_id, paper_page) DO UPDATE SET declared_cent = excluded.declared_cent, updated_at = excluded.updated_at` |
| 同上 | `NOW(3)` | 由 Go 传入 `time.Now()` 参数 |
| `book/store/store.go` 合并时写 `merged_at` | `gorm.Expr("NOW(3)")` | `time.Now()` |
| `book/store`、`record/store` 的 `LockBooks`、`LockRecords` | `clause.Locking{Strength: "UPDATE"}` | 删除锁子句，改为普通查询 |

关于最后一项：SQLite 没有行锁，也不支持 `SELECT ... FOR UPDATE` 语法，保留锁子句要么报语法错误、要么被驱动静默丢弃，两种情况都不能提供保护。并发安全改由 `_txlock=immediate` 保证：事务开始时就持有整库写锁，同一时刻只有一个写事务在执行，原来靠行锁防止的"并发重复合并""并发重复回礼"自然不会发生。删除锁子句是为了不让读代码的人误以为还有行锁，方法名改为 `GetBooksInTx`、`GetRecordsInTx`，注释写明依赖写事务串行。

`paper_page_check` 的 upsert（识别批量写入时调用）用 GORM 的 `clause.OnConflict`，不手写 SQL：

```go
db.Clauses(clause.OnConflict{
	Columns:   []clause.Column{{Name: "owner_id"}, {Name: "book_id"}, {Name: "paper_page"}},
	DoUpdates: clause.AssignmentColumns([]string{"declared_cent", "updated_at"}),
}).Create(&check)
```

其余写法两边通用，不需要改：`COALESCE`、`CASE WHEN`、`COUNT(DISTINCT ...)`、`LIKE`、`IN ?`、`LIMIT/OFFSET`。

### 搜索的大小写

MySQL 的 `utf8mb4_0900_ai_ci` 不区分大小写；SQLite 的 `LIKE` 默认只对 ASCII 字母不区分大小写。搜索依赖的拼音和首字母都是小写 ASCII，中文没有大小写，行为一致。写入拼音时继续统一转小写，查询关键字同样转小写。

### 写入只读列

`total_cent` 是生成列，model 上已经有 `->` 只读标签，GORM 不会写它，SQLite 下同样适用。

## 测试

`backend/test/main_test.go` 改为：

1. `TestMain` 在 `os.MkdirTemp` 下创建临时文件 `harmony_test.db`，用与线上相同的 DSN 参数打开。
1. 执行内嵌的 `schema.sql`。
1. 每个用例照旧开启事务、结束时回滚；嵌套事务走 savepoint，SQLite 支持。
1. 进程结束时删除临时目录。

测试不再依赖本机 MySQL，`go test ./...` 在任何机器上都能直接运行。

新增两个只针对 SQLite 的用例：

| 用例 | 断言 |
| --- | --- |
| 日期边界 | 记录日期 2011-10-31，查询 `to=2011-10-31` 能查到；查询 `from=2011-11-01` 查不到 |
| 并发写 | 10 个 goroutine 同时对同一条收礼记录发起回礼，最终只有一条成功把状态改为已回礼，其余返回 409，没有 `database is locked` 错误 |

## 现有数据

本机 MySQL 的 `harmony` 库目前只有开发测试数据（25 条记录、15 个亲友、1 个识别任务），切换时不迁移，直接用快照建新库。`backend/data/uploads/` 下的测试照片一并删除。

以后如果需要把一个 SQLite 库搬到另一台机器，停服务后复制 `harmony.db`（WAL 模式下要连同 `harmony.db-wal`、`harmony.db-shm` 一起复制，或者先执行一次 `PRAGMA wal_checkpoint(TRUNCATE)`）和 `uploads/` 目录即可。

## 备份

备份方案放在 `09-部署上线.md` 的"备份与恢复"一节，这里只说明原则：

- 不能直接 `cp` 正在写入的数据库文件，要用 SQLite 的在线备份：`VACUUM INTO '<目标文件>'`。
- `harmony backup` 子命令执行 `VACUUM INTO`，产出一个完整、自洽、已压缩的单文件，服务不用停。

## 改动清单

| 文件 | 改动 |
| --- | --- |
| `backend/go.mod` | 删除 `gorm.io/driver/mysql`、`gorm.io/driver/postgres`、`gorm.io/driver/sqlite`，新增 `github.com/glebarez/sqlite` |
| `backend/infra/rdb/rdb.go` | 只保留 SQLite；拼接 pragma；设置连接池；启动时检查 WAL |
| `backend/infra/consts/consts.go` | 删除数据库驱动常量 |
| `backend/component/config/config.go` | `DatabaseConfig` 只剩 `Path`；环境变量调整 |
| `backend/configs/config-local.yaml` | 改为 `path` |
| `backend/infra/utils` | 新增 `Date` 类型 |
| `backend/infra/model/record`、`event` | 日期字段改用 `utils.Date` |
| `backend/component/stats/store`、`book/store`、`record/store` | 方言替换、删除锁子句 |
| `backend/migrate/sqlite/schema.sql` | 新增；删除 `migrate/mysql/` |
| `backend/cmd/harmony/main.go` | 新增 `migrate`、`backup` 子命令 |
| `backend/test/main_test.go` | 改用临时 SQLite 文件 |
| `backend/Makefile` | `CGO_ENABLED=0`；新增 `migrate` 目标 |
| `.gitignore` | 忽略 `backend/data/*.db*` |
| `docs/00`、`docs/01`、`docs/02`、`README.md`、`AGENTS.md` | MySQL 相关描述改为 SQLite |

## 验收

1. `CGO_ENABLED=0 go build ./...`、`go vet ./...`、`go test ./...` 全部通过，测试过程不连接 MySQL。
1. `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/harmony` 在 Mac 上成功产出 Linux 二进制。
1. 全新目录下执行 `harmony migrate` 后启动，健康检查返回 `database: sqlite`、`journalMode: wal`。
1. 在 H5 上走一遍 `fen.md` 9.1 节全部条目，结果与 MySQL 时一致。
1. 仓库中搜索 `mysql`（不区分大小写）只在历史文档的变更说明里出现。
