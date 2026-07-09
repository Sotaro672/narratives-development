// backend/internal/domain/mint/mint_product_task.go
package mint

import (
	"errors"
	"time"
)

// ------------------------------------------------------
// Entity: MintProductTask
// ------------------------------------------------------
//
// productId 単位で 1 件ずつ on-chain mint を実行するための task entity です。
//
// Firestore 推奨構造:
//
// mints/{mintID}/products/{productID}
//
// - mintId             : string
// - productId          : string
// - status             : string
// - attemptCount       : int
// - mintAddress        : string
// - signature          : string
// - errorMessage       : string
// - createdAt          : time.Time
// - updatedAt          : time.Time
// - mintingStartedAt   : *time.Time
// - mintedAt           : *time.Time
// - lastFailedAt       : *time.Time
//
// NOTE:
// - 親 Mint は全体進捗を管理します。
// - MintProductTask は productId 単位の実行状態を管理します。
// - 1件成功後に次の1件を enqueue / execute する設計の中心になります。
type MintProductTask struct {
	MintID    string `json:"mintId"`
	ProductID string `json:"productId"`

	Status MintProductTaskStatus `json:"status"`

	AttemptCount int `json:"attemptCount"`

	MintAddress string `json:"mintAddress,omitempty"`
	Signature   string `json:"signature,omitempty"`

	ErrorMessage string `json:"errorMessage,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	MintingStartedAt *time.Time `json:"mintingStartedAt,omitempty"`
	MintedAt         *time.Time `json:"mintedAt,omitempty"`
	LastFailedAt     *time.Time `json:"lastFailedAt,omitempty"`
}

// ------------------------------------------------------
// Status
// ------------------------------------------------------

type MintProductTaskStatus string

const (
	// MintProductTaskStatusPending は、まだmint実行されていない状態です。
	MintProductTaskStatusPending MintProductTaskStatus = "PENDING"

	// MintProductTaskStatusMinting は、worker が対象 product を処理中の状態です。
	MintProductTaskStatusMinting MintProductTaskStatus = "MINTING"

	// MintProductTaskStatusMinted は、on-chain mint と保存処理が完了した状態です。
	MintProductTaskStatusMinted MintProductTaskStatus = "MINTED"

	// MintProductTaskStatusFailedRetryable は、RPC 429 / timeout など再実行可能な失敗状態です。
	MintProductTaskStatusFailedRetryable MintProductTaskStatus = "FAILED_RETRYABLE"

	// MintProductTaskStatusFailedFatal は、入力不正など再実行しても成功しない可能性が高い失敗状態です。
	MintProductTaskStatusFailedFatal MintProductTaskStatus = "FAILED_FATAL"
)

func (s MintProductTaskStatus) IsValid() bool {
	switch s {
	case MintProductTaskStatusPending,
		MintProductTaskStatusMinting,
		MintProductTaskStatusMinted,
		MintProductTaskStatusFailedRetryable,
		MintProductTaskStatusFailedFatal:
		return true
	default:
		return false
	}
}

func (s MintProductTaskStatus) IsFinished() bool {
	return s == MintProductTaskStatusMinted ||
		s == MintProductTaskStatusFailedFatal
}

func (s MintProductTaskStatus) IsRetryable() bool {
	return s == MintProductTaskStatusPending ||
		s == MintProductTaskStatusFailedRetryable
}

// ------------------------------------------------------
// Errors
// ------------------------------------------------------

var (
	ErrInvalidMintProductTaskMintID        = errors.New("mint product task: invalid mintId")
	ErrInvalidMintProductTaskProductID     = errors.New("mint product task: invalid productId")
	ErrInvalidMintProductTaskStatus        = errors.New("mint product task: invalid status")
	ErrInvalidMintProductTaskCreatedAt     = errors.New("mint product task: invalid createdAt")
	ErrInvalidMintProductTaskUpdatedAt     = errors.New("mint product task: invalid updatedAt")
	ErrInvalidMintProductTaskAttemptCount  = errors.New("mint product task: invalid attemptCount")
	ErrInvalidMintProductTaskMintedAt      = errors.New("mint product task: invalid mintedAt")
	ErrInvalidMintProductTaskResult        = errors.New("mint product task: invalid mint result")
	ErrInconsistentMintProductTaskStatus   = errors.New("mint product task: inconsistent status")
	ErrMintProductTaskAlreadyMinted        = errors.New("mint product task: already minted")
	ErrMintProductTaskNotExecutable        = errors.New("mint product task: not executable")
	ErrMintProductTaskNotFound             = errors.New("mint product task: not found")
	ErrMintProductTaskConcurrentProcessing = errors.New("mint product task: already processing")
)

// ------------------------------------------------------
// Constructors
// ------------------------------------------------------

func NewMintProductTask(
	mintID string,
	productID string,
	now time.Time,
) (MintProductTask, error) {
	if mintID == "" {
		return MintProductTask{}, ErrInvalidMintProductTaskMintID
	}
	if productID == "" {
		return MintProductTask{}, ErrInvalidMintProductTaskProductID
	}
	if now.IsZero() {
		return MintProductTask{}, ErrInvalidMintProductTaskCreatedAt
	}

	t := now.UTC()

	task := MintProductTask{
		MintID:    mintID,
		ProductID: productID,

		Status: MintProductTaskStatusPending,

		AttemptCount: 0,

		MintAddress:  "",
		Signature:    "",
		ErrorMessage: "",

		CreatedAt: t,
		UpdatedAt: t,

		MintingStartedAt: nil,
		MintedAt:         nil,
		LastFailedAt:     nil,
	}

	if err := task.Validate(); err != nil {
		return MintProductTask{}, err
	}

	return task, nil
}

// ------------------------------------------------------
// Behavior
// ------------------------------------------------------

func (t *MintProductTask) MarkMinting(now time.Time) error {
	if t == nil {
		return ErrInvalidMintProductTaskProductID
	}
	if now.IsZero() {
		return ErrInvalidMintProductTaskUpdatedAt
	}

	if t.Status == MintProductTaskStatusMinted {
		return ErrMintProductTaskAlreadyMinted
	}

	if t.Status == MintProductTaskStatusMinting {
		return ErrMintProductTaskConcurrentProcessing
	}

	if !t.Status.IsRetryable() {
		return ErrMintProductTaskNotExecutable
	}

	utc := now.UTC()

	t.Status = MintProductTaskStatusMinting
	t.AttemptCount++
	t.ErrorMessage = ""
	t.MintingStartedAt = &utc
	t.UpdatedAt = utc

	return t.Validate()
}

func (t *MintProductTask) MarkMinted(
	now time.Time,
	mintAddress string,
	signature string,
) error {
	if t == nil {
		return ErrInvalidMintProductTaskProductID
	}
	if now.IsZero() {
		return ErrInvalidMintProductTaskMintedAt
	}
	if mintAddress == "" || signature == "" {
		return ErrInvalidMintProductTaskResult
	}

	utc := now.UTC()

	t.Status = MintProductTaskStatusMinted
	t.MintAddress = mintAddress
	t.Signature = signature
	t.ErrorMessage = ""
	t.MintedAt = &utc
	t.UpdatedAt = utc

	return t.Validate()
}

func (t *MintProductTask) MarkFailedRetryable(
	now time.Time,
	message string,
) error {
	if t == nil {
		return ErrInvalidMintProductTaskProductID
	}
	if now.IsZero() {
		return ErrInvalidMintProductTaskUpdatedAt
	}

	utc := now.UTC()

	t.Status = MintProductTaskStatusFailedRetryable
	t.ErrorMessage = message
	t.LastFailedAt = &utc
	t.UpdatedAt = utc

	return t.Validate()
}

func (t *MintProductTask) MarkFailedFatal(
	now time.Time,
	message string,
) error {
	if t == nil {
		return ErrInvalidMintProductTaskProductID
	}
	if now.IsZero() {
		return ErrInvalidMintProductTaskUpdatedAt
	}

	utc := now.UTC()

	t.Status = MintProductTaskStatusFailedFatal
	t.ErrorMessage = message
	t.LastFailedAt = &utc
	t.UpdatedAt = utc

	return t.Validate()
}

func (t *MintProductTask) ResetToPending(now time.Time) error {
	if t == nil {
		return ErrInvalidMintProductTaskProductID
	}
	if now.IsZero() {
		return ErrInvalidMintProductTaskUpdatedAt
	}

	if t.Status == MintProductTaskStatusMinted {
		return ErrMintProductTaskAlreadyMinted
	}

	utc := now.UTC()

	t.Status = MintProductTaskStatusPending
	t.ErrorMessage = ""
	t.MintingStartedAt = nil
	t.LastFailedAt = nil
	t.UpdatedAt = utc

	return t.Validate()
}

// ------------------------------------------------------
// Validation
// ------------------------------------------------------

func (t MintProductTask) Validate() error {
	if t.MintID == "" {
		return ErrInvalidMintProductTaskMintID
	}

	if t.ProductID == "" {
		return ErrInvalidMintProductTaskProductID
	}

	if t.Status == "" {
		return ErrInvalidMintProductTaskStatus
	}

	if !t.Status.IsValid() {
		return ErrInvalidMintProductTaskStatus
	}

	if t.AttemptCount < 0 {
		return ErrInvalidMintProductTaskAttemptCount
	}

	if t.CreatedAt.IsZero() {
		return ErrInvalidMintProductTaskCreatedAt
	}

	if t.UpdatedAt.IsZero() {
		return ErrInvalidMintProductTaskUpdatedAt
	}

	if t.Status == MintProductTaskStatusPending {
		if t.MintedAt != nil {
			return ErrInconsistentMintProductTaskStatus
		}
		if t.MintAddress != "" || t.Signature != "" {
			return ErrInconsistentMintProductTaskStatus
		}
	}

	if t.Status == MintProductTaskStatusMinting {
		if t.MintedAt != nil {
			return ErrInconsistentMintProductTaskStatus
		}
		if t.MintAddress != "" || t.Signature != "" {
			return ErrInconsistentMintProductTaskStatus
		}
	}

	if t.Status == MintProductTaskStatusMinted {
		if t.MintedAt == nil {
			return ErrInvalidMintProductTaskMintedAt
		}
		if t.MintAddress == "" || t.Signature == "" {
			return ErrInvalidMintProductTaskResult
		}
	}

	if t.Status == MintProductTaskStatusFailedRetryable ||
		t.Status == MintProductTaskStatusFailedFatal {
		if t.MintedAt != nil {
			return ErrInconsistentMintProductTaskStatus
		}
		if t.MintAddress != "" || t.Signature != "" {
			return ErrInconsistentMintProductTaskStatus
		}
	}

	return nil
}
