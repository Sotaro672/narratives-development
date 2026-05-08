// backend/internal/adapters/out/firestore/productBlueprint/repository_core_mutations.go
// Responsibility: ProductBlueprint の変更系（Create/Update/Delete/Save/printedガード/モデル参照追記/印刷確定）を提供する。
package productBlueprint

import (
	"context"
	"errors"
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

	// ★ 修正案A前提: usecase が ID を生成して渡す
	id := in.ID
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	now := time.Now().UTC()

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	// ★ domain.New は ID 必須。Firestore採番は使わない。
	pb, err := pbdom.New(
		id,
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

	// ★ docId も pb.ID と一致させる（衝突時は AlreadyExists を返せる）
	docRef := r.col().Doc(pb.ID)

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

// AppendModelRefsWithoutTouch updates modelRefs only, without touching updatedAt/updatedBy.
// - updatedAt / updatedBy を更新しない（このメソッドでは一切書き換えない）
// - modelRefs のみ部分更新する
// - printed=true の場合は更新不可（ErrForbidden）
func (r *ProductBlueprintRepositoryFS) AppendModelRefsWithoutTouch(
	ctx context.Context,
	id string,
	refs []pbdom.ModelRef,
) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

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

// Delete removes a ProductBlueprint by ID (physical delete).
func (r *ProductBlueprintRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	if id == "" {
		// contract次第だが、InvalidID の方が自然
		return pbdom.ErrInvalidID
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

// MarkPrinted sets printed=true and returns updated blueprint.
func (r *ProductBlueprintRepositoryFS) MarkPrinted(ctx context.Context, id string) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

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

	// printed=true は更新不可（運用次第で idempotent にしても良いが現状は forbidden）
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

// Update updates a ProductBlueprint by patch (touches updatedAt; updatedBy is best-effort).
func (r *ProductBlueprintRepositoryFS) Update(ctx context.Context, id string, patch pbdom.Patch) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

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
		v := *patch.ProductName
		if v == "" {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidProduct
		}
		pb.ProductName = v
	}
	if patch.BrandID != nil {
		v := *patch.BrandID
		if v == "" {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidBrand
		}
		pb.BrandID = v
	}
	if patch.ItemType != nil {
		if !pbdom.IsValidItemType(*patch.ItemType) {
			return pbdom.ProductBlueprint{}, pbdom.ErrInvalidItemType
		}
		pb.ItemType = *patch.ItemType
	}
	if patch.Fit != nil {
		pb.Fit = *patch.Fit
	}
	if patch.Material != nil {
		pb.Material = *patch.Material
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
		v := *patch.AssigneeID
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

	// touch updatedAt
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

// Save upserts a ProductBlueprint (Set).
// - printed=true の場合は更新不可（ErrForbidden）
// - updatedAt は now に更新（updatedBy は v.UpdatedBy を保存する）
// - companyId 境界の検証は usecase 側で担保されている前提
func (r *ProductBlueprintRepositoryFS) Save(ctx context.Context, v pbdom.ProductBlueprint) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id := v.ID
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
		if t, ok := snap.Data()["createdAt"].(time.Time); ok && !t.IsZero() {
			v.CreatedAt = t.UTC()
		} else {
			v.CreatedAt = now
		}
	} else {
		v.CreatedAt = v.CreatedAt.UTC()
	}

	v.UpdatedAt = now

	data, err := productBlueprintToDoc(v, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	data["id"] = id

	if _, err := docRef.Set(ctx, data); err != nil {
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
