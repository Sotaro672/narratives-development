// backend/internal/domain/common/deletion_lifecycle.go
package common

import "time"

// DeletionStatusは、論理削除対象Entityの状態を表す。
type DeletionStatus string

const (
	DeletionStatusActive  DeletionStatus = "active"
	DeletionStatusDeleted DeletionStatus = "deleted"
)

// DeletionRestoreWindowは、論理削除後に復旧できる期間を表す。
//
// deletedAtから30日間は復旧可能とし、
// purgeAt以降は物理削除対象とする。
const DeletionRestoreWindow = 30 * 24 * time.Hour

// DeletionLifecycleは、論理削除・復旧・物理削除判定に必要な
// 共通ライフサイクル情報を保持する。
//
// ProductBlueprintとModelの両方で利用する。
type DeletionLifecycle struct {
	Status DeletionStatus

	DeletedAt *time.Time
	DeletedBy *string
	PurgeAt   *time.Time
}

// NewActiveDeletionLifecycleは、未削除状態のライフサイクルを生成する。
func NewActiveDeletionLifecycle() DeletionLifecycle {
	return DeletionLifecycle{
		Status:    DeletionStatusActive,
		DeletedAt: nil,
		DeletedBy: nil,
		PurgeAt:   nil,
	}
}

// NewDeletedDeletionLifecycleは、論理削除済み状態の
// ライフサイクルを生成する。
func NewDeletedDeletionLifecycle(
	deletedAt time.Time,
	deletedBy *string,
) DeletionLifecycle {
	deletedAtUTC := deletedAt.UTC()
	purgeAt := deletedAtUTC.Add(DeletionRestoreWindow)

	return DeletionLifecycle{
		Status:    DeletionStatusDeleted,
		DeletedAt: &deletedAtUTC,
		DeletedBy: cloneStringPointer(deletedBy),
		PurgeAt:   &purgeAt,
	}
}

// IsValidDeletionStatusは、削除状態が有効な値か判定する。
func IsValidDeletionStatus(status DeletionStatus) bool {
	switch status {
	case DeletionStatusActive,
		DeletionStatusDeleted:
		return true

	default:
		return false
	}
}

// NormalizeDeletionStatusは、既存Documentとの互換のため、
// 空のstatusをactiveとして扱う。
//
// Firestore既存Documentのstatusバックフィル完了後も、
// 読み取り互換のためこの処理は残してよい。
func NormalizeDeletionStatus(
	status DeletionStatus,
) DeletionStatus {
	if status == "" {
		return DeletionStatusActive
	}

	return status
}

// IsActiveは、Entityが通常利用可能な状態か判定する。
func (
	lifecycle DeletionLifecycle,
) IsActive() bool {
	return NormalizeDeletionStatus(
		lifecycle.Status,
	) == DeletionStatusActive
}

// IsDeletedは、Entityが論理削除済みか判定する。
func (
	lifecycle DeletionLifecycle,
) IsDeleted() bool {
	return NormalizeDeletionStatus(
		lifecycle.Status,
	) == DeletionStatusDeleted
}

// MarkDeletedは、Entityを論理削除状態へ遷移させる。
//
// 同じEntityへ複数回実行された場合は、最初のdeletedAtとpurgeAtを維持する。
func (
	lifecycle *DeletionLifecycle,
) MarkDeleted(
	deletedAt time.Time,
	deletedBy *string,
) {
	if lifecycle == nil {
		return
	}

	if lifecycle.IsDeleted() {
		return
	}

	deletedAtUTC := deletedAt.UTC()
	purgeAt := deletedAtUTC.Add(
		DeletionRestoreWindow,
	)

	lifecycle.Status =
		DeletionStatusDeleted

	lifecycle.DeletedAt =
		&deletedAtUTC

	lifecycle.DeletedBy =
		cloneStringPointer(
			deletedBy,
		)

	lifecycle.PurgeAt =
		&purgeAt
}

// Restoreは、論理削除状態を未削除状態へ戻す。
//
// 復旧期限の判定はCanRestoreで行った後に呼び出す。
func (
	lifecycle *DeletionLifecycle,
) Restore() {
	if lifecycle == nil {
		return
	}

	lifecycle.Status =
		DeletionStatusActive

	lifecycle.DeletedAt = nil
	lifecycle.DeletedBy = nil
	lifecycle.PurgeAt = nil
}

// CanRestoreは、指定時刻時点で復旧可能か判定する。
//
// now < purgeAtの場合のみ復旧可能。
// now == purgeAtの場合は復旧不可とする。
func (
	lifecycle DeletionLifecycle,
) CanRestore(
	now time.Time,
) bool {
	if !lifecycle.IsDeleted() {
		return false
	}

	if lifecycle.DeletedAt == nil ||
		lifecycle.PurgeAt == nil {
		return false
	}

	return now.UTC().Before(
		lifecycle.PurgeAt.UTC(),
	)
}

// IsPurgeEligibleは、指定時刻時点で物理削除対象か判定する。
//
// now >= purgeAtの場合に物理削除対象とする。
func (
	lifecycle DeletionLifecycle,
) IsPurgeEligible(
	now time.Time,
) bool {
	if !lifecycle.IsDeleted() {
		return false
	}

	if lifecycle.DeletedAt == nil ||
		lifecycle.PurgeAt == nil {
		return false
	}

	return !now.UTC().Before(
		lifecycle.PurgeAt.UTC(),
	)
}

// HasConsistentDeletionStateは、削除状態と削除関連項目の
// 組み合わせが整合しているか判定する。
func (
	lifecycle DeletionLifecycle,
) HasConsistentDeletionState() bool {
	switch NormalizeDeletionStatus(
		lifecycle.Status,
	) {
	case DeletionStatusActive:
		return lifecycle.DeletedAt == nil &&
			lifecycle.DeletedBy == nil &&
			lifecycle.PurgeAt == nil

	case DeletionStatusDeleted:
		if lifecycle.DeletedAt == nil ||
			lifecycle.PurgeAt == nil {
			return false
		}

		expectedPurgeAt :=
			lifecycle.DeletedAt.
				UTC().
				Add(
					DeletionRestoreWindow,
				)

		return lifecycle.PurgeAt.
			UTC().
			Equal(
				expectedPurgeAt,
			)

	default:
		return false
	}
}

// Cloneは、ポインタ値を共有しないDeletionLifecycleを返す。
func (
	lifecycle DeletionLifecycle,
) Clone() DeletionLifecycle {
	return DeletionLifecycle{
		Status: NormalizeDeletionStatus(
			lifecycle.Status,
		),

		DeletedAt: cloneTimePointer(
			lifecycle.DeletedAt,
		),

		DeletedBy: cloneStringPointer(
			lifecycle.DeletedBy,
		),

		PurgeAt: cloneTimePointer(
			lifecycle.PurgeAt,
		),
	}
}

func cloneStringPointer(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	cloned := *value

	return &cloned
}

func cloneTimePointer(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}

	cloned := value.UTC()

	return &cloned
}
