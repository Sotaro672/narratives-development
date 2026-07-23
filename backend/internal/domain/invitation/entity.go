// backend/internal/domain/invitation/entity.go
package invitation

import (
	"context"
	"errors"
	"strings"
	"time"
)

const DefaultInvitationDeliveryMaxAttempts = 5

type DeliveryStatus string

const (
	InvitationDeliveryStatusPending         DeliveryStatus = "pending"
	InvitationDeliveryStatusProcessing      DeliveryStatus = "processing"
	InvitationDeliveryStatusDelivered       DeliveryStatus = "delivered"
	InvitationDeliveryStatusRetryableFailed DeliveryStatus = "retryable_failed"
	InvitationDeliveryStatusFailed          DeliveryStatus = "failed"
)

var (
	ErrInvitationTokenNotFound               = errors.New("invitation: token not found")
	ErrInvitationTokenExpired                = errors.New("invitation: token expired")
	ErrInvitationTokenUsed                   = errors.New("invitation: token already used")
	ErrInvitationTokenRevoked                = errors.New("invitation: token revoked")
	ErrInvitationTokenNotDelivered           = errors.New("invitation: token not delivered")
	ErrInvitationTokenRequired               = errors.New("invitation: token is required")
	ErrInvitationMemberIDRequired            = errors.New("invitation: memberID is required")
	ErrInvitationMemberNotFound              = errors.New("invitation: member not found")
	ErrInvitationCompanyIDRequired           = errors.New("invitation: companyID is required")
	ErrInvitationCompanyMismatch             = errors.New("invitation: company mismatch")
	ErrInvitationUIDRequired                 = errors.New("invitation: UID is required")
	ErrInvitationUIDAlreadyInUse             = errors.New("invitation: UID already in use")
	ErrInvitationNameFieldsRequired          = errors.New("invitation: name fields are required")
	ErrInvitationEmailRequired               = errors.New("invitation: email is required")
	ErrInvitationEmailMismatch               = errors.New("invitation: email mismatch")
	ErrInvitationDeliveryIDRequired          = errors.New("invitation: deliveryID is required")
	ErrInvitationDeliveryNotFound            = errors.New("invitation: delivery not found")
	ErrInvitationDeliveryStatusInvalid       = errors.New("invitation: delivery status is invalid")
	ErrInvitationDeliveryNotClaimable        = errors.New("invitation: delivery is not claimable")
	ErrInvitationDeliveryAttemptCountInvalid = errors.New("invitation: delivery attempt count is invalid")
	ErrInvitationDeliveryMaxAttemptsInvalid  = errors.New("invitation: delivery max attempts is invalid")
	ErrInvitationDeliveryAttemptLimit        = errors.New("invitation: delivery attempt limit reached")
	ErrInvitationDeliveryLeaseInvalid        = errors.New("invitation: delivery lease is invalid")
	ErrInvitationDeliveryErrorRequired       = errors.New("invitation: delivery error is required")
	ErrInvitationDeliveryNextAttemptInvalid  = errors.New("invitation: delivery next attempt is invalid")
)

type InvitationToken struct {
	Token            string
	DeliveryID       string
	MemberID         string
	CompanyID        string
	AssignedBrandIDs []string
	Permissions      []string
	Email            string
	CreatedAt        time.Time
	ExpiresAt        *time.Time
	DeliveredAt      *time.Time
	UsedAt           *time.Time
	RevokedAt        *time.Time
	UpdatedAt        *time.Time
}

type InvitationInfo struct {
	MemberID         string
	CompanyID        string
	AssignedBrandIDs []string
	Permissions      []string
	Email            string
}

type InvitationCompletion struct {
	Token         string
	UID           string
	LastName      string
	LastNameKana  string
	FirstName     string
	FirstNameKana string
	Email         string
}

type InvitationDelivery struct {
	ID                  string
	Token               string
	MemberID            string
	CompanyID           string
	AssignedBrandIDs    []string
	Permissions         []string
	Email               string
	Status              DeliveryStatus
	AttemptCount        int
	MaxAttempts         int
	LastError           string
	ProviderMessageID   string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	NextAttemptAt       *time.Time
	ProcessingStartedAt *time.Time
	ProcessingUntil     *time.Time
	DeliveredAt         *time.Time
	FailedAt            *time.Time
}

