
CREATE TABLE IF NOT EXISTS campaign_performances (
  id UUID PRIMARY KEY,
  campaign_id UUID NOT NULL,
  impressions INTEGER NOT NULL DEFAULT 0,
  clicks INTEGER NOT NULL DEFAULT 0,
  conversions INTEGER NOT NULL DEFAULT 0,
  purchases INTEGER NOT NULL DEFAULT 0,
  last_updated_at TIMESTAMPTZ NOT NULL,

  CONSTRAINT fk_campaign_performances_campaign
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE,

  -- Non-negative counts
  CHECK (impressions >= 0),
  CHECK (clicks >= 0),
  CHECK (conversions >= 0),
  CHECK (purchases >= 0),

  -- Monotone relations
  CHECK (clicks <= impressions),
  CHECK (conversions <= clicks),
  CHECK (purchases <= conversions)
);

CREATE INDEX IF NOT EXISTS idx_campaign_performances_campaign_id ON campaign_performances (campaign_id);
CREATE INDEX IF NOT EXISTS idx_campaign_performances_last_updated_at ON campaign_performances (last_updated_at);
