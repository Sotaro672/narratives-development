// backend/internal/domain/model/common.go
package model

import (
	"errors"
	"time"

	commondom "narratives/internal/domain/common"
)

var (
	// ErrInvalidDeletionStateは、statusと削除関連項目の
	// 組み合わせが不整合であることを表します。
	ErrInvalidDeletionState = errors.New(
		"model: invalid deletion state",
	)

	// ErrRestorePeriodExpiredは、論理削除後の
	// 復旧可能期間を経過していることを表します。
	ErrRestorePeriodExpired = errors.New(
		"model: restore period expired",
	)
)

// ModelVariationはModel variationが共通して提供する
// 読み取り・検証用の契約です。
type ModelVariation interface {
	GetID() string

	GetProductBlueprintID() string

	GetKind() ModelVariationKind

	GetModelNumber() string

	GetDeletionLifecycle() commondom.DeletionLifecycle

	// CanModifyはModel自身のLifecycleだけを判定します。
	//
	// ProductBlueprintのprinted状態は、
	// UsecaseまたはRepository側で別途確認します。
	CanModify() bool

	CanRestore(
		now time.Time,
	) bool

	IsPurgeEligible(
		now time.Time,
	) bool

	Validate() error
}

// MutableModelVariationは論理削除と復旧を行える
// Model variationの契約です。
//
// SoftDeleteとRestoreはEntityを変更するため、
// concrete typeのポインタだけがこのInterfaceを満たします。
type MutableModelVariation interface {
	ModelVariation

	SoftDelete(
		now time.Time,
		deletedBy *string,
	) error

	Restore(
		now time.Time,
		restoredBy *string,
	) error
}

func validateModelDeletionLifecycle(
	lifecycle commondom.DeletionLifecycle,
) error {
	if !lifecycle.
		HasConsistentDeletionState() {
		return ErrInvalidDeletionState
	}

	return nil
}

func canModifyModelDeletionLifecycle(
	lifecycle commondom.DeletionLifecycle,
) bool {
	return lifecycle.IsActive()
}

func softDeleteModelDeletionLifecycle(
	lifecycle *commondom.DeletionLifecycle,
	updatedAt *time.Time,
	updatedBy **string,
	now time.Time,
	deletedBy *string,
) error {
	if lifecycle == nil ||
		updatedAt == nil ||
		updatedBy == nil {
		return ErrInvalid
	}

	if err :=
		validateModelDeletionLifecycle(
			*lifecycle,
		); err != nil {
		return err
	}

	if lifecycle.IsDeleted() {
		return nil
	}

	now = normalizeModelLifecycleTime(
		now,
	)

	lifecycle.MarkDeleted(
		now,
		deletedBy,
	)

	*updatedAt = now

	*updatedBy =
		cloneModelLifecycleStringPointer(
			deletedBy,
		)

	return nil
}

func restoreModelDeletionLifecycle(
	lifecycle *commondom.DeletionLifecycle,
	updatedAt *time.Time,
	updatedBy **string,
	now time.Time,
	restoredBy *string,
) error {
	if lifecycle == nil ||
		updatedAt == nil ||
		updatedBy == nil {
		return ErrInvalid
	}

	if err :=
		validateModelDeletionLifecycle(
			*lifecycle,
		); err != nil {
		return err
	}

	if lifecycle.IsActive() {
		return nil
	}

	now = normalizeModelLifecycleTime(
		now,
	)

	if !lifecycle.CanRestore(now) {
		return ErrRestorePeriodExpired
	}

	lifecycle.Restore()

	*updatedAt = now

	*updatedBy =
		cloneModelLifecycleStringPointer(
			restoredBy,
		)

	return nil
}

func normalizeModelLifecycleTime(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}

	return value.UTC()
}

func cloneModelLifecycleStringPointer(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	cloned := *value

	return &cloned
}
