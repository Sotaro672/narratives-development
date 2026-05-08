// backend/internal/adapters/in/http/console/handler/list/feature_mutation.go
//
// Responsibility:
// - List の作成/更新（mutation）を担当する。
// - request body の読み取り/正規化と、usecase 呼び出しを行う。
package list

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	listdom "narratives/internal/domain/list"
)

func (h *ListHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}

	var item listdom.List
	if err := json.Unmarshal(body, &item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if item.InventoryID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "inventoryId is required"})
		return
	}
	if item.AssigneeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "assigneeId is required"})
		return
	}
	if item.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "title is required"})
		return
	}
	if item.CreatedBy == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "createdBy is required"})
		return
	}

	if item.ID == item.InventoryID {
		item.ID = ""
	}

	if string(item.Status) == "" {
		item.Status = listdom.ListStatus("listing")
	}

	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = &now
	item.UpdatedBy = nil
	item.DeletedAt = nil
	item.DeletedBy = nil

	created, err := h.uc.Create(ctx, item)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *ListHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}

	log.Printf("[list_handler] UPDATE received method=%s id=%q bytes=%d", r.Method, id, len(body))

	var in listdom.List
	if err := json.Unmarshal(body, &in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	current, err := h.uc.GetByID(ctx, id)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	if in.Title != "" {
		current.Title = in.Title
	}
	if in.Description != "" {
		current.Description = in.Description
	}
	if string(in.Status) != "" {
		current.Status = listdom.ListStatus(string(in.Status))
	}
	if in.Prices != nil {
		current.Prices = in.Prices
	}
	if in.UpdatedBy != nil {
		current.UpdatedBy = in.UpdatedBy
	}

	now := time.Now().UTC()
	current.UpdatedAt = &now

	updater, ok := any(h.uc).(interface {
		Update(ctx context.Context, item listdom.List) (listdom.List, error)
	})
	if !ok {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	updated, err := updater.Update(ctx, current)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	if h.qDetail != nil {
		if dto, e := h.qDetail.BuildListDetailDTO(ctx, id); e == nil {
			_ = json.NewEncoder(w).Encode(dto)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(updated)
}