func NewInvitationInfo(
	memberID string,
	companyID string,
	assignedBrandIDs []string,
	permissions []string,
	email string,
) (InvitationInfo, error) {
	info := InvitationInfo{
		MemberID:         strings.TrimSpace(memberID),
		CompanyID:        strings.TrimSpace(companyID),
		AssignedBrandIDs: normalizeStringValues(assignedBrandIDs),
		Permissions:      normalizeStringValues(permissions),
		Email:            normalizeEmail(email),
	}

	if info.MemberID == "" {
		return InvitationInfo{}, ErrInvitationMemberIDRequired
	}

	if info.CompanyID == "" {
		return InvitationInfo{}, ErrInvitationCompanyIDRequired
	}

	if info.Email == "" {
		return InvitationInfo{}, ErrInvitationEmailRequired
	}

	return info, nil
}

func (i InvitationInfo) Normalize() (InvitationInfo, error) {
	return NewInvitationInfo(
		i.MemberID,
		i.CompanyID,
		i.AssignedBrandIDs,
		i.Permissions,
		i.Email,
	)
}

func NewInvitationToken(
	token string,
	deliveryID string,
	info InvitationInfo,
	createdAt time.Time,
	expiresAt *time.Time,
) (InvitationToken, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return InvitationToken{}, ErrInvitationTokenRequired
	}

	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return InvitationToken{}, ErrInvitationDeliveryIDRequired
	}

	normalizedInfo, err := info.Normalize()
	if err != nil {
		return InvitationToken{}, err
	}

	createdAt = normalizeTime(createdAt)
	updatedAt := createdAt

	return InvitationToken{
		Token:            token,
		DeliveryID:       deliveryID,
		MemberID:         normalizedInfo.MemberID,
		CompanyID:        normalizedInfo.CompanyID,
		AssignedBrandIDs: append([]string(nil), normalizedInfo.AssignedBrandIDs...),
		Permissions:      append([]string(nil), normalizedInfo.Permissions...),
		Email:            normalizedInfo.Email,
		CreatedAt:        createdAt,
		ExpiresAt:        normalizeTimePointer(expiresAt),
		UpdatedAt:        &updatedAt,
	}, nil
}

func (t InvitationToken) InvitationInfo() (InvitationInfo, error) {
	return NewInvitationInfo(
		t.MemberID,
		t.CompanyID,
		t.AssignedBrandIDs,
		t.Permissions,
		t.Email,
	)
}

func (t InvitationToken) IsExpired(now time.Time) bool {
	if t.ExpiresAt == nil || t.ExpiresAt.IsZero() {
		return false
	}

	return !normalizeTime(now).Before(t.ExpiresAt.UTC())
}

func (t InvitationToken) IsDelivered() bool {
	return t.DeliveredAt != nil && !t.DeliveredAt.IsZero()
}

func (t InvitationToken) IsUsed() bool {
	return t.UsedAt != nil && !t.UsedAt.IsZero()
}

func (t InvitationToken) IsRevoked() bool {
	return t.RevokedAt != nil && !t.RevokedAt.IsZero()
}

func (t InvitationToken) ValidateUsable(now time.Time) error {
	if strings.TrimSpace(t.Token) == "" {
		return ErrInvitationTokenNotFound
	}

	if t.IsRevoked() {
		return ErrInvitationTokenRevoked
	}

	if t.IsUsed() {
		return ErrInvitationTokenUsed
	}

	if t.IsExpired(now) {
		return ErrInvitationTokenExpired
	}

	if !t.IsDelivered() {
		return ErrInvitationTokenNotDelivered
	}

	return nil
}

func NewInvitationCompletion(
	token string,
	uid string,
	lastName string,
	lastNameKana string,
	firstName string,
	firstNameKana string,
	email string,
) (InvitationCompletion, error) {
	completion := InvitationCompletion{
		Token:         strings.TrimSpace(token),
		UID:           strings.TrimSpace(uid),
		LastName:      strings.TrimSpace(lastName),
		LastNameKana:  strings.TrimSpace(lastNameKana),
		FirstName:     strings.TrimSpace(firstName),
		FirstNameKana: strings.TrimSpace(firstNameKana),
		Email:         normalizeEmail(email),
	}

	if completion.Token == "" {
		return InvitationCompletion{}, ErrInvitationTokenRequired
	}

	if completion.UID == "" {
		return InvitationCompletion{}, ErrInvitationUIDRequired
	}

	if completion.LastName == "" ||
		completion.LastNameKana == "" ||
		completion.FirstName == "" ||
		completion.FirstNameKana == "" {
		return InvitationCompletion{}, ErrInvitationNameFieldsRequired
	}

	if completion.Email == "" {
		return InvitationCompletion{}, ErrInvitationEmailRequired
	}

	return completion, nil
}

