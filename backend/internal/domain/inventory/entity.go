package inventory

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrNotFound                  = errors.New("inventory not found")
	ErrInvalidMintID             = errors.New("invalid inventory id")
	ErrInvalidTokenBlueprintID   = errors.New("invalid tokenBlueprintID")
	ErrInvalidProductBlueprintID = errors.New("invalid productBlueprintID")
	ErrInvalidModelID            = errors.New("invalid modelID")
	ErrInvalidProducts           = errors.New("invalid products")
	ErrInvalidAccumulation       = errors.New("invalid accumulation")
)

type Mint struct {
	ID                 string
	TokenBlueprintID   string
	ProductBlueprintID string // 互換/参照用に残す（docId には使わない）
	ModelID            string // ★ NEW: docId の主キー側

	Products     []string
	Accumulation int

	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewMint(
	id string,
	tokenBlueprintID string,
	productBlueprintID string,
	modelID string,
	products []string,
	accumulation int,
	now time.Time,
) (Mint, error) {
	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return Mint{}, ErrInvalidTokenBlueprintID
	}

	pbID := strings.TrimSpace(productBlueprintID)
	if pbID == "" {
		return Mint{}, ErrInvalidProductBlueprintID
	}

	mID := strings.TrimSpace(modelID)
	if mID == "" {
		return Mint{}, ErrInvalidModelID
	}

	ps := normalizeIDs(products)
	if len(ps) == 0 {
		return Mint{}, ErrInvalidProducts
	}

	if accumulation <= 0 {
		// accumulation は「保持している products の数」を基本とする
		accumulation = len(ps)
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	return Mint{
		ID:                 strings.TrimSpace(id),
		TokenBlueprintID:   tbID,
		ProductBlueprintID: pbID,
		ModelID:            mID,
		Products:           ps,
		Accumulation:       accumulation,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

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
