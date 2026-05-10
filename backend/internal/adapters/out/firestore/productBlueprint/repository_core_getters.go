// backend/internal/adapters/out/firestore/productBlueprint/repository_core_getters.go
// Responsibility: ProductBlueprint の取得系（GetByID・部分取得・modelID 逆引き・Patch生成・存在判定・会社ID別ID列挙・ブランドID別ID列挙）を提供する。
package productBlueprint

import (
	"context"
	"errors"

	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbdom "narratives/internal/domain/productBlueprint"
)

// GetByID returns a single ProductBlueprint by ID.
func (r *ProductBlueprintRepositoryFS) GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	return docToProductBlueprint(snap)
}

// GetBrandIDByID returns brandId only.
func (r *ProductBlueprintRepositoryFS) GetBrandIDByID(ctx context.Context, id string) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	if id == "" {
		return "", pbdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", pbdom.ErrNotFound
		}
		return "", err
	}

	data := snap.Data()
	if data != nil {
		if v, ok := data["brandId"].(string); ok && v != "" {
			return v, nil
		}
	}

	pb, err := docToProductBlueprint(snap)
	if err != nil {
		return "", err
	}
	brandID := pb.BrandID
	if brandID == "" {
		return "", pbdom.ErrNotFound
	}
	return brandID, nil
}

// ListIDsByBrandID returns blueprint IDs for given brandID.
func (r *ProductBlueprintRepositoryFS) ListIDsByBrandID(ctx context.Context, brandID string) ([]string, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if brandID == "" {
		return nil, pbdom.ErrNotFound
	}

	iter := r.col().
		Where("brandId", "==", brandID).
		Documents(ctx)
	defer iter.Stop()

	var ids []string
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		if snap == nil {
			break
		}
		ids = append(ids, snap.Ref.ID)
	}

	return ids, nil
}

// GetProductNameByID returns productName only.
func (r *ProductBlueprintRepositoryFS) GetProductNameByID(ctx context.Context, id string) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	if id == "" {
		return "", pbdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", pbdom.ErrNotFound
		}
		return "", err
	}

	data := snap.Data()
	if data != nil {
		if v, ok := data["productName"].(string); ok && v != "" {
			return v, nil
		}
	}

	pb, err := docToProductBlueprint(snap)
	if err != nil {
		return "", err
	}
	name := pb.ProductName
	if name == "" {
		return "", pbdom.ErrNotFound
	}
	return name, nil
}

// GetIDByModelID returns productBlueprintId by modelID.
// 方針: models / model_variations 側の productBlueprintId を参照して ProductBlueprint を特定する。
func (r *ProductBlueprintRepositoryFS) GetIDByModelID(ctx context.Context, modelID string) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	if modelID == "" {
		return "", pbdom.ErrNotFound
	}

	// models / model_variations 側に productBlueprintId を持っているケース（docID=modelID 想定）
	collections := []string{"models", "model_variations", "modelVariations"}

	for _, col := range collections {
		doc, err := r.Client.Collection(col).Doc(modelID).Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", err
			}
			continue
		}

		data := doc.Data()
		if data == nil {
			continue
		}

		if v, ok := data["productBlueprintId"]; ok && v != nil {
			if s, ok := v.(string); ok && s != "" {
				return s, nil
			}
		}
		// allow schema drift (rare): productBlueprintID
		if v, ok := data["productBlueprintID"]; ok && v != nil {
			if s, ok := v.(string); ok && s != "" {
				return s, nil
			}
		}
	}

	return "", pbdom.ErrNotFound
}

// GetModelRefsByModelID gets modelRefs (displayOrder included) by modelID.
// 方針: models / model_variations 側の productBlueprintId を参照して ProductBlueprint を特定する。
func (r *ProductBlueprintRepositoryFS) GetModelRefsByModelID(ctx context.Context, modelID string) ([]pbdom.ModelRef, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if modelID == "" {
		return nil, pbdom.ErrNotFound
	}

	// models / model_variations 側に productBlueprintId を持っているケース（docID=modelID 想定）
	collections := []string{"models", "model_variations", "modelVariations"}

	for _, col := range collections {
		doc, err := r.Client.Collection(col).Doc(modelID).Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			continue
		}

		data := doc.Data()
		if data == nil {
			continue
		}

		if v, ok := data["productBlueprintId"]; ok && v != nil {
			if s, ok := v.(string); ok && s != "" {
				blueprintID := s
				pb, err := r.GetByID(ctx, blueprintID)
				if err != nil {
					return nil, err
				}
				return cloneModelRefs(pb.ModelRefs), nil
			}
		}
	}

	return nil, pbdom.ErrNotFound
}

// GetPatchByID returns patch for mint/read-model usecases (display fields are not filled here).
func (r *ProductBlueprintRepositoryFS) GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error) {
	if r.Client == nil {
		return pbdom.Patch{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return pbdom.Patch{}, pbdom.ErrNotFound
	}

	pb, err := r.GetByID(ctx, id)
	if err != nil {
		return pbdom.Patch{}, err
	}

	name := pb.ProductName
	brandID := pb.BrandID
	category := pb.ProductBlueprintCategory
	fit := pb.Fit
	material := pb.Material
	weight := pb.Weight
	qa := make([]string, len(pb.QualityAssurance))
	copy(qa, pb.QualityAssurance)
	productIdTag := pb.ProductIdTag
	assigneeID := pb.AssigneeID

	var refsPtr *[]pbdom.ModelRef
	if pb.ModelRefs != nil {
		refs := cloneModelRefs(pb.ModelRefs)
		refsPtr = &refs
	}

	return pbdom.Patch{
		ProductName: &name,
		BrandID:     &brandID,

		ProductBlueprintCategory: &category,

		Fit:              &fit,
		Material:         &material,
		Weight:           &weight,
		QualityAssurance: &qa,
		ProductIdTag:     &productIdTag,
		AssigneeID:       &assigneeID,
		ModelRefs:        refsPtr,
	}, nil
}

// ListIDsByCompany returns blueprint IDs for given companyID.
func (r *ProductBlueprintRepositoryFS) ListIDsByCompany(ctx context.Context, companyID string) ([]string, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if companyID == "" {
		return nil, pbdom.ErrInvalidCompanyID
	}

	iter := r.col().
		Where("companyId", "==", companyID).
		Documents(ctx)
	defer iter.Stop()

	var ids []string
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		if snap == nil {
			// 念のため（通常は iterator.Done で抜ける）
			break
		}
		ids = append(ids, snap.Ref.ID)
	}
	return ids, nil
}

// Exists reports whether a ProductBlueprint with given ID exists.
func (r *ProductBlueprintRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	if id == "" {
		return false, nil
	}

	_, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
