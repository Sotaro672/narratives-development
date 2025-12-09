// backend\internal\domain\campaign\entity.go
package campaign

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// =======================
// Enums (mirror TS)
// =======================

type CampaignStatus string
type AdType string

const (
	// CampaignStatus
	StatusDraft     CampaignStatus = "draft"
	StatusActive    CampaignStatus = "active"
	StatusPaused    CampaignStatus = "paused"
	StatusScheduled CampaignStatus = "scheduled"
	StatusCompleted CampaignStatus = "completed"
	StatusDeleted   CampaignStatus = "deleted"

	// AdType
	AdImageCarousel AdType = "image_carousel"
	AdVideo         AdType = "video"
	AdStory         AdType = "story"
	AdReel          AdType = "reel"
	AdBanner        AdType = "banner"
	AdNative        AdType = "native"
)

func IsValidStatus(s CampaignStatus) bool {
	switch s {
	case StatusDraft, StatusActive, StatusPaused, StatusScheduled, StatusCompleted, StatusDeleted:
		return true
	default:
		return false
	}
}

func IsValidAdType(t AdType) bool {
	switch t {
	case AdImageCarousel, AdVideo, AdStory, AdReel, AdBanner, AdNative:
		return true
	default:
		return false
	}
}

// =======================
// Entity (mirror TS Campaign)
// =======================

