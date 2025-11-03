
-- MintRequests DDL generated from domain/mintRequest entity.
CREATE TABLE IF NOT EXISTS mint_requests (
  id UUID PRIMARY KEY,
  token_blueprint_id TEXT NOT NULL,
  production_id TEXT NOT NULL,
  mint_quantity INTEGER NOT NULL CHECK (mint_quantity > 0),
  burn_date DATE NULL,
  status TEXT NOT NULL DEFAULT 'planning' CHECK (status IN ('planning','requested','minted')),
  requested_by TEXT,
  requested_at TIMESTAMPTZ,
  minted_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL,
  created_by UUID NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
  updated_at TIMESTAMPTZ NOT NULL,
  updated_by UUID NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
  deleted_at TIMESTAMPTZ,
  deleted_by UUID REFERENCES members(id) ON DELETE RESTRICT,

  -- Non-empty checks
  CONSTRAINT chk_mint_requests_ids_non_empty CHECK (
    char_length(trim(id::text)) > 0
    AND char_length(trim(token_blueprint_id)) > 0
    AND char_length(trim(production_id)) > 0
  ),

  -- Audit coherence
  CONSTRAINT chk_mint_requests_time_order CHECK (
    updated_at >= created_at
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  ),
  CONSTRAINT chk_mint_requests_deleted_pair CHECK (
    (deleted_at IS NULL AND deleted_by IS NULL) OR
    (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)
  ),

  -- Coherence with status (mirrors entity validation)
  CONSTRAINT chk_mint_requests_status_coherence CHECK (
    (status = 'planning'  AND requested_by IS NULL AND requested_at IS NULL AND minted_at IS NULL) OR
    (status = 'requested' AND requested_by IS NOT NULL AND requested_at IS NOT NULL AND minted_at IS NULL) OR
    (status = 'minted'    AND requested_by IS NOT NULL AND requested_at IS NOT NULL AND minted_at IS NOT NULL AND minted_at >= requested_at)
  )
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_mint_requests_status               ON mint_requests(status);
CREATE INDEX IF NOT EXISTS idx_mint_requests_token_blueprint_id   ON mint_requests(token_blueprint_id);
CREATE INDEX IF NOT EXISTS idx_mint_requests_production_id        ON mint_requests(production_id);
CREATE INDEX IF NOT EXISTS idx_mint_requests_burn_date            ON mint_requests(burn_date);
CREATE INDEX IF NOT EXISTS idx_mint_requests_created_at           ON mint_requests(created_at);
CREATE INDEX IF NOT EXISTS idx_mint_requests_updated_at           ON mint_requests(updated_at);
CREATE INDEX IF NOT EXISTS idx_mint_requests_deleted_at           ON mint_requests(deleted_at);
CREATE INDEX IF NOT EXISTS idx_mint_requests_created_by           ON mint_requests(created_by);
CREATE INDEX IF NOT EXISTS idx_mint_requests_updated_by           ON mint_requests(updated_by);
CREATE INDEX IF NOT EXISTS idx_mint_requests_deleted_by           ON mint_requests(deleted_by);
