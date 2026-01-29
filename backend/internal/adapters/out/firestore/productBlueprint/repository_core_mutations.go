// backend/internal/adapters/out/firestore/productBlueprint/repository_core_mutations.go
// Responsibility: ProductBlueprint の変更系（Create/Update/Delete/Save/printedガード/モデル参照追記/印刷確定）を提供する。
package productBlueprint

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbdom "narratives/internal/domain/productBlueprint"
)

// Create inserts a new ProductBlueprint (no upsert) from domain CreateInput.
func (r *ProductBlueprintRepositoryFS) Create(ctx context.Context, in pbdom.CreateInput) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	pb, err := pbdom.New(
		"", // ID は Firestore 採番
		in.ProductName,
		in.BrandID,
		in.ItemType,
		in.Fit,
		in.Material,
		in.Weight,
		in.QualityAssurance,
		in.ProductIdTag,
		in.AssigneeID,
		in.CreatedBy,
		createdAt,
		in.CompanyID,
	)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	// modelRefs（任意）
	if len(in.ModelRefs) > 0 {
		refs, err := sanitizeModelRefs(in.ModelRefs)
		if err != nil {
			return pbdom.ProductBlueprint{}, err
		}
		pb.ModelRefs = refs
	}

	docRef := r.col().NewDoc()
	pb.ID = docRef.ID

	data, err := productBlueprintToDoc(pb, pb.CreatedAt, pb.UpdatedAt)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	data["id"] = pb.ID

	if _, err := docRef.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return pbdom.ProductBlueprint{}, pbdom.ErrConflict
		}
		return pbdom.ProductBlueprint{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	return docToProductBlueprint(snap)
}

// Update updates a ProductBlueprint by patch (touches updatedAt; updatedBy is best-effort).
func (r *ProductBlueprintRepositoryFS) Update(ctx context.Context, id string, patch pbdom.Patch) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	docRef := r.col().Doc(id)

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	// printed guard
	if v, ok := snap.Data()["printed"].(bool); ok && v {
		return pbdom.ProductBlueprint{}, pbdom.ErrForbidden
	}

	pb, err := docToProductBlueprint(snap)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	// apply patch
	if patch.ProductName != nil {
		v := strings.TrimSpace(*patch.ProductName)
		if v == "" {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidProduct
		}
		pb.ProductName = v
	}
	if patch.BrandID != nil {
		v := strings.TrimSpace(*patch.BrandID)
		if v == "" {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidBrand
		}
		pb.BrandID = v
	}
	if patch.CompanyID != nil {
		// NOTE: contract says display-only; do not persist companyId changes via Patch
		// so intentionally ignored
	}
	if patch.ItemType != nil {
		if !pbdom.IsValidItemType(*patch.ItemType) {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidItemType
		}
		pb.ItemType = *patch.ItemType
	}
	if patch.Fit != nil {
		pb.Fit = strings.TrimSpace(*patch.Fit)
	}
	if patch.Material != nil {
		pb.Material = strings.TrimSpace(*patch.Material)
	}
	if patch.Weight != nil {
		if *patch.Weight < 0 {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidWeight
		}
		pb.Weight = *patch.Weight
	}
	if patch.QualityAssurance != nil {
		pb.QualityAssurance = dedupTrimStrings(*patch.QualityAssurance)
	}
	if patch.ProductIdTag != nil {
		if !pbdom.IsValidTagType(patch.ProductIdTag.Type) {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidTagType
		}
		pb.ProductIdTag = *patch.ProductIdTag
	}
	if patch.AssigneeID != nil {
		v := strings.TrimSpace(*patch.AssigneeID)
		if v == "" {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidAssignee
		}
		pb.AssigneeID = v
	}
	if patch.ModelRefs != nil {
		refs, err := sanitizeModelRefs(*patch.ModelRefs)
		if err != nil {
			return pbdom.ProductBlueprint{}, err
		}
		pb.ModelRefs = refs
	}

	// touch updatedAt (updatedBy は Patch にないため現状維持)
	pb.UpdatedAt = time.Now().UTC()

	data, err := productBlueprintToDoc(pb, pb.CreatedAt, pb.UpdatedAt)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	data["id"] = pb.ID

	if _, err := docRef.Set(ctx, data); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	snap, err = docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}
	return docToProductBlueprint(snap)
}

