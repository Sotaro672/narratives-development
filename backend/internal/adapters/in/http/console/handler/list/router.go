// backend/internal/adapters/in/http/console/handler/list/router.go
//
// Responsibility:
// - /lists 配下のパス/メソッドを解釈し、各機能ハンドラへディスパッチする。
// - ここでは「分岐のみ」を行い、実処理は各 feature ファイルへ委譲する。
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
	id := strings.TrimSpace(parts[0])
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
			// /lists/{id}/images/{sub}
			sub := ""
			if len(parts) >= 3 {
				sub = strings.TrimSpace(parts[2])
			}

			// ✅ /lists/{id}/images/signed-url (先に判定して安全にする)
			if strings.EqualFold(sub, "signed-url") {
				if r.Method != http.MethodPost {
					methodNotAllowed(w)
					return
				}
				h.issueSignedURL(w, r, id)
				return
			}

			// /lists/{id}/images
			if len(parts) == 2 {
				switch r.Method {
				case http.MethodGet:
					h.listImages(w, r, id)
					return
				case http.MethodPost:
					// signed URL PUT 後の object をレコード化
					h.saveImageFromGCS(w, r, id)
					return
				default:
					methodNotAllowed(w)
					return
				}
			}

			// /lists/{id}/images/{imageId} DELETE
			if r.Method == http.MethodDelete && sub != "" {
				h.deleteImage(w, r, id, sub)
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
