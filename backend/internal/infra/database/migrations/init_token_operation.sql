
-- Migration: Initialize TokenOperation domain and related tables
-- Mirrors backend/internal/domain/tokenOperation/entity.go and repository_pg.go

BEGIN;

-- token_operations: minimal operational state tracking
CREATE TABLE IF NOT EXISTS token_operations (
  id                  TEXT        PRIMARY KEY,
  token_blueprint_id  TEXT        NOT NULL,
  assignee_id         TEXT        NOT NULL,
  name                TEXT        NOT NULL DEFAULT '',
  status              TEXT        NOT NULL DEFAULT 'operational',
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by          TEXT        NOT NULL DEFAULT '',

  CONSTRAINT chk_to_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(token_blueprint_id)) > 0
    AND char_length(trim(assignee_id)) > 0
  )
);

-- Optional FKs
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='token_blueprints') THEN
    BEGIN
      ALTER TABLE token_operations
        ADD CONSTRAINT fk_to_tb
        FOREIGN KEY (token_blueprint_id) REFERENCES token_blueprints(id) ON DELETE RESTRICT;
    EXCEPTION WHEN duplicate_object THEN NULL;
    END;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='members') THEN
    BEGIN
      ALTER TABLE token_operations
        ADD CONSTRAINT fk_to_assignee
        FOREIGN KEY (assignee_id) REFERENCES members(id) ON DELETE RESTRICT;
    EXCEPTION WHEN duplicate_object THEN NULL;
    END;
  END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_to_token_blueprint_id ON token_operations(token_blueprint_id);
CREATE INDEX IF NOT EXISTS idx_to_assignee_id        ON token_operations(assignee_id);
CREATE INDEX IF NOT EXISTS idx_to_updated_at         ON token_operations(updated_at);

-- token_holders: wallet holders for a token
CREATE TABLE IF NOT EXISTS token_holders (
  id              TEXT        PRIMARY KEY,
  token_id        TEXT        NOT NULL,
  wallet_address  TEXT        NOT NULL,
  balance         TEXT        NOT NULL,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_th_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(token_id)) > 0
    AND char_length(trim(wallet_address)) > 0
  )
);

-- Optional FK to tokens table if it exists
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='tokens') THEN
    BEGIN
      ALTER TABLE token_holders
        ADD CONSTRAINT fk_th_token
        FOREIGN KEY (token_id) REFERENCES tokens(id) ON DELETE CASCADE;
    EXCEPTION WHEN duplicate_object THEN NULL;
    END;
  END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_th_token_id        ON token_holders(token_id);
CREATE INDEX IF NOT EXISTS idx_th_wallet_address  ON token_holders(wallet_address);
CREATE INDEX IF NOT EXISTS idx_th_updated_at      ON token_holders(updated_at);

-- token_update_history: audit of updates
CREATE TABLE IF NOT EXISTS token_update_history (
  id          TEXT        PRIMARY KEY,
  token_id    TEXT        NOT NULL,
  event       TEXT        NOT NULL,
  assignee_id TEXT        NOT NULL,
  note        TEXT        NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_tuh_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(token_id)) > 0
    AND char_length(trim(event)) > 0
  )
);

CREATE INDEX IF NOT EXISTS idx_tuh_token_id    ON token_update_history(token_id);
CREATE INDEX IF NOT EXISTS idx_tuh_created_at  ON token_update_history(created_at);

-- token_operation_contents: auxiliary contents tied to operational token
CREATE TABLE IF NOT EXISTS token_operation_contents (
  id           TEXT        PRIMARY KEY,
  token_id     TEXT        NOT NULL,
  type         TEXT        NOT NULL,
  url          TEXT        NOT NULL,
  description  TEXT        NOT NULL DEFAULT '',
  published_by TEXT        NOT NULL DEFAULT '',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_toc_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(token_id)) > 0
    AND char_length(trim(type)) > 0
    AND char_length(trim(url)) > 0
  ),
  CONSTRAINT chk_toc_url_format CHECK (url ~* '^(https?)://')
);

CREATE INDEX IF NOT EXISTS idx_toc_token_id    ON token_operation_contents(token_id);
CREATE INDEX IF NOT EXISTS idx_toc_type        ON token_operation_contents(type);
CREATE INDEX IF NOT EXISTS idx_toc_created_at  ON token_operation_contents(created_at);

-- product_details: minimal reference table used by repository
CREATE TABLE IF NOT EXISTS product_details (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_pd_name ON product_details(name);

COMMIT;