// Delete removes a ProductBlueprint by ID (physical delete).
func (r *ProductBlueprintRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ErrNotFound
		}
		return err
	}
	return nil
}

// AppendModelRefsWithoutTouch updates modelRefs only, without touching updatedAt/updatedBy.
// 要件:
// - updatedAt / updatedBy を更新しない（このメソッドでは一切書き換えない）
// - modelRefs のみ部分更新する
// - printed=true の場合は更新不可（ドメイン制約に合わせて ErrForbidden）
// 実装:
// - 既存 + 追加入力をマージし、重複排除しつつ displayOrder を 1..N で採番し直す
func (r *ProductBlueprintRepositoryFS) AppendModelRefsWithoutTouch(
	ctx context.Context,
	id string,
	refs []pbdom.ModelRef,
) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	docRef := r.col().Doc(id)

	// existence + printed guard
	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}
	if v, ok := snap.Data()["printed"].(bool); ok && v {
		return pbdom.ProductBlueprint{}, pbdom.ErrForbidden
	}

	currentPB, err := docToProductBlueprint(snap)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	appendRefs, err := sanitizeModelRefs(refs)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	if len(appendRefs) == 0 {
		return pbdom.ProductBlueprint{}, pbdom.WrapInvalid(nil, "modelRefs has no valid items")
	}

	merged := mergeAndRenumberModelRefs(currentPB.ModelRefs, appendRefs)

	modelRefsDoc := make([]map[string]any, 0, len(merged))
	for _, mr := range merged {
		modelRefsDoc = append(modelRefsDoc, map[string]any{
			"modelId":      mr.ModelID,
			"displayOrder": mr.DisplayOrder,
		})
	}

	// IMPORTANT: do NOT update updatedAt/updatedBy here
	if _, err := docRef.Update(ctx, []firestore.Update{
		{Path: "modelRefs", Value: modelRefsDoc},
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	// re-fetch
	snap, err = docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}
	return docToProductBlueprint(snap)
}

// MarkPrinted sets printed=true and returns updated blueprint.
func (r *ProductBlueprintRepositoryFS) MarkPrinted(ctx context.Context, id string) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	docRef := r.col().Doc(id)

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	// printed は bool のみ
	if v, ok := snap.Data()["printed"].(bool); ok && v {
		return pbdom.ProductBlueprint{}, pbdom.ErrForbidden
	}

	now := time.Now().UTC()
	if _, err := docRef.Update(ctx, []firestore.Update{
		{Path: "printed", Value: true},
		{Path: "updatedAt", Value: now},
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	snap, err = docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	return docToProductBlueprint(snap)
}

// Save upserts a ProductBlueprint (Set).
// - printed=true の場合は更新不可（ErrForbidden）
// - updatedAt は now に更新（updatedBy は v.UpdatedBy を保存する）
// - companyId 境界の検証は usecase 側で担保されている前提（必要ならここで追加可能）
func (r *ProductBlueprintRepositoryFS) Save(ctx context.Context, v pbdom.ProductBlueprint) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	docRef := r.col().Doc(id)

	// existence + printed guard
	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}
	if vv, ok := snap.Data()["printed"].(bool); ok && vv {
		return pbdom.ProductBlueprint{}, pbdom.ErrForbidden
	}

	// createdAt は維持（入力がゼロなら既存から補完）
	now := time.Now().UTC()

	if v.CreatedAt.IsZero() {
		// 既存 createdAt を優先
		if t, ok := snap.Data()["createdAt"].(time.Time); ok && !t.IsZero() {
			v.CreatedAt = t.UTC()
		} else {
			v.CreatedAt = now
		}
	} else {
		v.CreatedAt = v.CreatedAt.UTC()
	}

	v.UpdatedAt = now

	// doc 化
	data, err := productBlueprintToDoc(v, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	data["id"] = id

	// upsert
	if _, err := docRef.Set(ctx, data); err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	// re-fetch
	snap, err = docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}
	return docToProductBlueprint(snap)
}
