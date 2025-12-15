// backend/internal/application/usecase/inventory_usecase.go
package usecase

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	invdom "narratives/internal/domain/inventory"
)

// ============================================================
// InventoryUsecase
// ============================================================
//
// Inventory (mints) を扱うアプリケーション層ユースケース。
// - RepositoryPort を介して永続化層に依存する（Hexagonal）
// - 入力値の最低限バリデーションを行い、ドメイン整合性は domain.NewMint / validate に委譲する
type InventoryUsecase struct {
	repo invdom.RepositoryPort
}

func NewInventoryUsecase(repo invdom.RepositoryPort) *InventoryUsecase {
	return &InventoryUsecase{repo: repo}
}

// ============================================================
// Commands
// ============================================================

// UpsertFromMint
//
// mint から渡された情報（tokenBlueprintId + productBlueprintId + productIds）をもとに、
// 既に同一組み合わせの Inventory(Mint) が存在する場合は
//   - products に未登録 productId を追加
//   - 追加された件数ぶん accumulation を増加（idempotent）
//
// 存在しない場合は
//   - レコード自体を新規作成（accumulation は productIds 件数で初期化）
//
// ※ docID 生成規則は MintUsecase 側（buildInventoryDocID）と合わせる。
func (uc *InventoryUsecase) UpsertFromMint(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	productIDs []string,
) (invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return invdom.Mint{}, errors.New("inventory usecase/repo is nil")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	pbID := strings.TrimSpace(productBlueprintID)
	if tbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidTokenBlueprintID
	}
	if pbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidProductBlueprintID
	}

	ids := normalizeIDs(productIDs)
	if len(ids) == 0 {
		return invdom.Mint{}, invdom.ErrInvalidProducts
	}

	docID := buildInventoryDocID(tbID, pbID)

	// 1) まず「同一組み合わせ」が既にあるか確認（docID で取得）
	existing, err := uc.repo.GetByID(ctx, docID)
	if err != nil {
		// 無ければ新規作成
		if errors.Is(err, invdom.ErrNotFound) {
			now := time.Now().UTC()
			ent, err := invdom.NewMint(
				docID, // docId を固定（tokenBlueprint + productBlueprint の組み合わせ）
				tbID,
				pbID,
				ids,
				len(ids), // accumulation 初期値 = products 件数
				now,
			)
			if err != nil {
				return invdom.Mint{}, err
			}
			return uc.repo.Create(ctx, ent)
		}
		return invdom.Mint{}, err
	}

	// 2) 既存あり → products に未登録分だけ追加し、その差分だけ accumulation を増やす
	if existing.Products == nil {
		existing.Products = map[string]string{}
	}

	added := 0
	for _, pid := range ids {
		if _, ok := existing.Products[pid]; ok {
			continue
		}
		existing.Products[pid] = "" // mintAddress は後から埋める想定
		added++
	}

	// products 更新
	updated, err := uc.repo.Update(ctx, existing)
	if err != nil {
		return invdom.Mint{}, err
	}

	// accumulation 増加（added が 0 なら no-op）
	if added > 0 {
		updated, err = uc.repo.IncrementAccumulation(ctx, docID, added)
		if err != nil {
			return invdom.Mint{}, err
		}
	}

	return updated, nil
}

// CreateMint creates a new inventory Mint record.
//
// NOTE:
// - createdAt / updatedAt は createdAt を基準に初期化します。
// - products は productIDs から map へ変換され、mintAddress は "" 初期化です。
func (uc *InventoryUsecase) CreateMint(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	productIDs []string,
	accumulation int,
) (invdom.Mint, error) {
	now := time.Now().UTC()

	m, err := invdom.NewMint(
		"", // IDはrepo側で採番してよい
		tokenBlueprintID,
		productBlueprintID,
		productIDs,
		accumulation,
		now,
	)
	if err != nil {
		return invdom.Mint{}, err
	}

	return uc.repo.Create(ctx, m)
}

// UpdateMint updates an existing Mint.
// - UpdatedAt は repo 実装で更新しても良いが、ここでも更新して渡す。
func (uc *InventoryUsecase) UpdateMint(
	ctx context.Context,
	m invdom.Mint,
) (invdom.Mint, error) {
	// 最低限の必須チェック（ドメイン validate にも委譲）
	if strings.TrimSpace(m.ID) == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	m.UpdatedAt = time.Now().UTC()
	return uc.repo.Update(ctx, m)
}

// DeleteMint deletes a Mint by id.
func (uc *InventoryUsecase) DeleteMint(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.ErrInvalidMintID
	}
	return uc.repo.Delete(ctx, id)
}

// ============================================================
// Queries
// ============================================================

func (uc *InventoryUsecase) GetMintByID(ctx context.Context, id string) (invdom.Mint, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	return uc.repo.GetByID(ctx, id)
}

func (uc *InventoryUsecase) ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]invdom.Mint, error) {
	return uc.repo.ListByTokenBlueprintID(ctx, tokenBlueprintID)
}

func (uc *InventoryUsecase) ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error) {
	return uc.repo.ListByProductBlueprintID(ctx, productBlueprintID)
}

func (uc *InventoryUsecase) ListByTokenAndProductBlueprintID(ctx context.Context, tokenBlueprintID, productBlueprintID string) ([]invdom.Mint, error) {
	return uc.repo.ListByTokenAndProductBlueprintID(ctx, tokenBlueprintID, productBlueprintID)
}

// ============================================================
// Helpers (local)
// ============================================================

func normalizeIDs(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func buildInventoryDocID(tokenBlueprintID, productBlueprintID string) string {
	sanitize := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, "/", "_")
		return s
	}
	return sanitize(tokenBlueprintID) + "__" + sanitize(productBlueprintID)
}
