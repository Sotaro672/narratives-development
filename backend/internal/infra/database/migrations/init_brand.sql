
CREATE TABLE IF NOT EXISTS brands (
    id UUID PRIMARY KEY,
    company_id TEXT NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(1000) NOT NULL,
    website_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    manager_id TEXT,
    wallet_address TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NULL,
    updated_at TIMESTAMPTZ NULL,
    updated_by TEXT NULL,
    deleted_at TIMESTAMPTZ NULL,
    deleted_by TEXT NULL,
    CONSTRAINT fk_brands_company
        FOREIGN KEY (company_id)
        REFERENCES companies(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,
    CONSTRAINT fk_brands_manager
        FOREIGN KEY (manager_id)
        REFERENCES members(id)
        ON UPDATE CASCADE
        ON DELETE SET NULL,
    CONSTRAINT uq_brands_company_name UNIQUE (company_id, name),
    CHECK (updated_at IS NULL OR updated_at >= created_at),
    CHECK (deleted_at IS NULL OR deleted_at >= created_at)
);
CREATE INDEX IF NOT EXISTS idx_brands_company_id ON brands(company_id);
CREATE INDEX IF NOT EXISTS idx_brands_manager_id ON brands(manager_id);
CREATE INDEX IF NOT EXISTS idx_brands_is_active ON brands(is_active);
CREATE INDEX IF NOT EXISTS idx_brands_deleted_at ON brands(deleted_at);
