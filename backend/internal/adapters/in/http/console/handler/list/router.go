// backend/internal/adapters/in/http/console/handler/list/router.go
//
// Responsibility:
// - /lists 配下のパス/メソッドを解釈し、各機能ハンドラへディスパッチする。
// - ここでは「分岐のみ」を行い、実処理は各 feature ファイルへ委譲する。
//
// Firebase Storage migration policy:
// - backend は GCS signed URL を発行しない
// - /lists/{id}/images/signed-url は旧式 endpoint のため削除
// - frontend が Firebase Storage へ直接 upload する
// - backend は /lists/{listId}/images/{imageId} record の保存・取得・削除のみ担当する
package list

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

	// GET /lists/create-seed?inventoryId=...&modelIds=...
	if path == "/lists/create-seed" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		h.createSeed(w, r)
		return
	}

	if path == "/lists" {
		switch r.Method {
		case http.MethodPost:
			h.create(w, r)
			return
		case http.MethodGet:
			h.listIndex(w, r)
			return
		default:
			methodNotAllowed(w)
			return
		}
	}

	if !strings.HasPrefix(path, "/lists/") {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	rest := strings.TrimPrefix(path, "/lists/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if len(parts) > 1 {
		switch parts[1] {
		case "aggregate":
			if r.Method != http.MethodGet {
				methodNotAllowed(w)
				return
			}
			h.getAggregate(w, r, id)
			return

		case "images":
			sub := ""
			if len(parts) >= 3 {
				sub = parts[2]
			}

			// /lists/{id}/images
			if len(parts) == 2 {
				switch r.Method {
				case http.MethodGet:
					h.listImages(w, r, id)
					return
				case http.MethodPost:
					// Firebase Storage へ frontend から直接 upload 済みの画像情報を
					// Firestore record として保存する。
					h.saveImageFromGCS(w, r, id)
					return
				default:
					methodNotAllowed(w)
					return
				}
			}

			// /lists/{id}/images/{imageId}
			if len(parts) == 3 && sub != "" {
				if r.Method == http.MethodDelete {
					h.deleteImage(w, r, id, sub)
					return
				}

				methodNotAllowed(w)
				return
			}

			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return

		case "primary-image":
			if r.Method != http.MethodPut && r.Method != http.MethodPost && r.Method != http.MethodPatch {
				methodNotAllowed(w)
				return
			}
			h.setPrimaryImage(w, r, id)
			return

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
	}

	// /lists/{id}
	switch r.Method {
	case http.MethodGet:
		h.get(w, r, id)
		return
	case http.MethodPut, http.MethodPatch:
		h.update(w, r, id)
		return
	default:
		methodNotAllowed(w)
		return
	}
}
