// backend/internal/adapters/in/http/console/handler/inventory/query_detail_or_fallback.go
package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	invdom "narratives/internal/domain/inventory"
)

// ============================================================
// ✅ Detail endpoint（確定）
// - Query があれば必ず GetDetailByID を呼ぶ
// - Query が無い場合のみ UC fallback
// ============================================================

func (h *InventoryHandler) GetDetailByIDQueryOrFallback(w http.ResponseWriter, r *http.Request, inventoryID string) {
	ctx := r.Context()
	id := strings.TrimSpace(inventoryID)

	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	// 1) Query があるなら確定で呼ぶ
	if h.Q != nil {
		dto, err := h.Q.GetDetailByID(ctx, id)
		if err != nil {
			if errors.Is(err, invdom.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// ✅ 返却 DTO に "tokenBlueprintPatch" を合成して返す
		// - dto 型を変えずに拡張するため、JSON round-trip で map にする
		respMap, mErr := anyToMap(dto)
		if mErr == nil {
			tbID := firstNonEmptyString(
				asString(respMap["tokenBlueprintId"]),
				asString(respMap["tokenBlueprintID"]),
				asString(respMap["tokenBlueprintId"]),
			)

			if tbID != "" {
				if patch, _ := h.FetchTokenBlueprintPatch(ctx, tbID); patch != nil {
					// dto 側に既に tokenBlueprintPatch があっても上書きする（常に最新を優先）
					respMap["tokenBlueprintPatch"] = patch
				}
			}

			writeJSON(w, http.StatusOK, respMap)
			return
		}

		// map 化に失敗したら通常返却
		writeJSON(w, http.StatusOK, dto)
		return
	}

	// 2) fallback: UC.GetByID
	if h.UC == nil {
		writeError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	m, err := h.UC.GetByID(ctx, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	// entity.go 準拠: reserved を考慮した “表示用” 合計も返す（fallback のみ）
	totalAcc := totalAccumulation(m)
	totalRes := totalReserved(m)
	totalAvail := totalAvailable(m)

	resp := map[string]any{
		"inventoryId":        strings.TrimSpace(m.ID),
		"id":                 strings.TrimSpace(m.ID),
		"inventoryIds":       []string{strings.TrimSpace(m.ID)},
		"tokenBlueprintId":   strings.TrimSpace(m.TokenBlueprintID),
		"productBlueprintId": strings.TrimSpace(m.ProductBlueprintID),
		"modelId":            "",

		"productBlueprintPatch": map[string]any{},
		"tokenBlueprintPatch":   map[string]any{}, // ✅ 追加（fallback は空で返す）
		"rows":                  []any{},

		// 互換のため totalStock を維持しつつ、reserved/available も追加
		"totalStock":        totalAvail,
		"totalAccumulation": totalAcc,
		"totalReserved":     totalRes,
		"totalAvailable":    totalAvail,
	}

	writeJSON(w, http.StatusOK, resp)
}

// ============================================================
// ✅ TokenBlueprint Patch（呼び出し関数）
// - Query 側に実装があれば呼ぶ（存在しない場合は nil を返す）
// ============================================================

func (h *InventoryHandler) FetchTokenBlueprintPatch(ctx context.Context, tokenBlueprintID string) (any, error) {
	if h == nil || h.Q == nil {
		return nil, nil
	}
	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return nil, nil
	}

	// 期待する Query 側メソッド名（将来の命名揺れを許容）
	candidates := []string{
		"GetTokenBlueprintPatchByID",
		"GetTokenBlueprintPatchById",
		"FetchTokenBlueprintPatchByID",
		"FetchTokenBlueprintPatchById",
		"GetTokenBlueprintPatch",
		"FetchTokenBlueprintPatch",
	}

	qv := reflect.ValueOf(h.Q)
	for _, name := range candidates {
		m := qv.MethodByName(name)
		if !m.IsValid() {
			continue
		}

		// func(ctx context.Context, id string) (T, error)
		mt := m.Type()
		if mt.NumIn() != 2 ||
			mt.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() ||
			mt.In(1).Kind() != reflect.String {
			continue
		}
		if mt.NumOut() != 2 || !mt.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			continue
		}

		outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(id)})

		var outErr error
		if !outs[1].IsNil() {
			outErr = outs[1].Interface().(error)
		}
		if outErr != nil {
			return nil, outErr
		}

		patch := outs[0].Interface()
		return patch, nil
	}

	// Query 側に未実装（またはシグネチャ不一致）
	return nil, nil
}

// ============================================================
// helpers for dto merge
// ============================================================

func anyToMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func asString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func firstNonEmptyString(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
