package mintrequest

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// MintRequestStatus mirrors TS: 'planning' | 'requested' | 'minted'
type MintRequestStatus string

const (
	StatusPlanning  MintRequestStatus = "planning"
	StatusRequested MintRequestStatus = "requested"
	StatusMinted    MintRequestStatus = "minted"
)

var (
	ErrIDRequired                       = errors.New("id is required")
	ErrProductionIDRequired             = errors.New("productionId is required")
	ErrTokenBlueprintIDRequired         = errors.New("tokenBlueprintId is required")
	ErrRequestedByRequired              = errors.New("requestedBy is required")
	ErrInvalidStatusTransition          = errors.New("invalid status transition")
	ErrTokenBlueprintChangeOnlyPlanning = errors.New("token blueprint change is allowed only in 'planning'")
)

func IsValidStatus(s MintRequestStatus) bool {
	switch s {
	case StatusPlanning, StatusRequested, StatusMinted:
		return true
	default:
		return false
	}
}

// MintRequest mirrors TS interface:
//
//	export interface MintRequest {
//	  id: string
//	  tokenBlueprintId: string
//	  productionId: string
//	  mintQuantity: number
//	  burnDate: string | null
//	  status: MintRequestStatus
//	  requestedBy: string | null
//	  requestedAt: string | null
//	  mintedAt?: string | null
//	  createdAt: string
//	  createdBy: string
//	  updatedAt: string
//	  updatedBy: string
//	  deletedAt: string | null
//	  deletedBy: string | null
//	}
type MintRequest struct {
	ID               string
	TokenBlueprintID string
	ProductionID     string
	MintQuantity     int
	BurnDate         *time.Time
	Status           MintRequestStatus
	RequestedBy      *string
	RequestedAt      *time.Time
	MintedAt         *time.Time

	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string
	DeletedAt *time.Time
	DeletedBy *string
}

// Errors
var (
	ErrInvalidID               = errors.New("mintRequest: invalid id")
	ErrInvalidTokenBlueprintID = errors.New("mintRequest: invalid tokenBlueprintId")
	ErrInvalidProductionID     = errors.New("mintRequest: invalid productionId")
	ErrInvalidQuantity         = errors.New("mintRequest: invalid mintQuantity")
	ErrInvalidBurnDate         = errors.New("mintRequest: invalid burnDate")
	ErrInvalidStatus           = errors.New("mintRequest: invalid status")
	ErrInvalidRequestedBy      = errors.New("mintRequest: invalid requestedBy")
	ErrInvalidRequestedAt      = errors.New("mintRequest: invalid requestedAt")
	ErrInvalidMintedAt         = errors.New("mintRequest: invalid mintedAt")
	ErrInvalidCreatedAt        = errors.New("mintRequest: invalid createdAt")
	ErrInvalidCreatedBy        = errors.New("mintRequest: invalid createdBy")
	ErrInvalidUpdatedAt        = errors.New("mintRequest: invalid updatedAt")
	ErrInvalidUpdatedBy        = errors.New("mintRequest: invalid updatedBy")
	ErrInvalidDeletedAt        = errors.New("mintRequest: invalid deletedAt")
	ErrInvalidDeletedBy        = errors.New("mintRequest: invalid deletedBy")
	ErrInvalidTransition       = errors.New("mintRequest: invalid status transition")
)

// Constructors