type Campaign struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	BrandID        string         `json:"brandId"`
	AssigneeID     string         `json:"assigneeId"`
	ListID         string         `json:"listId"`
	Status         CampaignStatus `json:"status"`
	Budget         float64        `json:"budget"`
	Spent          float64        `json:"spent"`
	StartDate      time.Time      `json:"startDate"`
	EndDate        time.Time      `json:"endDate"`
	TargetAudience string         `json:"targetAudience"`
	AdType         AdType         `json:"adType"`
	Headline       string         `json:"headline"`
	Description    string         `json:"description"`

	PerformanceID *string    `json:"performanceId,omitempty"`
	ImageID       *string    `json:"imageId,omitempty"`
	CreatedBy     *string    `json:"createdBy,omitempty"`
	CreatedAt     time.Time  `json:"createdAt,omitempty"`
	UpdatedBy     *string    `json:"updatedBy,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
	DeletedBy     *string    `json:"deletedBy,omitempty"`
}

// =======================
// Errors
// =======================

var (
	ErrInvalidID             = errors.New("campaign: invalid id")
	ErrInvalidName           = errors.New("campaign: invalid name")
	ErrInvalidBrandID        = errors.New("campaign: invalid brandId")
	ErrInvalidAssigneeID     = errors.New("campaign: invalid assigneeId")
	ErrInvalidListID         = errors.New("campaign: invalid listId")
	ErrInvalidStatus         = errors.New("campaign: invalid status")
	ErrInvalidBudget         = errors.New("campaign: invalid budget")
	ErrInvalidSpent          = errors.New("campaign: invalid spent")
	ErrOverspend             = errors.New("campaign: spent exceeds budget")
	ErrInvalidDates          = errors.New("campaign: invalid dates")
	ErrInvalidTargetAudience = errors.New("campaign: invalid targetAudience")
	ErrInvalidAdType         = errors.New("campaign: invalid adType")
	ErrInvalidHeadline       = errors.New("campaign: invalid headline")
	ErrInvalidDescription    = errors.New("campaign: invalid description")
	ErrInvalidPerformanceID  = errors.New("campaign: invalid performanceId")
	ErrInvalidImageID        = errors.New("campaign: invalid imageId")
	ErrInvalidCreatedBy      = errors.New("campaign: invalid createdBy")
	ErrInvalidCreatedAt      = errors.New("campaign: invalid createdAt")
	ErrInvalidUpdatedBy      = errors.New("campaign: invalid updatedBy")
	ErrInvalidUpdatedAt      = errors.New("campaign: invalid updatedAt")
	ErrInvalidDeletedAt      = errors.New("campaign: invalid deletedAt")
	ErrInvalidDeletedBy      = errors.New("campaign: invalid deletedBy")
)

// =======================
// Policy
// =======================

var (
	MinBudget            float64 = 0
	MaxBudget            float64 = 0 // 0 = no upper bound
	MinSpent             float64 = 0
	MaxSpent             float64 = 0    // 0 = no upper bound
	DisallowOverspend            = true // if true: spent <= budget
	MaxNameLength                = 200
	MaxAudienceLength            = 1000
	MaxHeadlineLength            = 120
	MaxDescriptionLength         = 2000
)

// =======================
// Constructors
// =======================

func New(
	id, name, brandID, assigneeID, listID string,
	status CampaignStatus,
	budget, spent float64,
	startDate, endDate time.Time,
	targetAudience string,
	adType AdType,
	headline, description string,
	performanceID, imageID, createdBy, updatedBy, deletedBy *string,
	createdAt time.Time,
	updatedAt, deletedAt *time.Time,
) (Campaign, error) {
	c := Campaign{
		ID:             strings.TrimSpace(id),
		Name:           strings.TrimSpace(name),
		BrandID:        strings.TrimSpace(brandID),
		AssigneeID:     strings.TrimSpace(assigneeID),
		ListID:         strings.TrimSpace(listID),
		Status:         status,
		Budget:         budget,
		Spent:          spent,
		StartDate:      startDate.UTC(),
		EndDate:        endDate.UTC(),
		TargetAudience: strings.TrimSpace(targetAudience),
		AdType:         adType,
		Headline:       strings.TrimSpace(headline),
		Description:    strings.TrimSpace(description),
		PerformanceID:  normalizePtr(performanceID),
		ImageID:        normalizePtr(imageID),
		CreatedBy:      normalizePtr(createdBy),
		CreatedAt:      createdAt.UTC(),
		UpdatedBy:      normalizePtr(updatedBy),
		UpdatedAt:      normalizeTimePtr(updatedAt),
		DeletedAt:      normalizeTimePtr(deletedAt),
		DeletedBy:      normalizePtr(deletedBy),
	}
	if err := c.validate(); err != nil {
		return Campaign{}, err
	}
	return c, nil
}

func NewWithNow(
	id, name, brandID, assigneeID, listID string,
	status CampaignStatus,
	budget, spent float64,
	startDate, endDate time.Time,
	targetAudience string,
	adType AdType,
	headline, description string,
	performanceID, imageID, createdBy, updatedBy, deletedBy *string,
	now time.Time,
	updatedAt, deletedAt *time.Time,
) (Campaign, error) {
	now = now.UTC()
	return New(
		id, name, brandID, assigneeID, listID,
		status, budget, spent,
		startDate, endDate,
		targetAudience, adType, headline, description,
		performanceID, imageID, createdBy, updatedBy, deletedBy,
		now, updatedAt, deletedAt,
	)
}

func NewFromStringDates(
	id, name, brandID, assigneeID, listID string,
	status CampaignStatus,
	budget, spent float64,
	startDateStr, endDateStr string,
	targetAudience string,
	adType AdType,
	headline, description string,
	performanceID, imageID, createdBy, updatedBy, deletedBy *string,
	createdAtStr string,
	updatedAtStr, deletedAtStr *string,
) (Campaign, error) {
	sd, err := parseTime(startDateStr, ErrInvalidDates)
	if err != nil {
		return Campaign{}, err
	}
	ed, err := parseTime(endDateStr, ErrInvalidDates)
	if err != nil {
		return Campaign{}, err
	}
	ca, err := parseTime(createdAtStr, ErrInvalidCreatedAt)
	if err != nil {
		return Campaign{}, err
	}
	var ua, da *time.Time
	if updatedAtStr != nil && strings.TrimSpace(*updatedAtStr) != "" {
		t, err := parseTime(*updatedAtStr, ErrInvalidUpdatedAt)
		if err != nil {
			return Campaign{}, err
		}
		ua = &t
	}
	if deletedAtStr != nil && strings.TrimSpace(*deletedAtStr) != "" {
		t, err := parseTime(*deletedAtStr, ErrInvalidDeletedAt)
		if err != nil {
			return Campaign{}, err
		}
		da = &t
	}
	return New(
		id, name, brandID, assigneeID, listID,
		status, budget, spent,
		sd, ed,
		targetAudience, adType, headline, description,
		performanceID, imageID, createdBy, updatedBy, deletedBy,
		ca, ua, da,
	)
}

// =======================
// Behavior
// =======================

func (c *Campaign) SetStatus(s CampaignStatus) error {
	if !IsValidStatus(s) {
		return ErrInvalidStatus
	}
	c.Status = s
	return nil
}

func (c *Campaign) UpdateBudgetSpent(budget, spent float64) error {
	if budget < MinBudget || (MaxBudget > 0 && budget > MaxBudget) {
		return ErrInvalidBudget
	}
	if spent < MinSpent || (MaxSpent > 0 && spent > MaxSpent) {
		return ErrInvalidSpent
	}
	if DisallowOverspend && spent > budget {
		return ErrOverspend
	}
	c.Budget = budget
	c.Spent = spent
	return nil
}

func (c *Campaign) AdjustSpend(delta float64) error {
	newSpent := c.Spent + delta
	if newSpent < MinSpent || (MaxSpent > 0 && newSpent > MaxSpent) {
		return ErrInvalidSpent
	}
	if DisallowOverspend && newSpent > c.Budget {
		return ErrOverspend
	}
	c.Spent = newSpent
	return nil
}

func (c *Campaign) SetDates(start, end time.Time) error {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return ErrInvalidDates
	}
	c.StartDate = start.UTC()
	c.EndDate = end.UTC()
	return nil
}

func (c *Campaign) UpdateMeta(name, audience string, adType AdType, headline, description string) error {
	name = strings.TrimSpace(name)
	audience = strings.TrimSpace(audience)
	headline = strings.TrimSpace(headline)
	description = strings.TrimSpace(description)

	if name == "" || (MaxNameLength > 0 && len([]rune(name)) > MaxNameLength) {
		return ErrInvalidName
	}
	if audience == "" || (MaxAudienceLength > 0 && len([]rune(audience)) > MaxAudienceLength) {
		return ErrInvalidTargetAudience
	}
	if !IsValidAdType(adType) {
		return ErrInvalidAdType
	}
	if headline == "" || (MaxHeadlineLength > 0 && len([]rune(headline)) > MaxHeadlineLength) {
		return ErrInvalidHeadline
	}
	if description == "" || (MaxDescriptionLength > 0 && len([]rune(description)) > MaxDescriptionLength) {
		return ErrInvalidDescription
	}

	c.Name = name
	c.TargetAudience = audience
	c.AdType = adType
	c.Headline = headline
	c.Description = description
	return nil
}

func (c *Campaign) Reassign(brandID, assigneeID, listID string) error {
	brandID = strings.TrimSpace(brandID)
	assigneeID = strings.TrimSpace(assigneeID)
	listID = strings.TrimSpace(listID)
	if brandID == "" {
		return ErrInvalidBrandID
	}
	if assigneeID == "" {
		return ErrInvalidAssigneeID
	}
	if listID == "" {
		return ErrInvalidListID
	}
	c.BrandID = brandID
	c.AssigneeID = assigneeID
	c.ListID = listID
	return nil
}

func (c *Campaign) SetOptionalIDs(performanceID, imageID, createdBy, updatedBy, deletedBy *string) error {
	if performanceID != nil && strings.TrimSpace(*performanceID) == "" {
		return ErrInvalidPerformanceID
	}
	if imageID != nil && strings.TrimSpace(*imageID) == "" {
		return ErrInvalidImageID
	}
	if createdBy != nil && strings.TrimSpace(*createdBy) == "" {
		return ErrInvalidCreatedBy
	}
	if updatedBy != nil && strings.TrimSpace(*updatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if deletedBy != nil && strings.TrimSpace(*deletedBy) == "" {
		return ErrInvalidDeletedBy
	}
	c.PerformanceID = normalizePtr(performanceID)
	c.ImageID = normalizePtr(imageID)
	c.CreatedBy = normalizePtr(createdBy)
	c.UpdatedBy = normalizePtr(updatedBy)
	c.DeletedBy = normalizePtr(deletedBy)
	return nil
}

// =======================
// Validation
// =======================

func (c Campaign) validate() error {
	if c.ID == "" {
		return ErrInvalidID
	}
	if c.Name == "" || (MaxNameLength > 0 && len([]rune(c.Name)) > MaxNameLength) {
		return ErrInvalidName
	}
	if c.BrandID == "" {
		return ErrInvalidBrandID
	}
	if strings.TrimSpace(c.AssigneeID) == "" {
		return ErrInvalidAssigneeID
	}
	if strings.TrimSpace(c.ListID) == "" {
		return ErrInvalidListID
	}
	if !IsValidStatus(c.Status) {
		return ErrInvalidStatus
	}
	if c.Budget < MinBudget || (MaxBudget > 0 && c.Budget > MaxBudget) {
		return ErrInvalidBudget
	}
	if c.Spent < MinSpent || (MaxSpent > 0 && c.Spent > MaxSpent) {
		return ErrInvalidSpent
	}
	if DisallowOverspend && c.Spent > c.Budget {
		return ErrOverspend
	}
	if c.StartDate.IsZero() || c.EndDate.IsZero() || c.EndDate.Before(c.StartDate) {
		return ErrInvalidDates
	}
	if c.TargetAudience == "" || (MaxAudienceLength > 0 && len([]rune(c.TargetAudience)) > MaxAudienceLength) {
		return ErrInvalidTargetAudience
	}
	if !IsValidAdType(c.AdType) {
		return ErrInvalidAdType
	}
	if c.Headline == "" || (MaxHeadlineLength > 0 && len([]rune(c.Headline)) > MaxHeadlineLength) {
		return ErrInvalidHeadline
	}
	if c.Description == "" || (MaxDescriptionLength > 0 && len([]rune(c.Description)) > MaxDescriptionLength) {
		return ErrInvalidDescription
	}
	if c.PerformanceID != nil && strings.TrimSpace(*c.PerformanceID) == "" {
		return ErrInvalidPerformanceID
	}
	if c.ImageID != nil && strings.TrimSpace(*c.ImageID) == "" {
		return ErrInvalidImageID
	}
	if c.CreatedBy != nil && strings.TrimSpace(*c.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}
	if c.UpdatedBy != nil && strings.TrimSpace(*c.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if c.DeletedBy != nil && strings.TrimSpace(*c.DeletedBy) == "" {
		return ErrInvalidDeletedBy
	}
	if c.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if c.UpdatedAt != nil && c.UpdatedAt.Before(c.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if c.DeletedAt != nil && c.DeletedAt.Before(c.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	if c.DeletedAt != nil && c.UpdatedAt != nil && c.DeletedAt.Before(*c.UpdatedAt) {
		return ErrInvalidDeletedAt
	}
	return nil
}

// =======================
// Helpers
// =======================

func normalizePtr(p *string) *string {
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
	if p == nil || p.IsZero() {
		return nil
	}
	t := p.UTC()
	return &t
}

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