func (c InvitationCompletion) Normalize() (InvitationCompletion, error) {
	return NewInvitationCompletion(
		c.Token,
		c.UID,
		c.LastName,
		c.LastNameKana,
		c.FirstName,
		c.FirstNameKana,
		c.Email,
	)
}

func NewInvitationDelivery(
	deliveryID string,
	token string,
	info InvitationInfo,
	createdAt time.Time,
	maxAttempts int,
) (InvitationDelivery, error) {
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return InvitationDelivery{}, ErrInvitationDeliveryIDRequired
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return InvitationDelivery{}, ErrInvitationTokenRequired
	}

	normalizedInfo, err := info.Normalize()
	if err != nil {
		return InvitationDelivery{}, err
	}

	if maxAttempts == 0 {
		maxAttempts = DefaultInvitationDeliveryMaxAttempts
	}

	if maxAttempts < 1 {
		return InvitationDelivery{}, ErrInvitationDeliveryMaxAttemptsInvalid
	}

	createdAt = normalizeTime(createdAt)

	return InvitationDelivery{
		ID:               deliveryID,
		Token:            token,
		MemberID:         normalizedInfo.MemberID,
		CompanyID:        normalizedInfo.CompanyID,
		AssignedBrandIDs: append([]string(nil), normalizedInfo.AssignedBrandIDs...),
		Permissions:      append([]string(nil), normalizedInfo.Permissions...),
		Email:            normalizedInfo.Email,
		Status:           InvitationDeliveryStatusPending,
		MaxAttempts:      maxAttempts,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
		NextAttemptAt:    timePointer(createdAt),
	}, nil
}

func (d InvitationDelivery) Normalize() (InvitationDelivery, error) {
	d.ID = strings.TrimSpace(d.ID)
	if d.ID == "" {
		return InvitationDelivery{}, ErrInvitationDeliveryIDRequired
	}

	d.Token = strings.TrimSpace(d.Token)
	if d.Token == "" {
		return InvitationDelivery{}, ErrInvitationTokenRequired
	}

	info, err := NewInvitationInfo(
		d.MemberID,
		d.CompanyID,
		d.AssignedBrandIDs,
		d.Permissions,
		d.Email,
	)
	if err != nil {
		return InvitationDelivery{}, err
	}

	d.MemberID = info.MemberID
	d.CompanyID = info.CompanyID
	d.AssignedBrandIDs = append([]string(nil), info.AssignedBrandIDs...)
	d.Permissions = append([]string(nil), info.Permissions...)
	d.Email = info.Email

	d.Status = DeliveryStatus(strings.TrimSpace(string(d.Status)))
	if d.Status == "" {
		d.Status = InvitationDeliveryStatusPending
	}

	if !d.Status.IsValid() {
		return InvitationDelivery{}, ErrInvitationDeliveryStatusInvalid
	}

	if d.AttemptCount < 0 {
		return InvitationDelivery{}, ErrInvitationDeliveryAttemptCountInvalid
	}

	if d.MaxAttempts == 0 {
		d.MaxAttempts = DefaultInvitationDeliveryMaxAttempts
	}

	if d.MaxAttempts < 1 || d.AttemptCount > d.MaxAttempts {
		return InvitationDelivery{}, ErrInvitationDeliveryMaxAttemptsInvalid
	}

	d.LastError = strings.TrimSpace(d.LastError)
	d.ProviderMessageID = strings.TrimSpace(d.ProviderMessageID)
	d.CreatedAt = normalizeTime(d.CreatedAt)

	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = d.CreatedAt
	} else {
		d.UpdatedAt = d.UpdatedAt.UTC()
	}

	d.NextAttemptAt = normalizeTimePointer(d.NextAttemptAt)
	d.ProcessingStartedAt = normalizeTimePointer(d.ProcessingStartedAt)
	d.ProcessingUntil = normalizeTimePointer(d.ProcessingUntil)
	d.DeliveredAt = normalizeTimePointer(d.DeliveredAt)
	d.FailedAt = normalizeTimePointer(d.FailedAt)

	switch d.Status {
	case InvitationDeliveryStatusProcessing:
		if d.ProcessingUntil == nil {
			return InvitationDelivery{}, ErrInvitationDeliveryLeaseInvalid
		}

	case InvitationDeliveryStatusDelivered:
		if d.DeliveredAt == nil {
			return InvitationDelivery{}, ErrInvitationDeliveryStatusInvalid
		}

	case InvitationDeliveryStatusRetryableFailed:
		if d.LastError == "" {
			return InvitationDelivery{}, ErrInvitationDeliveryErrorRequired
		}

		if d.NextAttemptAt == nil {
			return InvitationDelivery{}, ErrInvitationDeliveryNextAttemptInvalid
		}

	case InvitationDeliveryStatusFailed:
		if d.LastError == "" {
			return InvitationDelivery{}, ErrInvitationDeliveryErrorRequired
		}

		if d.FailedAt == nil {
			return InvitationDelivery{}, ErrInvitationDeliveryStatusInvalid
		}
	}

	return d, nil
}