// New creates a MintRequest with full audit fields.
func New(
	id, tokenBlueprintID, productionID string,
	mintQuantity int,
	burnDate *time.Time,
	status MintRequestStatus,
	requestedBy *string,
	requestedAt, mintedAt *time.Time,
	createdAt time.Time,
	createdBy string,
	updatedAt time.Time,
	updatedBy string,
	deletedAt *time.Time,
	deletedBy *string,
) (MintRequest, error) {
	mr := MintRequest{
		ID:               strings.TrimSpace(id),
		TokenBlueprintID: strings.TrimSpace(tokenBlueprintID),
		ProductionID:     strings.TrimSpace(productionID),
		MintQuantity:     mintQuantity,
		BurnDate:         normalizeTimePtr(burnDate),
		Status:           status,
		RequestedBy:      normalizeStringPtr(requestedBy),
		RequestedAt:      normalizeTimePtr(requestedAt),
		MintedAt:         normalizeTimePtr(mintedAt),

		CreatedAt: createdAt.UTC(),
		CreatedBy: strings.TrimSpace(createdBy),
		UpdatedAt: updatedAt.UTC(),
		UpdatedBy: strings.TrimSpace(updatedBy),
		DeletedAt: normalizeTimePtr(deletedAt),
		DeletedBy: normalizeStringPtr(deletedBy),
	}
	if mr.Status == "" {
		mr.Status = StatusPlanning
	}
	if err := mr.validate(); err != nil {
		return MintRequest{}, err
	}
	return mr, nil
}

// NewPlanning is a convenience constructor for initial planning state.
// updatedAt/updatedBy are initialized with createdAt/createdBy.
func NewPlanning(id, tokenBlueprintID, productionID string, mintQuantity int, createdAt time.Time, createdBy string) (MintRequest, error) {
	return New(
		id, tokenBlueprintID, productionID,
		mintQuantity,
		nil,
		StatusPlanning,
		nil,
		nil, nil,
		createdAt, createdBy,
		createdAt, createdBy,
		nil, nil,
	)
}

