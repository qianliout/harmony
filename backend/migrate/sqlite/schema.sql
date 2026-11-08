CREATE TABLE IF NOT EXISTS app_user (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  openid     TEXT NULL,
  unionid    TEXT NULL,
  nickname   TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_app_user_openid ON app_user(openid);

CREATE TABLE IF NOT EXISTS auth_token (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id      INTEGER NOT NULL,
  token_hash   TEXT NOT NULL,
  client       TEXT NOT NULL,
  device       TEXT NOT NULL DEFAULT '',
  created_at   DATETIME NOT NULL,
  expires_at   DATETIME NOT NULL,
  last_used_at DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_auth_token_hash ON auth_token(token_hash);
CREATE INDEX IF NOT EXISTS idx_auth_token_user ON auth_token(user_id);

CREATE TABLE IF NOT EXISTS dict_item (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id   INTEGER NOT NULL,
  dict_type  TEXT NOT NULL,
  code       TEXT NOT NULL,
  label      TEXT NOT NULL,
  color      TEXT NOT NULL DEFAULT '',
  sort       INTEGER NOT NULL DEFAULT 0,
  builtin    INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_dict_item_owner_type_code ON dict_item(owner_id, dict_type, code);

CREATE TABLE IF NOT EXISTS person (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id      INTEGER NOT NULL,
  name          TEXT NOT NULL,
  name_pinyin   TEXT NOT NULL DEFAULT '',
  name_initials TEXT NOT NULL DEFAULT '',
  relation      TEXT NOT NULL DEFAULT 'other',
  phone         TEXT NOT NULL DEFAULT '',
  remark        TEXT NOT NULL DEFAULT '',
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_person_owner_name ON person(owner_id, name);
CREATE INDEX IF NOT EXISTS idx_person_owner_initials ON person(owner_id, name_initials);

CREATE TABLE IF NOT EXISTS book (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id         INTEGER NOT NULL,
  name             TEXT NOT NULL,
  category         TEXT NOT NULL DEFAULT 'renqing',
  default_event_id INTEGER NULL,
  description      TEXT NOT NULL DEFAULT '',
  status           TEXT NOT NULL DEFAULT 'normal',
  merged_into_id   INTEGER NULL,
  merged_at        DATETIME NULL,
  created_at       DATETIME NOT NULL,
  updated_at       DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_book_owner_status ON book(owner_id, status);

CREATE TABLE IF NOT EXISTS event (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id       INTEGER NOT NULL,
  name           TEXT NOT NULL,
  type           TEXT NOT NULL DEFAULT 'other',
  event_date     TEXT NULL,
  location       TEXT NOT NULL DEFAULT '',
  host           TEXT NOT NULL DEFAULT 'other',
  host_person_id INTEGER NULL,
  date_precision TEXT NOT NULL DEFAULT 'day',
  placeholder    INTEGER NOT NULL DEFAULT 0,
  remark         TEXT NOT NULL DEFAULT '',
  created_at     DATETIME NOT NULL,
  updated_at     DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_event_owner_date ON event(owner_id, event_date);
CREATE INDEX IF NOT EXISTS idx_event_owner_host_person ON event(owner_id, host_person_id);

CREATE TABLE IF NOT EXISTS record (
  id                INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id          INTEGER NOT NULL,
  person_id         INTEGER NOT NULL,
  event_id          INTEGER NOT NULL,
  book_id           INTEGER NOT NULL,
  origin_book_id    INTEGER NOT NULL,
  item_kind         TEXT NOT NULL,
  direction         TEXT NOT NULL,
  amount_cent       INTEGER NOT NULL DEFAULT 0,
  gift_name         TEXT NOT NULL DEFAULT '',
  gift_value_cent   INTEGER NOT NULL DEFAULT 0,
  total_cent        INTEGER GENERATED ALWAYS AS (amount_cent + gift_value_cent) STORED,
  occurred_on       TEXT NOT NULL,
  date_precision    TEXT NOT NULL DEFAULT 'day',
  status            TEXT NOT NULL,
  remark            TEXT NOT NULL DEFAULT '',
  paper_note        TEXT NOT NULL DEFAULT '',
  paper_page        TEXT NOT NULL DEFAULT '',
  paper_seq         INTEGER NOT NULL DEFAULT 0,
  source            TEXT NOT NULL DEFAULT 'manual',
  recognize_task_id INTEGER NULL,
  created_at        DATETIME NOT NULL,
  updated_at        DATETIME NOT NULL,
  CONSTRAINT ck_amount_nonneg CHECK (amount_cent >= 0 AND gift_value_cent >= 0),
  CONSTRAINT ck_amount_or_gift CHECK (amount_cent > 0 OR gift_name <> '')
);
CREATE INDEX IF NOT EXISTS idx_record_owner_person_date ON record(owner_id, person_id, occurred_on);
CREATE INDEX IF NOT EXISTS idx_record_owner_event ON record(owner_id, event_id);
CREATE INDEX IF NOT EXISTS idx_record_owner_book ON record(owner_id, book_id);
CREATE INDEX IF NOT EXISTS idx_record_owner_dir_status ON record(owner_id, direction, status, occurred_on);
CREATE INDEX IF NOT EXISTS idx_record_owner_date ON record(owner_id, occurred_on);
CREATE INDEX IF NOT EXISTS idx_record_dup ON record(owner_id, person_id, event_id, book_id, item_kind);
CREATE INDEX IF NOT EXISTS idx_record_owner_book_page ON record(owner_id, book_id, paper_page, paper_seq);

CREATE TABLE IF NOT EXISTS paper_page_check (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id      INTEGER NOT NULL,
  book_id       INTEGER NOT NULL,
  paper_page    TEXT NOT NULL,
  declared_cent INTEGER NOT NULL,
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_paper_page_check_book_page ON paper_page_check(owner_id, book_id, paper_page);

CREATE TABLE IF NOT EXISTS record_repay (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id         INTEGER NOT NULL,
  repay_record_id  INTEGER NOT NULL,
  origin_record_id INTEGER NOT NULL,
  created_at       DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_record_repay_pair ON record_repay(repay_record_id, origin_record_id);
CREATE INDEX IF NOT EXISTS idx_record_repay_origin ON record_repay(origin_record_id);

CREATE TABLE IF NOT EXISTS book_merge_log (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id       INTEGER NOT NULL,
  target_book_id INTEGER NOT NULL,
  source_books   TEXT NOT NULL,
  record_count   INTEGER NOT NULL,
  cross_category INTEGER NOT NULL DEFAULT 0,
  created_at     DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_book_merge_log_owner ON book_merge_log(owner_id, created_at);

CREATE TABLE IF NOT EXISTS recognize_task (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id      INTEGER NOT NULL,
  image_path    TEXT NOT NULL,
  image_sha256  TEXT NOT NULL,
  image_size    INTEGER NOT NULL,
  status        TEXT NOT NULL,
  provider      TEXT NOT NULL DEFAULT '',
  model         TEXT NOT NULL DEFAULT '',
  raw_response  TEXT NULL,
  items         TEXT NULL,
  error_message TEXT NOT NULL DEFAULT '',
  committed     INTEGER NOT NULL DEFAULT 0,
  created_at    DATETIME NOT NULL,
  finished_at   DATETIME NULL
);
CREATE INDEX IF NOT EXISTS idx_recognize_task_owner_created ON recognize_task(owner_id, created_at);

CREATE TABLE IF NOT EXISTS user_setting (
  owner_id   INTEGER NOT NULL,
  k          TEXT NOT NULL,
  v          TEXT NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (owner_id, k)
);

INSERT INTO app_user (id, nickname, created_at, updated_at)
VALUES (1, '本机用户', strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime'), strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime'))
ON CONFLICT(id) DO UPDATE SET nickname = excluded.nickname;

INSERT INTO dict_item (owner_id, dict_type, code, label, color, sort, builtin, created_at, updated_at) VALUES
  (1, 'book_category', 'renqing', '人情礼', 'cinnabar', 10, 1, strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime'), strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime')),
  (1, 'book_category', 'huopao',  '火炮',   'amber',    20, 1, strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime'), strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime')),
  (1, 'book_category', 'other',   '其他',   'stone',    30, 1, strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime'), strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime')),
  (1, 'item_kind',     'cash',    '现金',   'cinnabar', 10, 1, strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime'), strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime')),
  (1, 'item_kind',     'huopao',  '火炮',   'amber',    20, 1, strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime'), strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime')),
  (1, 'item_kind',     'gift',    '实物礼品', 'teal',   30, 1, strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime'), strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime'))
ON CONFLICT(owner_id, dict_type, code) DO UPDATE SET label = excluded.label, color = excluded.color, sort = excluded.sort;

INSERT INTO event (owner_id, name, type, event_date, location, host, date_precision, placeholder, remark, created_at, updated_at)
SELECT 1, '纸本未注明事项', 'other', NULL, '', 'other', 'day', 1, '', strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime'), strftime('%Y-%m-%d %H:%M:%f', 'now', 'localtime')
WHERE NOT EXISTS (SELECT 1 FROM event WHERE owner_id = 1 AND placeholder = 1);
