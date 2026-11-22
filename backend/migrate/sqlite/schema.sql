CREATE TABLE IF NOT EXISTS app_user (
  id         INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  nickname   TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS person (
  id            INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  owner_id      INTEGER NOT NULL DEFAULT 0,
  name          TEXT NOT NULL DEFAULT '',
  remark        TEXT NOT NULL DEFAULT '',
  name_pinyin   TEXT NOT NULL DEFAULT '',
  name_initials TEXT NOT NULL DEFAULT '',
  created_at    INTEGER NOT NULL DEFAULT 0,
  updated_at    INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_person_owner_name_remark ON person(owner_id, name, remark);

CREATE TABLE IF NOT EXISTS book (
  id          INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  owner_id    INTEGER NOT NULL DEFAULT 0,
  name        TEXT NOT NULL DEFAULT '',
  category    TEXT NOT NULL DEFAULT 'renqing',
  description TEXT NOT NULL DEFAULT '',
  occurred_on TEXT NOT NULL DEFAULT '',
  created_at  INTEGER NOT NULL DEFAULT 0,
  updated_at  INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_book_owner_name ON book(owner_id, name);

CREATE TABLE IF NOT EXISTS record (
  id          INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  owner_id    INTEGER NOT NULL DEFAULT 0,
  person_id   INTEGER NOT NULL DEFAULT 0,
  book_id     INTEGER NOT NULL DEFAULT 0,
  direction   TEXT NOT NULL DEFAULT 'in',
  amount_cent INTEGER NOT NULL DEFAULT 0,
  matter      TEXT NOT NULL DEFAULT '',
  note        TEXT NOT NULL DEFAULT '',
  occurred_on TEXT NOT NULL DEFAULT '',
  created_at  INTEGER NOT NULL DEFAULT 0,
  updated_at  INTEGER NOT NULL DEFAULT 0,
  CONSTRAINT ck_record_direction CHECK (direction IN ('in', 'out')),
  CONSTRAINT ck_record_amount CHECK (amount_cent >= 0),
  CONSTRAINT ck_record_in_has_book CHECK (direction <> 'in' OR book_id > 0)
);
CREATE INDEX IF NOT EXISTS idx_record_owner_person ON record(owner_id, person_id, occurred_on);
CREATE INDEX IF NOT EXISTS idx_record_owner_book ON record(owner_id, book_id, occurred_on);
CREATE INDEX IF NOT EXISTS idx_record_owner_direction ON record(owner_id, direction, occurred_on);

CREATE TABLE IF NOT EXISTS recognize_task (
  id            INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  owner_id      INTEGER NOT NULL DEFAULT 0,
  image_path    TEXT NOT NULL DEFAULT '',
  image_sha256  TEXT NOT NULL DEFAULT '',
  image_size    INTEGER NOT NULL DEFAULT 0,
  status        TEXT NOT NULL DEFAULT 'pending',
  provider      TEXT NOT NULL DEFAULT '',
  model         TEXT NOT NULL DEFAULT '',
  raw_response  TEXT NOT NULL DEFAULT '',
  items         TEXT NOT NULL DEFAULT '',
  sum_cent      INTEGER NOT NULL DEFAULT 0,
  error_message TEXT NOT NULL DEFAULT '',
  committed     TEXT NOT NULL DEFAULT 'false',
  created_at    INTEGER NOT NULL DEFAULT 0,
  updated_at    INTEGER NOT NULL DEFAULT 0,
  finished_at   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_recognize_task_owner_created ON recognize_task(owner_id, created_at);

INSERT INTO app_user (id, nickname, created_at, updated_at)
VALUES (1, '本机用户', CAST(strftime('%s', 'now') AS INTEGER) * 1000, CAST(strftime('%s', 'now') AS INTEGER) * 1000)
ON CONFLICT(id) DO UPDATE SET nickname = excluded.nickname;
