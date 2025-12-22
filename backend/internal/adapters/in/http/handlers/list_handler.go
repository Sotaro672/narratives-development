// backend/internal/adapters/in/http/handlers/list_handler.go
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	query "narratives/internal/application/query"
	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
	listimgdom "narratives/internal/domain/listImage"
)

type ListImageUploader interface {
	Upload(ctx context.Context, in listimgdom.UploadImageInput) (*listimgdom.ListImage, error)
}

type ListImageDeleter interface {
	Delete(ctx context.Context, imageID string) error
}

type ListHandler struct {
	uc *usecase.ListUsecase

	// ✅ split: listManagement.tsx / listCreate.tsx 向け
	qMgmt *query.ListManagementQuery

	// ✅ split: listDetail.tsx 向け
	qDetail *query.ListDetailQuery

	// ✅ ListImage (optional)
	imgUploader ListImageUploader
	imgDeleter  ListImageDeleter
}

func NewListHandler(uc *usecase.ListUsecase) http.Handler {
	return &ListHandler{uc: uc, qMgmt: nil, qDetail: nil, imgUploader: nil, imgDeleter: nil}
}

// ✅ NEW: 2種類の Query を注入できる ctor
func NewListHandlerWithQueries(
	uc *usecase.ListUsecase,
	qMgmt *query.ListManagementQuery,
	qDetail *query.ListDetailQuery,
) http.Handler {
	return &ListHandler{uc: uc, qMgmt: qMgmt, qDetail: qDetail, imgUploader: nil, imgDeleter: nil}
}

// ✅ NEW: ListImage も注入できる ctor（既存呼び出しを壊さない）
func NewListHandlerWithQueriesAndListImage(
	uc *usecase.ListUsecase,
	qMgmt *query.ListManagementQuery,
	qDetail *query.ListDetailQuery,
	uploader ListImageUploader,
	deleter ListImageDeleter,
) http.Handler {
	return &ListHandler{
		uc:          uc,
		qMgmt:       qMgmt,
		qDetail:     qDetail,
		imgUploader: uploader,
		imgDeleter:  deleter,
	}
}

// ✅ backward-ish: 片方だけ注入したい場合
func NewListHandlerWithManagementQuery(uc *usecase.ListUsecase, q *query.ListManagementQuery) http.Handler {
	return &ListHandler{uc: uc, qMgmt: q, qDetail: nil, imgUploader: nil, imgDeleter: nil}
}
func NewListHandlerWithDetailQuery(uc *usecase.ListUsecase, q *query.ListDetailQuery) http.Handler {
	return &ListHandler{uc: uc, qMgmt: nil, qDetail: q, imgUploader: nil, imgDeleter: nil}
}

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

	// GET /lists/create-seed?inventoryId={pbId}__{tbId}&modelIds=a,b,c
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

			// /lists/{id}/images/{sub}
			sub := ""
			if len(parts) >= 3 {
				sub = strings.TrimSpace(parts[2])
			}

			// ✅ NEW: /lists/{id}/images/signed-url で signed URL を発行
			if strings.EqualFold(sub, "signed-url") {
				if r.Method != http.MethodPost {
					methodNotAllowed(w)
					return
				}
				h.issueSignedURL(w, r, id)
				return
			}

			// ✅ NEW: /lists/{id}/images/upload（dataURL 直アップロード） ※旧方式互換
			if strings.EqualFold(sub, "upload") {
				if r.Method != http.MethodPost {
					methodNotAllowed(w)
					return
				}
				h.uploadImage(w, r, id)
				return
			}

			// ✅ NEW: /lists/{id}/images/{imageId} DELETE
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

