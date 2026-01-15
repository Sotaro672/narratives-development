// backend/internal/application/mint/mint_request_repo_adapter.go
package mint

import (
	"context"
	"errors"
	"log"
	"strings"

	mintdom "narratives/internal/domain/mint"
)

// MintRequestRepositoryAdapter は、mintdom.MintRepository の互換差分（GetByID/Get）を吸収する Adapter です。
// - 目的: Usecase から「repo のメソッド揺れ」を排除し、オーケストレーションに集中させる。
// - 契約: NotFound の場合は (nil, nil) を返す（Usecase 側で ErrNotFound に変換する方針）。
type MintRequestRepositoryAdapter struct {
	repo mintdom.MintRepository
}

func NewMintRequestRepositoryAdapter(repo mintdom.MintRepository) *MintRequestRepositoryAdapter {
	return &MintRequestRepositoryAdapter{repo: repo}
}

// Load loads mint by id with compatibility for GetByID/Get.
// NotFound の場合は (*mintdom.Mint)(nil), nil を返す。
func (a *MintRequestRepositoryAdapter) Load(ctx context.Context, id string) (*mintdom.Mint, error) {
	if a == nil {
		return nil, errors.New("mint request repository adapter is nil")
	}
	if a.repo == nil {
		return nil, errors.New("mint repo is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("id is empty")
	}

	// 1) GetByID があれば最優先
	if getter, ok := any(a.repo).(interface {
		GetByID(ctx context.Context, id string) (mintdom.Mint, error)
	}); ok {
		m, err := getter.GetByID(ctx, id)
		if err == nil {
			return &m, nil
		}
		if isNotFoundErr(err) {
			return nil, nil
		}
		log.Printf("[mint_repo_adapter] Load mint(GetByID) error id=%q err=%v", id, err)
		return nil, err
	}

	// 2) Get があればそれを使う
	if getter, ok := any(a.repo).(interface {
		Get(ctx context.Context, id string) (mintdom.Mint, error)
	}); ok {
		m, err := getter.Get(ctx, id)
		if err == nil {
			return &m, nil
		}
		if isNotFoundErr(err) {
			return nil, nil
		}
		log.Printf("[mint_repo_adapter] Load mint(Get) error id=%q err=%v", id, err)
		return nil, err
	}

	log.Printf("[mint_repo_adapter] Load mintRepo has no GetByID/Get (type=%T)", a.repo)
	return nil, errors.New("mint repo does not support GetByID/Get")
}
