// backend/internal/adapters/out/firestore/productBlueprint/repository_history.go
// Responsibility: product_blueprints_history サブコレクションを用いた履歴保存・履歴一覧・履歴単体取得（versioned snapshot）を提供する。
package productBlueprint

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbdom "narratives/internal/domain/productBlueprint"
)

func (r *ProductBlueprintRepositoryFS) SaveHistorySnapshot(ctx context.Context, blueprintID string, h pbdom.HistoryRecord) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return pbdom.ErrInvalidID
	}

	if strings.TrimSpace(h.Blueprint.ID) == "" || h.Blueprint.ID != blueprintID {
		h.Blueprint.ID = blueprintID
	}

	if h.UpdatedAt.IsZero() {
		h.UpdatedAt = h.Blueprint.UpdatedAt
	}
	if h.UpdatedAt.IsZero() {
		h.UpdatedAt = time.Now().UTC()
	}

	docID := fmt.Sprintf("%d", h.Version)
	docRef := r.historyCol(blueprintID).Doc(docID)

	data, err := productBlueprintToDoc(h.Blueprint, h.Blueprint.CreatedAt, h.Blueprint.UpdatedAt)
	if err != nil {
		return err
	}
	data["id"] = blueprintID
	data["version"] = h.Version
	data["historyUpdatedAt"] = h.UpdatedAt.UTC()
	if h.UpdatedBy != nil {
		if s := strings.TrimSpace(*h.UpdatedBy); s != "" {
			data["historyUpdatedBy"] = s
		}
	}

	if _, err := docRef.Set(ctx, data); err != nil {
		return err
	}
	return nil
}

func (r *ProductBlueprintRepositoryFS) ListHistory(ctx context.Context, blueprintID string) ([]pbdom.HistoryRecord, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return nil, pbdom.ErrInvalidID
	}

	q := r.historyCol(blueprintID).OrderBy("version", firestore.Desc)
	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	out := make([]pbdom.HistoryRecord, 0, len(snaps))
	for _, snap := range snaps {
		data := snap.Data()
		if data == nil {
			continue
		}

		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}

		var version int64
		if v, ok := data["version"]; ok {
			switch x := v.(type) {
			case int64:
				version = x
			case int:
				version = int64(x)
			case float64:
				version = int64(x)
			}
		}

		var histUpdatedAt time.Time
		if v, ok := data["historyUpdatedAt"].(time.Time); ok && !v.IsZero() {
			histUpdatedAt = v.UTC()
		} else {
			histUpdatedAt = pb.UpdatedAt
		}

		var histUpdatedBy *string
		if v, ok := data["historyUpdatedBy"].(string); ok && strings.TrimSpace(v) != "" {
			s := strings.TrimSpace(v)
			histUpdatedBy = &s
		} else {
			histUpdatedBy = pb.UpdatedBy
		}

		out = append(out, pbdom.HistoryRecord{
			Blueprint: pb,
			Version:   version,
			UpdatedAt: histUpdatedAt,
			UpdatedBy: histUpdatedBy,
		})
	}
	return out, nil
}

func (r *ProductBlueprintRepositoryFS) GetHistoryByVersion(ctx context.Context, blueprintID string, version int64) (pbdom.HistoryRecord, error) {
	if r.Client == nil {
		return pbdom.HistoryRecord{}, errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return pbdom.HistoryRecord{}, pbdom.ErrInvalidID
	}

	docID := fmt.Sprintf("%d", version)
	snap, err := r.historyCol(blueprintID).Doc(docID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.HistoryRecord{}, pbdom.ErrNotFound
		}
		return pbdom.HistoryRecord{}, err
	}

	data := snap.Data()
	if data == nil {
		return pbdom.HistoryRecord{}, fmt.Errorf("empty history document: %s", snap.Ref.Path)
	}

	pb, err := docToProductBlueprint(snap)
	if err != nil {
		return pbdom.HistoryRecord{}, err
	}

	var ver int64
	if v, ok := data["version"]; ok {
		switch x := v.(type) {
		case int64:
			ver = x
		case int:
			ver = int64(x)
		case float64:
			ver = int64(x)
		}
	}
	if ver == 0 {
		ver = version
	}

	var histUpdatedAt time.Time
	if v, ok := data["historyUpdatedAt"].(time.Time); ok && !v.IsZero() {
		histUpdatedAt = v.UTC()
	} else {
		histUpdatedAt = pb.UpdatedAt
	}

	var histUpdatedBy *string
	if v, ok := data["historyUpdatedBy"].(string); ok && strings.TrimSpace(v) != "" {
		s := strings.TrimSpace(v)
		histUpdatedBy = &s
	} else {
		histUpdatedBy = pb.UpdatedBy
	}

	return pbdom.HistoryRecord{
		Blueprint: pb,
		Version:   ver,
		UpdatedAt: histUpdatedAt,
		UpdatedBy: histUpdatedBy,
	}, nil
}