// ==============================
// POST /lists/{id}/images/signed-url
// ==============================
//
// ✅ フロントが `signed_url_response_invalid` を出さないように、
//
//	必須キーを handler 側で "固定の形" に正規化して返す。
//	- uploadUrl / bucket / objectPath / id / publicUrl を必ず返す（可能な限り）
func (h *ListHandler) issueSignedURL(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	var req struct {
		FileName         string `json:"fileName"`
		ContentType      string `json:"contentType"`
		Size             int64  `json:"size"`
		DisplayOrder     int    `json:"displayOrder"`
		ExpiresInSeconds int    `json:"expiresInSeconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	ct := strings.ToLower(strings.TrimSpace(req.ContentType))
	if ct == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "contentType is required"})
		return
	}

	// ✅ map[string]struct{} の value(struct{})を TrimSpace しない（以前のコンパイルエラー対策）
	if _, ok := listimgdom.SupportedImageMIMEs[ct]; !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unsupported contentType"})
		return
	}

	if req.Size > 0 && req.Size > int64(listimgdom.DefaultMaxImageSizeBytes) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file too large"})
		return
	}

	rawOut, err := h.uc.IssueImageSignedURL(ctx, usecase.ListImageIssueSignedURLInput{
		ListID:           listID,
		FileName:         strings.TrimSpace(req.FileName),
		ContentType:      ct,
		Size:             req.Size,
		DisplayOrder:     req.DisplayOrder,
		ExpiresInSeconds: req.ExpiresInSeconds,
	})
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	// ------------------------------------------------------------
	// ✅ 正規化（usecase の返却形が多少ズレてもフロントが壊れないようにする）
	// ------------------------------------------------------------
	bs, _ := json.Marshal(rawOut)
	var m map[string]any
	_ = json.Unmarshal(bs, &m)

	getString := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k]; ok {
				switch t := v.(type) {
				case string:
					if s := strings.TrimSpace(t); s != "" {
						return s
					}
				}
			}
		}
		return ""
	}
	getInt64 := func(keys ...string) int64 {
		for _, k := range keys {
			if v, ok := m[k]; ok {
				switch t := v.(type) {
				case float64:
					return int64(t)
				case int64:
					return t
				case int:
					return int64(t)
				case string:
					s := strings.TrimSpace(t)
					if s == "" {
						continue
					}
					if n, e := strconv.ParseInt(s, 10, 64); e == nil {
						return n
					}
				}
			}
		}
		return 0
	}
	getInt := func(keys ...string) int {
		for _, k := range keys {
			if v, ok := m[k]; ok {
				switch t := v.(type) {
				case float64:
					return int(t)
				case int:
					return t
				case int64:
					return int(t)
				case string:
					s := strings.TrimSpace(t)
					if s == "" {
						continue
					}
					if n, e := strconv.Atoi(s); e == nil {
						return n
					}
				}
			}
		}
		return 0
	}

	// 互換候補キーを広めに吸収
	id := getString("id", "imageId", "imageID")
	bucket := getString("bucket")
	objectPath := getString("objectPath", "object_path", "object", "path")
	publicURL := getString("publicUrl", "publicURL", "public_url", "url")
	uploadURL := getString("uploadUrl", "uploadURL", "signedUrl", "signedURL", "signed_url", "upload_url", "putUrl", "putURL")
	expiresAt := getString("expiresAt", "expires_at", "expireAt", "expire_at")
	fileName := getString("fileName", "filename", "file_name")
	contentType := getString("contentType", "content_type", "mime")

	// 値の補完
	if strings.TrimSpace(fileName) == "" {
		fileName = strings.TrimSpace(req.FileName)
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = ct
	}
	if strings.TrimSpace(publicURL) == "" && strings.TrimSpace(bucket) != "" && strings.TrimSpace(objectPath) != "" {
		publicURL = fmt.Sprintf("https://storage.googleapis.com/%s/%s", strings.TrimSpace(bucket), strings.TrimLeft(strings.TrimSpace(objectPath), "/"))
	}
	if strings.TrimSpace(id) == "" {
		// objectPath が {listId}/{imageId}/{fileName} の想定なら imageId を抜く
		p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
		pp := strings.Split(p, "/")
		if len(pp) >= 3 && strings.TrimSpace(pp[1]) != "" {
			id = strings.TrimSpace(pp[1])
		} else if p != "" {
			// 最終フォールバック（フロントが "id が必須" の場合に落ちないように）
			id = p
		}
	}

	size := getInt64("size")
	if size <= 0 {
		size = req.Size
	}
	displayOrder := getInt("displayOrder", "display_order", "order")
	if displayOrder == 0 {
		displayOrder = req.DisplayOrder
	}

	// expiresAt が無い場合は計算して埋める（フロントが必須扱いの場合の保険）
	if strings.TrimSpace(expiresAt) == "" {
		sec := req.ExpiresInSeconds
		if sec <= 0 {
			sec = 15 * 60
		}
		expiresAt = time.Now().UTC().Add(time.Duration(sec) * time.Second).Format(time.RFC3339)
	}

	// ✅ フロントが "uploadUrl が無い" と invalid 扱いするケースが多いので、ここで弾く
	if strings.TrimSpace(uploadURL) == "" || strings.TrimSpace(bucket) == "" || strings.TrimSpace(objectPath) == "" || strings.TrimSpace(id) == "" {
		// 期待形にできない＝フロントが invalid になるので 500 で明示
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "signed_url_response_invalid"})
		return
	}

	type resp struct {
		ID           string `json:"id"`
		Bucket       string `json:"bucket"`
		ObjectPath   string `json:"objectPath"`
		PublicURL    string `json:"publicUrl"`
		UploadURL    string `json:"uploadUrl"`
		ExpiresAt    string `json:"expiresAt"`
		ContentType  string `json:"contentType"`
		Size         int64  `json:"size"`
		DisplayOrder int    `json:"displayOrder"`
		FileName     string `json:"fileName"`
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp{
		ID:           strings.TrimSpace(id),
		Bucket:       strings.TrimSpace(bucket),
		ObjectPath:   strings.TrimLeft(strings.TrimSpace(objectPath), "/"),
		PublicURL:    strings.TrimSpace(publicURL),
		UploadURL:    strings.TrimSpace(uploadURL),
		ExpiresAt:    strings.TrimSpace(expiresAt),
		ContentType:  strings.TrimSpace(contentType),
		Size:         size,
		DisplayOrder: displayOrder,
		FileName:     strings.TrimSpace(fileName),
	})
}

// ==============================
// GET /lists/create-seed
// ==============================

func (h *ListHandler) createSeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "handler is nil"})
		return
	}

	// ✅ management query が無いと seed を作れない
	if h.qMgmt == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	qp := r.URL.Query()

	invID := strings.TrimSpace(qp.Get("inventoryId"))
	if invID == "" {
		invID = strings.TrimSpace(qp.Get("inventory_id"))
	}
	if invID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "inventoryId is required"})
		return
	}

	// modelIds:
	// - modelIds=a&modelIds=b
	// - modelIds=a,b,c
	modelIDs := []string{}
	if vv := qp["modelIds"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x == "" {
				continue
			}
			for _, s := range splitCSV(x) {
				s = strings.TrimSpace(s)
				if s != "" {
					modelIDs = append(modelIDs, s)
				}
			}
		}
	} else if vv := qp["model_ids"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x == "" {
				continue
			}
			for _, s := range splitCSV(x) {
				s = strings.TrimSpace(s)
				if s != "" {
					modelIDs = append(modelIDs, s)
				}
			}
		}
	} else {
		raw := strings.TrimSpace(qp.Get("modelIds"))
		if raw == "" {
			raw = strings.TrimSpace(qp.Get("model_ids"))
		}
		if raw != "" {
			for _, s := range splitCSV(raw) {
				s = strings.TrimSpace(s)
				if s != "" {
					modelIDs = append(modelIDs, s)
				}
			}
		}
	}

	out, err := h.qMgmt.BuildCreateSeed(ctx, invID, modelIDs)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}

		msg := strings.TrimSpace(err.Error())
		if strings.Contains(strings.ToLower(msg), "invalid") || strings.Contains(strings.ToLower(msg), "inventory") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
			return
		}

		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ==============================
// GET /lists
// ==============================

func (h *ListHandler) listIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	qp := r.URL.Query()

	// ---- filter ----
	var f listdom.Filter

	if s := strings.TrimSpace(qp.Get("q")); s != "" {
		f.SearchQuery = s
	} else if s := strings.TrimSpace(qp.Get("search")); s != "" {
		f.SearchQuery = s
	}

	if v := strings.TrimSpace(qp.Get("assigneeId")); v != "" {
		f.AssigneeID = &v
	} else if v := strings.TrimSpace(qp.Get("assignee_id")); v != "" {
		f.AssigneeID = &v
	}

	statusesRaw := strings.TrimSpace(qp.Get("statuses"))
	if statusesRaw == "" {
		statusesRaw = strings.TrimSpace(qp.Get("status"))
	}
	if statusesRaw != "" {
		ss := splitCSV(statusesRaw)
		if len(ss) == 1 {
			st := listdom.ListStatus(strings.TrimSpace(ss[0]))
			if st != "" {
				f.Status = &st
			}
		} else if len(ss) > 1 {
			out := make([]listdom.ListStatus, 0, len(ss))
			for _, s := range ss {
				st := listdom.ListStatus(strings.TrimSpace(s))
				if st != "" {
					out = append(out, st)
				}
			}
			f.Statuses = out
		}
	}

	if dv := strings.TrimSpace(qp.Get("deleted")); dv != "" {
		if b, err := strconv.ParseBool(dv); err == nil {
			f.Deleted = &b
		}
	}

	if v := strings.TrimSpace(qp.Get("minPrice")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MinPrice = &n
		}
	}
	if v := strings.TrimSpace(qp.Get("maxPrice")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MaxPrice = &n
		}
	}

	// ✅ modelNumbers 互換は廃止。modelIds のみ採用。
	if vv := qp["modelIds"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x != "" {
				f.ModelNumbers = append(f.ModelNumbers, x)
			}
		}
	} else if vv := qp["model_ids"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x != "" {
				f.ModelNumbers = append(f.ModelNumbers, x)
			}
		}
	}

	if t := parseRFC3339Ptr(qp.Get("createdFrom")); t != nil {
		f.CreatedFrom = t
	}
	if t := parseRFC3339Ptr(qp.Get("createdTo")); t != nil {
		f.CreatedTo = t
	}
	if t := parseRFC3339Ptr(qp.Get("updatedFrom")); t != nil {
		f.UpdatedFrom = t
	}
	if t := parseRFC3339Ptr(qp.Get("updatedTo")); t != nil {
		f.UpdatedTo = t
	}
	if t := parseRFC3339Ptr(qp.Get("deletedFrom")); t != nil {
		f.DeletedFrom = t
	}
	if t := parseRFC3339Ptr(qp.Get("deletedTo")); t != nil {
		f.DeletedTo = t
	}

	// ---- sort ----
	sort := listdom.Sort{} // repo側のデフォルトに任せる

	// ---- page ----
	pageNum := parseIntDefault(qp.Get("page"), 1)
	perPage := parseIntDefault(qp.Get("perPage"), 50)
	page := listdom.Page{Number: pageNum, PerPage: perPage}

	// ✅ management query があれば query 経由で返す
	if h.qMgmt != nil {
		pr, err := h.qMgmt.ListRows(ctx, f, sort, page)
		if err != nil {
			if isNotSupported(err) {
				w.WriteHeader(http.StatusNotImplemented)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
				return
			}
			writeListErr(w, err)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"items":      pr.Items,
			"totalCount": pr.TotalCount,
			"totalPages": pr.TotalPages,
			"page":       pr.Page,
			"perPage":    pr.PerPage,
		})
		return
	}

	// fallback: usecase のまま返す
	result, err := h.uc.List(ctx, f, sort, page)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":      result.Items,
		"totalCount": result.TotalCount,
		"totalPages": result.TotalPages,
		"page":       result.Page,
		"perPage":    perPage,
	})
}

// ==============================
// POST /lists
// ==============================

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

	item.ID = strings.TrimSpace(item.ID)
	item.Title = strings.TrimSpace(item.Title)
	item.AssigneeID = strings.TrimSpace(item.AssigneeID)
	item.InventoryID = strings.TrimSpace(item.InventoryID)
	item.ImageID = strings.TrimSpace(item.ImageID)
	item.Description = strings.TrimSpace(item.Description)
	item.CreatedBy = strings.TrimSpace(item.CreatedBy)

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

	// ✅ listId は inventoryId と独立したIDを使う（事故防止）
	if item.ID == item.InventoryID {
		item.ID = ""
	}

	if strings.TrimSpace(string(item.Status)) == "" {
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

// ==============================
// PUT|PATCH /lists/{id}
// ==============================

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

	log.Printf("[list_handler] UPDATE received method=%s id=%q bytes=%d", r.Method, strings.TrimSpace(id), len(body))

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

	if t := strings.TrimSpace(in.Title); t != "" {
		current.Title = t
	}
	if d := strings.TrimSpace(in.Description); d != "" {
		current.Description = d
	}

	if st := strings.TrimSpace(string(in.Status)); st != "" {
		current.Status = listdom.ListStatus(st)
	}

	if in.Prices != nil {
		current.Prices = in.Prices
	}

	if in.UpdatedBy != nil {
		current.UpdatedBy = normalizeStrPtr(in.UpdatedBy)
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

// ==============================
// GET /lists/{id}
// ==============================

func (h *ListHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.qDetail == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	dto, err := h.qDetail.BuildListDetailDTO(ctx, id)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(dto)
}

// GET /lists/{id}/images
func (h *ListHandler) listImages(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	items, err := h.uc.GetImages(ctx, id)
	if err != nil {
		writeListErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}

// ✅ 旧方式互換: POST /lists/{id}/images/upload
func (h *ListHandler) uploadImage(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}
	if h.imgUploader == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	var req struct {
		ImageData    string  `json:"imageData"`
		FileName     string  `json:"fileName"`
		SetAsPrimary bool    `json:"setAsPrimary"`
		DisplayOrder *int    `json:"displayOrder"`
		CreatedBy    string  `json:"createdBy"`
		UpdatedBy    *string `json:"updatedBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "createdBy is required"})
		return
	}

	in := listimgdom.UploadImageInput{
		ImageData: strings.TrimSpace(req.ImageData),
		FileName:  strings.TrimSpace(req.FileName),
		ListID:    listID,
	}

	img, err := h.imgUploader.Upload(ctx, in)
	if err != nil {
		writeListErr(w, err)
		return
	}
	if img == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "upload returned nil"})
		return
	}

	if req.SetAsPrimary {
		now := time.Now().UTC()
		_, e := h.uc.SetPrimaryImage(ctx, listID, strings.TrimSpace(img.ID), now, normalizeStrPtr(req.UpdatedBy))
		if e != nil {
			log.Printf("[list_handler] WARN: SetPrimaryImage failed listID=%s imageID=%s err=%v", listID, strings.TrimSpace(img.ID), e)
		}
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(img)
}

