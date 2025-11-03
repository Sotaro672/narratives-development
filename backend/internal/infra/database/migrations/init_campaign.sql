
CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    brand_id UUID NOT NULL,
    assignee_id TEXT NOT NULL,
    list_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft','active','paused','scheduled','completed','deleted')),
    budget NUMERIC(18,2) NOT NULL DEFAULT 0,
    spent NUMERIC(18,2) NOT NULL DEFAULT 0,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    target_audience TEXT NOT NULL,
    ad_type TEXT NOT NULL CHECK (ad_type IN ('image_carousel','video','story','reel','banner','native')),
    headline VARCHAR(120) NOT NULL,
    description TEXT NOT NULL,
    performance_id TEXT NULL,
    image_id TEXT NULL,
    created_by TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by TEXT NULL,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,
    deleted_by TEXT NULL,
    CONSTRAINT fk_campaigns_brand
        FOREIGN KEY (brand_id)
        REFERENCES brands(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,
    CHECK (budget >= 0),
    CHECK (spent >= 0),
    CHECK (end_date >= start_date),
    CHECK (updated_at IS NULL OR updated_at >= created_at),
    CHECK (deleted_at IS NULL OR deleted_at >= created_at)
);

CREATE INDEX IF NOT EXISTS idx_campaigns_brand_id ON campaigns(brand_id);
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);
CREATE INDEX IF NOT EXISTS idx_campaigns_assignee_id ON campaigns(assignee_id);
CREATE INDEX IF NOT EXISTS idx_campaigns_list_id ON campaigns(list_id);
CREATE INDEX IF NOT EXISTS idx_campaigns_created_at ON campaigns(created_at);
CREATE INDEX IF NOT EXISTS idx_campaigns_start_date ON campaigns(start_date);
CREATE INDEX IF NOT EXISTS idx_campaigns_end_date ON campaigns(end_date);
CREATE INDEX IF NOT EXISTS idx_campaigns_deleted_at ON campaigns(deleted_at);
