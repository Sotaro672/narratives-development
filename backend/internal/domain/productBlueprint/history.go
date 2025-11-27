// backend/internal/domain/productBlueprint/history.go
package productBlueprint

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ============================================================
// Version / History 用エラー
// ============================================================

var (
	ErrInvalidVersionNumber = errors.New("productBlueprint: invalid version number")
	ErrInvalidSnapshot      = errors.New("productBlueprint: invalid snapshot")
)

// ============================================================
// モデル側のスナップショット構造体
//  ※ productBlueprint に従属する models の「フルスナップショット」用。
//  ※ domain/model を直接 import せず、history 用の専用構造体として定義。
// ============================================================

type ModelVariationSnapshot struct {
	ID string `firestore:"id" json:"id"`

	ProductBlueprintID string `firestore:"productBlueprintId" json:"productBlueprintId"`

	ModelNumber string `firestore:"modelNumber" json:"modelNumber"`
	Size        string `firestore:"size" json:"size"`

	ColorName string `firestore:"colorName" json:"colorName"`
	ColorRGB  int    `firestore:"colorRgb" json:"colorRgb"`

	Measurements map[string]int `firestore:"measurements" json:"measurements"`

	CreatedAt *time.Time `firestore:"createdAt,omitempty" json:"createdAt,omitempty"`
	CreatedBy *string    `firestore:"createdBy,omitempty" json:"createdBy,omitempty"`
	UpdatedAt *time.Time `firestore:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	UpdatedBy *string    `firestore:"updatedBy,omitempty" json:"updatedBy,omitempty"`
	DeletedAt *time.Time `firestore:"deletedAt,omitempty" json:"deletedAt,omitempty"`
	DeletedBy *string    `firestore:"deletedBy,omitempty" json:"deletedBy,omitempty"`
	// ※ models 側の論理削除 TTL などを持たせたい場合はここに ExpireAt を追加してもよい
}

// ============================================================
// ProductBlueprint + Models のフルスナップショット
// ============================================================

type ProductBlueprintFullSnapshot struct {
	Blueprint ProductBlueprint         `firestore:"blueprint" json:"blueprint"`
	Models    []ModelVariationSnapshot `firestore:"models" json:"models"`
}

// ============================================================
// 履歴 1 バージョン分のエンティティ
//
// Firestore パス: product_blueprints_history/{blueprintId}/versions/{version}
// ============================================================

type ProductBlueprintHistoryVersion struct {
	// 親 Blueprint の ID（= productBlueprint ドキュメント ID）
	BlueprintID string `firestore:"blueprintId" json:"blueprintId"`

	// バージョン番号（1,2,3,... の単調増加）
	Version int64 `firestore:"version" json:"version"`

	// 当該バージョン時点のフルスナップショット
	Snapshot ProductBlueprintFullSnapshot `firestore:"snapshot" json:"snapshot"`

	// 変更メタ情報
	//   - CreatedAt: そのバージョンが確定した日時
	//   - CreatedBy: 操作者（UID など）
	CreatedAt time.Time `firestore:"createdAt" json:"createdAt"`
	CreatedBy *string   `firestore:"createdBy,omitempty" json:"createdBy,omitempty"`

	// 変更理由や概要（任意）
	ChangeSummary string `firestore:"changeSummary,omitempty" json:"changeSummary,omitempty"`

	// 変更種別（create/update/delete/restore など）も必要なら enum 的に追加
	ChangeType string `firestore:"changeType,omitempty" json:"changeType,omitempty"`
}

// ============================================================
// コンストラクタ / ヘルパ
// ============================================================

// NewHistoryVersion は、現在の ProductBlueprint と models のスナップショットから
// 履歴バージョンを生成する。
func NewHistoryVersion(
	blueprint ProductBlueprint,
	models []ModelVariationSnapshot,
	version int64,
	createdBy *string,
	at time.Time,
	changeType string,
	changeSummary string,
) (ProductBlueprintHistoryVersion, error) {
	if version <= 0 {
		return ProductBlueprintHistoryVersion{}, ErrInvalidVersionNumber
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	bid := strings.TrimSpace(blueprint.ID)
	if bid == "" {
		return ProductBlueprintHistoryVersion{}, fmt.Errorf("%w: empty blueprint id", ErrInvalidSnapshot)
	}

	// models 側の BlueprintID があれば軽く整合性チェック
	for i, m := range models {
		if m.ProductBlueprintID == "" {
			continue
		}
		if strings.TrimSpace(m.ProductBlueprintID) != bid {
			return ProductBlueprintHistoryVersion{}, fmt.Errorf(
				"%w: model[%d].productBlueprintId = %q does not match blueprintId = %q",
				ErrInvalidSnapshot, i, m.ProductBlueprintID, bid,
			)
		}
	}

	snap := ProductBlueprintFullSnapshot{
		Blueprint: blueprint,
		Models:    models,
	}

	h := ProductBlueprintHistoryVersion{
		BlueprintID:   bid,
		Version:       version,
		Snapshot:      snap,
		CreatedAt:     at.UTC(),
		CreatedBy:     createdBy,
		ChangeSummary: strings.TrimSpace(changeSummary),
		ChangeType:    strings.TrimSpace(changeType),
	}

	if err := h.validate(); err != nil {
		return ProductBlueprintHistoryVersion{}, err
	}
	return h, nil
}

// validate は履歴エンティティの整合性チェックを行う。
func (h ProductBlueprintHistoryVersion) validate() error {
	if strings.TrimSpace(h.BlueprintID) == "" {
		return ErrInvalidID
	}
	if h.Version <= 0 {
		return ErrInvalidVersionNumber
	}
	if h.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	// Blueprint の ID が BlueprintID と一致しているか
	if strings.TrimSpace(h.Snapshot.Blueprint.ID) != strings.TrimSpace(h.BlueprintID) {
		return fmt.Errorf(
			"%w: snapshot.blueprint.id=%q, blueprintId=%q",
			ErrInvalidSnapshot,
			h.Snapshot.Blueprint.ID,
			h.BlueprintID,
		)
	}

	// Models 側もあれば ID 一致を軽くチェック
	for i, m := range h.Snapshot.Models {
		if m.ProductBlueprintID == "" {
			continue
		}
		if strings.TrimSpace(m.ProductBlueprintID) != strings.TrimSpace(h.BlueprintID) {
			return fmt.Errorf(
				"%w: snapshot.models[%d].productBlueprintId=%q, blueprintId=%q",
				ErrInvalidSnapshot,
				i,
				m.ProductBlueprintID,
				h.BlueprintID,
			)
		}
	}

	return nil
}

// IsFor は指定された blueprintID の履歴かどうかを判定する。
func (h ProductBlueprintHistoryVersion) IsFor(blueprintID string) bool {
	return strings.TrimSpace(h.BlueprintID) == strings.TrimSpace(blueprintID)
}
