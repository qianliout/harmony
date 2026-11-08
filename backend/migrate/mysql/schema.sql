SET NAMES utf8mb4;

-- harmony 全量建表快照（MySQL 8，utf8mb4/utf8mb4_0900_ai_ci）。
-- 服务启动不自动迁移：改 schema 直接改本文件，再人工执行到目标库。
-- 用法：docker exec -i mysql mysql -uroot -proot harmony < backend/migrate/mysql/schema.sql

CREATE TABLE IF NOT EXISTS app_user (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  openid      VARCHAR(64)  NULL,
  unionid     VARCHAR(64)  NULL,
  nickname    VARCHAR(64)  NOT NULL DEFAULT '',
  created_at  DATETIME(3)  NOT NULL,
  updated_at  DATETIME(3)  NOT NULL,
  UNIQUE KEY uk_openid (openid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS dict_item (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  owner_id    BIGINT UNSIGNED NOT NULL,
  dict_type   VARCHAR(32)  NOT NULL,
  code        VARCHAR(32)  NOT NULL,
  label       VARCHAR(32)  NOT NULL,
  color       VARCHAR(16)  NOT NULL DEFAULT '',
  sort        INT          NOT NULL DEFAULT 0,
  builtin     TINYINT(1)   NOT NULL DEFAULT 0,
  created_at  DATETIME(3)  NOT NULL,
  updated_at  DATETIME(3)  NOT NULL,
  UNIQUE KEY uk_owner_type_code (owner_id, dict_type, code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS person (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  owner_id      BIGINT UNSIGNED NOT NULL,
  name          VARCHAR(32)  NOT NULL,
  name_pinyin   VARCHAR(128) NOT NULL DEFAULT '',
  name_initials VARCHAR(32)  NOT NULL DEFAULT '',
  relation      VARCHAR(16)  NOT NULL DEFAULT 'other',
  phone         VARCHAR(32)  NOT NULL DEFAULT '',
  remark        VARCHAR(255) NOT NULL DEFAULT '',
  created_at    DATETIME(3)  NOT NULL,
  updated_at    DATETIME(3)  NOT NULL,
  KEY idx_owner_name (owner_id, name),
  KEY idx_owner_initials (owner_id, name_initials)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS book (
  id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  owner_id         BIGINT UNSIGNED NOT NULL,
  name             VARCHAR(64)  NOT NULL,
  category         VARCHAR(32)  NOT NULL DEFAULT 'renqing',
  default_event_id BIGINT UNSIGNED NULL,
  description      VARCHAR(255) NOT NULL DEFAULT '',
  status           VARCHAR(16)  NOT NULL DEFAULT 'normal',
  merged_into_id   BIGINT UNSIGNED NULL,
  merged_at        DATETIME(3)  NULL,
  created_at       DATETIME(3)  NOT NULL,
  updated_at       DATETIME(3)  NOT NULL,
  KEY idx_owner_status (owner_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS event (
  id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  owner_id       BIGINT UNSIGNED NOT NULL,
  name           VARCHAR(64)  NOT NULL,
  type           VARCHAR(16)  NOT NULL DEFAULT 'other',
  event_date     DATE         NULL,
  location       VARCHAR(128) NOT NULL DEFAULT '',
  host           VARCHAR(16)  NOT NULL DEFAULT 'other',
  host_person_id BIGINT UNSIGNED NULL,
  date_precision VARCHAR(8)   NOT NULL DEFAULT 'day',
  placeholder    TINYINT(1)   NOT NULL DEFAULT 0,
  remark         VARCHAR(255) NOT NULL DEFAULT '',
  created_at     DATETIME(3)  NOT NULL,
  updated_at     DATETIME(3)  NOT NULL,
  KEY idx_owner_date (owner_id, event_date),
  KEY idx_owner_host_person (owner_id, host_person_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `record` (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  owner_id          BIGINT UNSIGNED NOT NULL,
  person_id         BIGINT UNSIGNED NOT NULL,
  event_id          BIGINT UNSIGNED NOT NULL,
  book_id           BIGINT UNSIGNED NOT NULL,
  origin_book_id    BIGINT UNSIGNED NOT NULL,
  item_kind         VARCHAR(32)   NOT NULL,
  direction         VARCHAR(8)    NOT NULL,
  amount_cent       BIGINT        NOT NULL DEFAULT 0,
  gift_name         VARCHAR(64)   NOT NULL DEFAULT '',
  gift_value_cent   BIGINT        NOT NULL DEFAULT 0,
  total_cent        BIGINT AS (amount_cent + gift_value_cent) STORED,
  occurred_on       DATE          NOT NULL,
  date_precision    VARCHAR(8)    NOT NULL DEFAULT 'day',
  status            VARCHAR(16)   NOT NULL,
  remark            VARCHAR(255)  NOT NULL DEFAULT '',
  paper_note        VARCHAR(255)  NOT NULL DEFAULT '',
  paper_page        VARCHAR(16)   NOT NULL DEFAULT '',
  paper_seq         INT           NOT NULL DEFAULT 0,
  source            VARCHAR(16)   NOT NULL DEFAULT 'manual',
  recognize_task_id BIGINT UNSIGNED NULL,
  created_at        DATETIME(3)   NOT NULL,
  updated_at        DATETIME(3)   NOT NULL,
  CONSTRAINT ck_amount_nonneg CHECK (amount_cent >= 0 AND gift_value_cent >= 0),
  CONSTRAINT ck_amount_or_gift CHECK (amount_cent > 0 OR gift_name <> ''),
  KEY idx_owner_person_date (owner_id, person_id, occurred_on),
  KEY idx_owner_event (owner_id, event_id),
  KEY idx_owner_book (owner_id, book_id),
  KEY idx_owner_dir_status (owner_id, direction, status, occurred_on),
  KEY idx_owner_date (owner_id, occurred_on),
  KEY idx_dup (owner_id, person_id, event_id, book_id, item_kind),
  KEY idx_owner_book_page (owner_id, book_id, paper_page, paper_seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS paper_page_check (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  owner_id      BIGINT UNSIGNED NOT NULL,
  book_id       BIGINT UNSIGNED NOT NULL,
  paper_page    VARCHAR(16)  NOT NULL,
  declared_cent BIGINT       NOT NULL,
  created_at    DATETIME(3)  NOT NULL,
  updated_at    DATETIME(3)  NOT NULL,
  UNIQUE KEY uk_book_page (owner_id, book_id, paper_page)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS record_repay (
  id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  owner_id         BIGINT UNSIGNED NOT NULL,
  repay_record_id  BIGINT UNSIGNED NOT NULL,
  origin_record_id BIGINT UNSIGNED NOT NULL,
  created_at       DATETIME(3)  NOT NULL,
  UNIQUE KEY uk_pair (repay_record_id, origin_record_id),
  KEY idx_origin (origin_record_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS book_merge_log (
  id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  owner_id       BIGINT UNSIGNED NOT NULL,
  target_book_id BIGINT UNSIGNED NOT NULL,
  source_books   JSON          NOT NULL,
  record_count   INT           NOT NULL,
  cross_category TINYINT(1)    NOT NULL DEFAULT 0,
  created_at     DATETIME(3)   NOT NULL,
  KEY idx_owner (owner_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS recognize_task (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  owner_id      BIGINT UNSIGNED NOT NULL,
  image_path    VARCHAR(255) NOT NULL,
  image_sha256  CHAR(64)     NOT NULL,
  image_size    INT          NOT NULL,
  status        VARCHAR(16)  NOT NULL,
  provider      VARCHAR(32)  NOT NULL DEFAULT '',
  model         VARCHAR(64)  NOT NULL DEFAULT '',
  raw_response  MEDIUMTEXT   NULL,
  items         JSON         NULL,
  error_message VARCHAR(512) NOT NULL DEFAULT '',
  committed     TINYINT(1)   NOT NULL DEFAULT 0,
  created_at    DATETIME(3)  NOT NULL,
  finished_at   DATETIME(3)  NULL,
  KEY idx_owner_created (owner_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user_setting (
  owner_id   BIGINT UNSIGNED NOT NULL,
  k          VARCHAR(64)   NOT NULL,
  v          VARCHAR(1024) NOT NULL,
  updated_at DATETIME(3)   NOT NULL,
  PRIMARY KEY (owner_id, k)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 种子数据：第一阶段只有本机用户 id=1。
INSERT INTO app_user (id, nickname, created_at, updated_at)
VALUES (1, '本机用户', NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE nickname = VALUES(nickname);

INSERT INTO dict_item (id, owner_id, dict_type, code, label, color, sort, builtin, created_at, updated_at) VALUES
  (1, 1, 'book_category', 'renqing', '人情礼', 'cinnabar', 10, 1, NOW(3), NOW(3)),
  (2, 1, 'book_category', 'huopao',  '火炮',   'amber',    20, 1, NOW(3), NOW(3)),
  (3, 1, 'book_category', 'other',   '其他',   'stone',    30, 1, NOW(3), NOW(3)),
  (4, 1, 'item_kind',     'cash',    '现金',   'cinnabar', 10, 1, NOW(3), NOW(3)),
  (5, 1, 'item_kind',     'huopao',  '火炮',   'amber',    20, 1, NOW(3), NOW(3)),
  (6, 1, 'item_kind',     'gift',    '实物礼品', 'teal',    30, 1, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE label = VALUES(label), color = VALUES(color), sort = VALUES(sort);

-- 占位事项：纸本标注没有写对方事项时归到这里，事项列表默认隐藏。
INSERT INTO event (owner_id, name, type, event_date, location, host, date_precision, placeholder, remark, created_at, updated_at)
SELECT 1, '纸本未注明事项', 'other', NULL, '', 'other', 'day', 1, '', NOW(3), NOW(3)
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM event WHERE owner_id = 1 AND placeholder = 1
);

UPDATE event SET name = '纸本未注明事项' WHERE owner_id = 1 AND placeholder = 1;
