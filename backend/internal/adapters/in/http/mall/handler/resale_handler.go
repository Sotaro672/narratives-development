// backend/internal/adapters/in/http/mall/handler/resale_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	mallquery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
	resaledom "narratives/internal/domain/resale"
)

// ResaleQuery is the read-side port used by mall resale handler.
// mall.ResaleQuery satisfies this interface.
//
// NOTE:
// /mall/me/resales は「自分の出品管理」を基本とし、/{resaleId}/reports は他者出品の通報にも利用する。
// /mall/resales/avatar/{avatarId} は公開アバターの出品一覧表示専用。
// 所有者一覧は ListOwned、公開アバター一覧は List を使用する。
type ResaleQuery interface {
	List(
		ctx context.Context,
		filter resaledom.Filter,
		sort resaledom.Sort,
		page resaledom.Page,
	) (resaledom.PageResult[resaledom.Resale], error)

	ListOwned(
		ctx context.Context,
		avatarID string,
		page resaledom.Page,
	) (resaledom.PageResult[resaledom.Resale], error)

	ListChatItems(
		ctx context.Context,
		avatarID string,
	) ([]mallquery.ResaleChatListItem, error)

	CountUnreadCommentsByAvatarID(
		ctx context.Context,
		avatarID string,
	) (int, error)

	GetByID(
		ctx context.Context,
		id string,
	) (resaledom.Resale, error)

	ListImages(
		ctx context.Context,
		resaleID string,
	) ([]resaledom.ResaleImage, error)
}

type ResaleHandler struct {
	uc             *usecase.ResaleUsecase
	query          ResaleQuery
	resaleReviewUC *usecase.ResaleReviewUsecase
	reportUC       *usecase.ReportUsecase
}

type NewResaleHandlerParams struct {
	UC             *usecase.ResaleUsecase
	Query          ResaleQuery
	ResaleReviewUC *usecase.ResaleReviewUsecase
	ReportUC       *usecase.ReportUsecase
}

func NewResaleHandler(p NewResaleHandlerParams) http.Handler {
	return &ResaleHandler{
		uc:             p.UC,
		query:          p.Query,
		resaleReviewUC: p.ResaleReviewUC,
		reportUC:       p.ReportUC,
	}
}

const (
	meResalesPath     = "/mall/me/resales"
	publicResalesPath = "/mall/resales"
)

func (h *ResaleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == "" {
		path = r.URL.Path
	}

	if path == publicResalesPath ||
		strings.HasPrefix(path, publicResalesPath+"/") {
		h.servePublic(w, r, path)
		return
	}

	if path == meResalesPath {
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

	if path == meResalesPath+"/chats" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.listOwnedResaleChats(w, r)
		return
	}

	if path == meResalesPath+"/chat-badge-count" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.getOwnedResaleChatBadgeCount(w, r)
		return
	}

	if !strings.HasPrefix(path, meResalesPath+"/") {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not_found",
		})
		return
	}

	rest := strings.TrimPrefix(path, meResalesPath+"/")
	parts := strings.Split(rest, "/")
	resaleID := parts[0]

	if resaleID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid resaleId",
		})
		return
	}

	if len(parts) > 1 {
		switch parts[1] {
		case "reports":
			if len(parts) != 2 {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "not_found",
				})
				return
			}

			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}

			h.reportResale(w, r, resaleID)
			return

		case "images", "condition-images":
			imageID := ""

			if len(parts) >= 3 {
				imageID = parts[2]
			}

			if len(parts) == 2 {
				switch r.Method {
				case http.MethodGet:
					h.listImages(w, r, resaleID)
					return
				case http.MethodPost:
					h.createImageFromFirebaseStorage(w, r, resaleID)
					return
				default:
					methodNotAllowed(w)
					return
				}
			}

			if len(parts) == 3 && imageID != "" {
				if r.Method == http.MethodDelete {
					h.deleteImage(w, r, resaleID, imageID)
					return
				}

				methodNotAllowed(w)
				return
			}

			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "not_found",
			})
			return

		case "primary-image":
			if r.Method != http.MethodPut {
				methodNotAllowed(w)
				return
			}

			h.setPrimaryImage(w, r, resaleID)
			return

		case "comments":
			if len(parts) == 2 {
				switch r.Method {
				case http.MethodGet:
					h.listOwnedResaleComments(w, r, resaleID)
					return
				case http.MethodPost:
					h.createOwnedResaleComment(w, r, resaleID)
					return
				default:
					methodNotAllowed(w)
					return
				}
			}

			if len(parts) == 3 && parts[2] == "mark-as-read" {
				if r.Method != http.MethodPost {
					methodNotAllowed(w)
					return
				}

				h.markOwnedResaleCommentsRead(w, r, resaleID)
				return
			}

			if len(parts) == 3 && parts[2] != "" {
				commentID := parts[2]

				if r.Method != http.MethodDelete {
					methodNotAllowed(w)
					return
				}

				h.deleteOwnedResaleComment(w, r, resaleID, commentID)
				return
			}

			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "not_found",
			})
			return

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "not_found",
			})
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.get(w, r, resaleID)
		return
	case http.MethodPut:
		h.update(w, r, resaleID)
		return
	case http.MethodDelete:
		h.delete(w, r, resaleID)
		return
	default:
		methodNotAllowed(w)
		return
	}
}