// NewFromStrings parses times from ISO8601 strings (RFC3339). Pass "" for null.
// burnDate expects a date string "2006-01-02" or full RFC3339.
func NewFromStrings(
	id, tokenBlueprintID, productionID string,
	mintQuantity int,
	burnDate string, // "" to represent null
	status MintRequestStatus,
	requestedBy string, // "" to represent null
	requestedAt string, // "" to represent null
	mintedAt string, // "" to represent null
	createdAt string,
	createdBy string,
	updatedAt string,
	updatedBy string,
	deletedAt string, // "" to represent null
	deletedBy string, // "" to represent null
) (MintRequest, error) {
	var (
		byPtr    *string
		reqPtr   *time.Time
		minPtr   *time.Time
		burnPtr  *time.Time
		delAtPtr *time.Time
		delByPtr *string
	)
	if strings.TrimSpace(requestedBy) != "" {
		by := strings.TrimSpace(requestedBy)
		byPtr = &by
	}
	if strings.TrimSpace(requestedAt) != "" {
		t, err := parseTime(requestedAt)
		if err != nil {
			return MintRequest{}, fmt.Errorf("%w: %v", ErrInvalidRequestedAt, err)
		}
		reqPtr = &t
	}
	if strings.TrimSpace(mintedAt) != "" {
		t, err := parseTime(mintedAt)
		if err != nil {
			return MintRequest{}, fmt.Errorf("%w: %v", ErrInvalidMintedAt, err)
		}
		minPtr = &t
	}
	if strings.TrimSpace(burnDate) != "" {
		t, err := parseTime(burnDate) // supports "2006-01-02"
		if err != nil {
			return MintRequest{}, fmt.Errorf("%w: %v", ErrInvalidBurnDate, err)
		}
		burnPtr = &t
	}
	ca, err := parseTime(createdAt)
	if err != nil {
		return MintRequest{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ua, err := parseTime(updatedAt)
	if err != nil {
		return MintRequest{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}
	cb := strings.TrimSpace(createdBy)
	ub := strings.TrimSpace(updatedBy)
	if cb == "" {
		return MintRequest{}, ErrInvalidCreatedBy
	}
	if ub == "" {
		return MintRequest{}, ErrInvalidUpdatedBy
	}
	if strings.TrimSpace(deletedAt) != "" {
		t, err := parseTime(deletedAt)
		if err != nil {
			return MintRequest{}, fmt.Errorf("%w: %v", ErrInvalidDeletedAt, err)
		}
		delAtPtr = &t
	}
	if strings.TrimSpace(deletedBy) != "" {
		db := strings.TrimSpace(deletedBy)
		delByPtr = &db
	}

	return New(
		id, tokenBlueprintID, productionID,
		mintQuantity,
		burnPtr,
		status,
		byPtr,
		reqPtr, minPtr,
		ca, cb,
		ua, ub,
		delAtPtr, delByPtr,
	)
}

// Behavior (state transitions)

// Request: planning -> requested (sets requestedBy/At)
func (m *MintRequest) Request(by string, at time.Time) error {
	if m.Status != StatusPlanning {
		return ErrInvalidTransition
	}
	by = strings.TrimSpace(by)
	if by == "" {
		return ErrInvalidRequestedBy
	}
	if at.IsZero() {
		return ErrInvalidRequestedAt
	}
	at = at.UTC()
	m.Status = StatusRequested
	m.RequestedBy = &by
	m.RequestedAt = &at
	m.MintedAt = nil
	return nil
}

// MarkMinted: requested -> minted (sets mintedAt)
func (m *MintRequest) MarkMinted(at time.Time) error {
	if m.Status != StatusRequested {
		return ErrInvalidTransition
	}
	if at.IsZero() {
		return ErrInvalidMintedAt
	}
	at = at.UTC()
	m.Status = StatusMinted
	m.MintedAt = &at
	return nil
}

// UpdateQuantity can be done only while planning.
func (m *MintRequest) UpdateQuantity(q int) error {
	if m.Status != StatusPlanning {
		return ErrInvalidTransition
	}
	if q <= 0 {
		return ErrInvalidQuantity
	}
	m.MintQuantity = q
	return nil
}

// Validation

func (m MintRequest) validate() error {
	if m.ID == "" {
		return ErrInvalidID
	}
	if m.TokenBlueprintID == "" {
		return ErrInvalidTokenBlueprintID
	}
	if m.ProductionID == "" {
		return ErrInvalidProductionID
	}
	if m.MintQuantity <= 0 {
		return ErrInvalidQuantity
	}
	if m.BurnDate != nil && m.BurnDate.IsZero() {
		return ErrInvalidBurnDate
	}
	if !IsValidStatus(m.Status) {
		return ErrInvalidStatus
	}
	// Coherence with status
	switch m.Status {
	case StatusPlanning:
		if m.RequestedBy != nil || m.RequestedAt != nil || m.MintedAt != nil {
			return ErrInvalidStatus
		}
	case StatusRequested:
		if m.RequestedBy == nil {
			return ErrInvalidRequestedBy
		}
		if m.RequestedAt == nil || m.RequestedAt.IsZero() {
			return ErrInvalidRequestedAt
		}
		if m.MintedAt != nil {
			return ErrInvalidMintedAt
		}
	case StatusMinted:
		if m.RequestedBy == nil {
			return ErrInvalidRequestedBy
		}
		if m.RequestedAt == nil || m.RequestedAt.IsZero() {
			return ErrInvalidRequestedAt
		}
		if m.MintedAt == nil || m.MintedAt.IsZero() {
			return ErrInvalidMintedAt
		}
		if m.MintedAt.Before(*m.RequestedAt) {
			return ErrInvalidMintedAt
		}
	}

	// Audit validations (align with TS: required created/updated, optional deleted)
	if m.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if strings.TrimSpace(m.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}
	if m.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}
	if strings.TrimSpace(m.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if m.UpdatedAt.Before(m.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if m.DeletedAt != nil && m.DeletedAt.Before(m.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	// If either deletedAt or deletedBy is set, require both
	if (m.DeletedAt == nil) != (m.DeletedBy == nil) {
		if m.DeletedAt == nil {
			return ErrInvalidDeletedAt
		}
		return ErrInvalidDeletedBy
	}
	if m.DeletedBy != nil && strings.TrimSpace(*m.DeletedBy) == "" {
		return ErrInvalidDeletedBy
	}

	return nil
}

// Helpers

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time")
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
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

func normalizeStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil {
		return nil
	}
	if p.IsZero() {
		return nil
	}
	utc := p.UTC()
	return &utc
}

// MintRequestsTableDDL defines the SQL for the mint_requests table migration.
const MintRequestsTableDDL = `
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
`
