// backend/internal/adapters/in/http/handlers/permission_handler.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	usecase "narratives/internal/application/usecase"
	dcommon "narratives/internal/domain/common"
	permissiondom "narratives/internal/domain/permission"
)

// PermissionHandler は /permissions 関連のエンドポイントを担当します。
type PermissionHandler struct {
	uc *usecase.PermissionUsecase
}

// NewPermissionHandler はHTTPハンドラを初期化します。
func NewPermissionHandler(uc *usecase.PermissionUsecase) http.Handler {
	return &PermissionHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *PermissionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// 一覧: GET /permissions または /permissions/
	case r.Method == http.MethodGet &&
		(r.URL.Path == "/permissions" || r.URL.Path == "/permissions/"):
		h.list(w, r)

	// 詳細: GET /permissions/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/permissions/"):
		id := strings.TrimPrefix(r.URL.Path, "/permissions/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /permissions?page=&perPage=&sort=&order=&search=&categories=
func (h *PermissionHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	// ページング
	pageNum, _ := strconv.Atoi(q.Get("page"))
	if pageNum <= 0 {
		pageNum = 1
	}
	perPage, _ := strconv.Atoi(q.Get("perPage"))
	if perPage <= 0 {
		perPage = 10
	}
	page := permissiondom.Page{
		Number:  pageNum,
		PerPage: perPage,
	}

	// フィルタ
	filter := permissiondom.Filter{
		FilterCommon: dcommon.FilterCommon{
			SearchQuery: strings.TrimSpace(q.Get("search")),
		},
	}
	// categories=wallet,brand,member → []PermissionCategory{"wallet", "brand", "member"}
	if cats := strings.TrimSpace(q.Get("categories")); cats != "" {
		for _, c := range strings.Split(cats, ",") {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			filter.Categories = append(filter.Categories, permissiondom.PermissionCategory(c))
		}
	}

	// ソート
	sort := permissiondom.Sort{
		// dcommon.SortColumn は存在しないので、素直に string を渡す
		Column: strings.TrimSpace(q.Get("sort")),
		Order:  dcommon.SortOrder(strings.TrimSpace(q.Get("order"))),
	}

	result, err := h.uc.List(ctx, filter, sort, page)
	if err != nil {
		writePermissionErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// GET /permissions/{id}
func (h *PermissionHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	perm, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writePermissionErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(perm)
}

// エラーハンドリング
func writePermissionErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case permissiondom.ErrInvalidID:
		code = http.StatusBadRequest
	case permissiondom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
