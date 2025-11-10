// backend\internal\domain\campaignPerformance\entity.go
package campaignPerformance

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Type (mirror TS)
type CampaignPerformance struct {
	ID            string
	CampaignID    string
	Impressions   int
	Clicks        int
	Conversions   int
	Purchases     int
	LastUpdatedAt time.Time
}

// Errors
var (
	ErrInvalidID            = errors.New("campaignPerformance: invalid id")
	ErrInvalidCampaignID    = errors.New("campaignPerformance: invalid campaignId")
	ErrInvalidCounts        = errors.New("campaignPerformance: invalid counts")
	ErrInconsistentCounts   = errors.New("campaignPerformance: inconsistent counts order")
	ErrInvalidLastUpdatedAt = errors.New("campaignPerformance: invalid lastUpdatedAt")
)

// Policy
var (
	// Non-negative counts
	MinImpressions = 0
	MinClicks      = 0
	MinConversions = 0
	MinPurchases   = 0

	// Enforce monotone relations
	EnforceClicksLEImpressions    = true // clicks <= impressions
	EnforceConversionsLEClicks    = true // conversions <= clicks
	EnforcePurchasesLEConversions = true // purchases <= conversions

	// Upper bounds (0 disables checks)
	MaxImpressions = 0
	MaxClicks      = 0
	MaxConversions = 0
	MaxPurchases   = 0
)

// Constructors

func New(
	id, campaignID string,
	impressions, clicks, conversions, purchases int,
	lastUpdatedAt time.Time,
) (CampaignPerformance, error) {
	cp := CampaignPerformance{
		ID:            strings.TrimSpace(id),
		CampaignID:    strings.TrimSpace(campaignID),
		Impressions:   impressions,
		Clicks:        clicks,
		Conversions:   conversions,
		Purchases:     purchases,
		LastUpdatedAt: lastUpdatedAt.UTC(),
	}
	if err := cp.validate(); err != nil {
		return CampaignPerformance{}, err
	}
	return cp, nil
}

func NewWithNow(
	id, campaignID string,
	impressions, clicks, conversions, purchases int,
	now time.Time,
) (CampaignPerformance, error) {
	now = now.UTC()
	return New(id, campaignID, impressions, clicks, conversions, purchases, now)
}

func NewFromStringTime(
	id, campaignID string,
	impressions, clicks, conversions, purchases int,
	lastUpdatedAtStr string,
) (CampaignPerformance, error) {
	ut, err := parseTime(lastUpdatedAtStr, ErrInvalidLastUpdatedAt)
	if err != nil {
		return CampaignPerformance{}, err
	}
	return New(id, campaignID, impressions, clicks, conversions, purchases, ut)
}

// Behavior

func (c *CampaignPerformance) Touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidLastUpdatedAt
	}
	c.LastUpdatedAt = now.UTC()
	return nil
}

func (c *CampaignPerformance) UpdateCounts(impressions, clicks, conversions, purchases int, now time.Time) error {
	c.Impressions = impressions
	c.Clicks = clicks
	c.Conversions = conversions
	c.Purchases = purchases
	if err := c.validateCounts(); err != nil {
		return err
	}
	return c.Touch(now)
}

// Validation

func (c CampaignPerformance) validate() error {
	if c.ID == "" {
		return ErrInvalidID
	}
	if c.CampaignID == "" {
		return ErrInvalidCampaignID
	}
	if err := c.validateCounts(); err != nil {
		return err
	}
	if c.LastUpdatedAt.IsZero() {
		return ErrInvalidLastUpdatedAt
	}
	return nil
}

func (c CampaignPerformance) validateCounts() error {
	// lower bounds
	if c.Impressions < MinImpressions ||
		c.Clicks < MinClicks ||
		c.Conversions < MinConversions ||
		c.Purchases < MinPurchases {
		return ErrInvalidCounts
	}
	// upper bounds
	if (MaxImpressions > 0 && c.Impressions > MaxImpressions) ||
		(MaxClicks > 0 && c.Clicks > MaxClicks) ||
		(MaxConversions > 0 && c.Conversions > MaxConversions) ||
		(MaxPurchases > 0 && c.Purchases > MaxPurchases) {
		return ErrInvalidCounts
	}
	// relations
	if EnforceClicksLEImpressions && c.Clicks > c.Impressions {
		return ErrInconsistentCounts
	}
	if EnforceConversionsLEClicks && c.Conversions > c.Clicks {
		return ErrInconsistentCounts
	}
	if EnforcePurchasesLEConversions && c.Purchases > c.Conversions {
		return ErrInconsistentCounts
	}
	return nil
}

// Helpers

func parseTime(s string, classify error) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, classify
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: cannot parse %q", classify, s)
}

// ========================================
// SQL DDL
// ========================================

// CampaignPerformanceTableDDL defines the SQL for the campaign_performances table.
const CampaignPerformanceTableDDL = `
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
`