// ✅ NEW: DELETE /lists/{id}/images/{imageId}
func (h *ListHandler) deleteImage(w http.ResponseWriter, r *http.Request, listID string, imageID string) {
	ctx := r.Context()

	if h == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "handler is nil"})
		return
	}
	if h.imgDeleter == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "imageId is required"})
		return
	}

	if err := h.imgDeleter.Delete(ctx, imageID); err != nil {
		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"listId":  strings.TrimSpace(listID),
		"imageId": imageID,
	})
}

// POST /lists/{id}/images
func (h *ListHandler) saveImageFromGCS(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	var req struct {
		ID           string  `json:"id"`
		FileName     string  `json:"fileName"`
		Bucket       string  `json:"bucket"`
		ObjectPath   string  `json:"objectPath"`
		Size         int64   `json:"size"`
		DisplayOrder int     `json:"displayOrder"`
		CreatedBy    string  `json:"createdBy"`
		CreatedAt    *string `json:"createdAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if strings.TrimSpace(req.ID) == "" || strings.TrimSpace(req.ObjectPath) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id and objectPath are required"})
		return
	}

	ca := time.Now().UTC()
	if req.CreatedAt != nil && strings.TrimSpace(*req.CreatedAt) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.CreatedAt)); err == nil {
			ca = t.UTC()
		}
	}

	img, err := h.uc.SaveImageFromGCS(
		ctx,
		strings.TrimSpace(req.ID),
		strings.TrimSpace(listID),
		strings.TrimSpace(req.Bucket),
		strings.TrimSpace(req.ObjectPath),
		req.Size,
		req.DisplayOrder,
		strings.TrimSpace(req.CreatedBy),
		ca,
	)
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
	_ = json.NewEncoder(w).Encode(img)
}

// PUT|POST|PATCH /lists/{id}/primary-image
func (h *ListHandler) setPrimaryImage(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	var req struct {
		ImageID   string  `json:"imageId"`
		UpdatedBy *string `json:"updatedBy"`
		Now       *string `json:"now"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}
	imageID := strings.TrimSpace(req.ImageID)
	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "imageId is required"})
		return
	}

	now := time.Now().UTC()
	if req.Now != nil && strings.TrimSpace(*req.Now) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.Now)); err == nil {
			now = t.UTC()
		}
	}

	item, err := h.uc.SetPrimaryImage(ctx, listID, imageID, now, normalizeStrPtr(req.UpdatedBy))
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(item)
}

// GET /lists/{id}/aggregate
func (h *ListHandler) getAggregate(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	agg, err := h.uc.GetAggregate(ctx, id)
	if err != nil {
		writeListErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(agg)
}

// ==============================
// error helpers
// ==============================

func writeListErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, listdom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, listdom.ErrConflict):
		code = http.StatusConflict
	default:
		msg := strings.ToLower(strings.TrimSpace(err.Error()))
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func normalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}