func (d InvitationDelivery) InvitationInfo() (InvitationInfo, error) {
	return NewInvitationInfo(
		d.MemberID,
		d.CompanyID,
		d.AssignedBrandIDs,
		d.Permissions,
		d.Email,
	)
}

func (s DeliveryStatus) IsValid() bool {
	switch s {
	case InvitationDeliveryStatusPending,
		InvitationDeliveryStatusProcessing,
		InvitationDeliveryStatusDelivered,
		InvitationDeliveryStatusRetryableFailed,
		InvitationDeliveryStatusFailed:
		return true

	default:
		return false
	}
}

func (d InvitationDelivery) IsTerminal() bool {
	return d.Status == InvitationDeliveryStatusDelivered ||
		d.Status == InvitationDeliveryStatusFailed
}

func (d InvitationDelivery) IsDue(now time.Time) bool {
	now = normalizeTime(now)

	switch d.Status {
	case InvitationDeliveryStatusPending,
		InvitationDeliveryStatusRetryableFailed:
		return d.NextAttemptAt == nil ||
			!now.Before(d.NextAttemptAt.UTC())

	case InvitationDeliveryStatusProcessing:
		return d.ProcessingUntil != nil &&
			!now.Before(d.ProcessingUntil.UTC())

	default:
		return false
	}
}

func (d InvitationDelivery) CanClaim(now time.Time) bool {
	return !d.IsTerminal() &&
		d.AttemptCount < d.MaxAttempts &&
		d.IsDue(now)
}

func (d InvitationDelivery) Claim(
	now time.Time,
	processingUntil time.Time,
) (InvitationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return InvitationDelivery{}, err
	}

	now = normalizeTime(now)
	processingUntil = processingUntil.UTC()

	if !processingUntil.After(now) {
		return InvitationDelivery{}, ErrInvitationDeliveryLeaseInvalid
	}

	if normalized.AttemptCount >= normalized.MaxAttempts {
		return InvitationDelivery{}, ErrInvitationDeliveryAttemptLimit
	}

	if !normalized.CanClaim(now) {
		return InvitationDelivery{}, ErrInvitationDeliveryNotClaimable
	}

	normalized.Status = InvitationDeliveryStatusProcessing
	normalized.AttemptCount++
	normalized.LastError = ""
	normalized.NextAttemptAt = nil
	normalized.ProcessingStartedAt = timePointer(now)
	normalized.ProcessingUntil = timePointer(processingUntil)
	normalized.UpdatedAt = now

	return normalized, nil
}

func (d InvitationDelivery) MarkDelivered(
	providerMessageID string,
	deliveredAt time.Time,
) (InvitationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return InvitationDelivery{}, err
	}

	if normalized.Status == InvitationDeliveryStatusDelivered {
		return normalized, nil
	}

	if normalized.Status != InvitationDeliveryStatusProcessing {
		return InvitationDelivery{}, ErrInvitationDeliveryNotClaimable
	}

	deliveredAt = normalizeTime(deliveredAt)

	normalized.Status = InvitationDeliveryStatusDelivered
	normalized.ProviderMessageID = strings.TrimSpace(providerMessageID)
	normalized.LastError = ""
	normalized.NextAttemptAt = nil
	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil
	normalized.DeliveredAt = timePointer(deliveredAt)
	normalized.FailedAt = nil
	normalized.UpdatedAt = deliveredAt

	return normalized, nil
}

func (d InvitationDelivery) MarkRetryableFailed(
	lastError string,
	nextAttemptAt time.Time,
	failedAt time.Time,
) (InvitationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return InvitationDelivery{}, err
	}

	if normalized.Status != InvitationDeliveryStatusProcessing {
		return InvitationDelivery{}, ErrInvitationDeliveryNotClaimable
	}

	if normalized.AttemptCount >= normalized.MaxAttempts {
		return InvitationDelivery{}, ErrInvitationDeliveryAttemptLimit
	}

	lastError = strings.TrimSpace(lastError)
	if lastError == "" {
		return InvitationDelivery{}, ErrInvitationDeliveryErrorRequired
	}

	failedAt = normalizeTime(failedAt)
	nextAttemptAt = nextAttemptAt.UTC()

	if !nextAttemptAt.After(failedAt) {
		return InvitationDelivery{}, ErrInvitationDeliveryNextAttemptInvalid
	}

	normalized.Status = InvitationDeliveryStatusRetryableFailed
	normalized.LastError = lastError
	normalized.NextAttemptAt = timePointer(nextAttemptAt)
	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil
	normalized.UpdatedAt = failedAt

	return normalized, nil
}

func (d InvitationDelivery) MarkFailed(
	lastError string,
	failedAt time.Time,
) (InvitationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return InvitationDelivery{}, err
	}

	if normalized.Status == InvitationDeliveryStatusDelivered {
		return InvitationDelivery{}, ErrInvitationDeliveryStatusInvalid
	}

	if normalized.Status == InvitationDeliveryStatusFailed {
		return normalized, nil
	}

	lastError = strings.TrimSpace(lastError)
	if lastError == "" {
		return InvitationDelivery{}, ErrInvitationDeliveryErrorRequired
	}

	failedAt = normalizeTime(failedAt)

	normalized.Status = InvitationDeliveryStatusFailed
	normalized.LastError = lastError
	normalized.NextAttemptAt = nil
	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil
	normalized.FailedAt = timePointer(failedAt)
	normalized.UpdatedAt = failedAt

	return normalized, nil
}

// Repositoryは、利用者によるtoken検証と招待完了処理を提供します。
type Repository interface {
	ResolveInvitationInfoByToken(
		ctx context.Context,
		token string,
	) (InvitationInfo, error)

	CompleteInvitation(
		ctx context.Context,
		completion InvitationCompletion,
	) error
}

// DeliveryRepositoryは、招待tokenとdelivery outboxを管理します。
//
// CreateOrReuseInvitationDeliveryは、同一Memberに対する未使用かつ
// 未失効のtokenを複数作成してはいけません。
//
// 同一Memberにpending、processing、retryable_failedのdeliveryが
// 存在する場合は、同じdeliveryとtokenを再利用します。
//
// MarkInvitationDeliveryDeliveredは、deliveryのdelivered化と
// invitationToken.deliveredAt更新を同一transactionで実行します。
//
// MarkInvitationDeliveryFailedは、deliveryのfailed化と
// invitationToken.revokedAt更新を同一transactionで実行します。
type DeliveryRepository interface {
	CreateOrReuseInvitationDelivery(
		ctx context.Context,
		info InvitationInfo,
	) (InvitationDelivery, error)

	ListDueInvitationDeliveries(
		ctx context.Context,
		now time.Time,
		limit int,
	) ([]InvitationDelivery, error)

	ClaimInvitationDelivery(
		ctx context.Context,
		deliveryID string,
		now time.Time,
		processingUntil time.Time,
	) (InvitationDelivery, error)

	MarkInvitationDeliveryDelivered(
		ctx context.Context,
		deliveryID string,
		expectedAttemptCount int,
		providerMessageID string,
		deliveredAt time.Time,
	) error

	MarkInvitationDeliveryRetryableFailed(
		ctx context.Context,
		deliveryID string,
		expectedAttemptCount int,
		lastError string,
		nextAttemptAt time.Time,
		failedAt time.Time,
	) error

	MarkInvitationDeliveryFailed(
		ctx context.Context,
		deliveryID string,
		expectedAttemptCount int,
		lastError string,
		failedAt time.Time,
	) error
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeStringValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}

	return value.UTC()
}

func normalizeTimePointer(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}

func timePointer(value time.Time) *time.Time {
	normalized := value.UTC()
	return &normalized
}
