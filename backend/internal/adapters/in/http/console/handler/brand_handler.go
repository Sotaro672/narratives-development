// backend/internal/adapters/in/http/console/handler/brand_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"time"

	shared "narratives/internal/adapters/in/http/shared"
	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
)

type BrandHandler struct {
	uc *usecase.BrandUsecase
}

func NewBrandHandler(uc *usecase.BrandUsecase) http.Handler {
	return &BrandHandler{uc: uc}
}

func (h *BrandHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if shared.RedirectTrailingSlash(w, r) {
		return
	}

	path := r.URL.Path

	switch {
	case r.Method == http.MethodGet && path == "/brands":
		h.list(w, r)

	case r.Method == http.MethodPost && path == "/brands":
		h.create(w, r)

	case r.Method == http.MethodGet && len(path) > len("/brands/") && path[:len("/brands/")] == "/brands/":
		id := path[len("/brands/"):]
		h.get(w, r, id)

	case r.Method == http.MethodPatch && len(path) > len("/brands/") && path[:len("/brands/")] == "/brands/":
		id := path[len("/brands/"):]
		h.update(w, r, id)

	case r.Method == http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

type brandDTO struct {
	ID                   string     `json:"id"`
	CompanyID            string     `json:"companyId"`
	Name                 string     `json:"name"`
	Description          string     `json:"description"`
	URL                  string     `json:"websiteUrl,omitempty"`
	BrandIcon            string     `json:"brandIcon,omitempty"`
	BrandBackgroundImage string     `json:"brandBackgroundImage,omitempty"`
	IsActive             bool       `json:"isActive"`
	ManagerID            *string    `json:"managerId,omitempty"`
	MemberName           string     `json:"memberName"`
	WalletAddress        string     `json:"walletAddress"`
	CreatedAt            time.Time  `json:"createdAt"`
	CreatedBy            *string    `json:"createdBy,omitempty"`
	UpdatedAt            *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy            *string    `json:"updatedBy,omitempty"`
	DeletedAt            *time.Time `json:"deletedAt,omitempty"`
	DeletedBy            *string    `json:"deletedBy,omitempty"`
}

func toBrandDTO(b branddom.Brand, memberName string) brandDTO {
	return brandDTO{
		ID:                   b.ID,
		CompanyID:            b.CompanyID,
		Name:                 b.Name,
		Description:          b.Description,
		URL:                  b.URL,
		BrandIcon:            b.BrandIcon,
		BrandBackgroundImage: b.BrandBackgroundImage,
		IsActive:             b.IsActive,
		ManagerID:            b.ManagerID,
		MemberName:           memberName,
		WalletAddress:        b.WalletAddress,
		CreatedAt:            b.CreatedAt,
		CreatedBy:            b.CreatedBy,
		UpdatedAt:            b.UpdatedAt,
		UpdatedBy:            b.UpdatedBy,
		DeletedAt:            b.DeletedAt,
		DeletedBy:            b.DeletedBy,
	}
}

func (h *BrandHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	vid, err := shared.StrictID(id, "id")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	brand, err := h.uc.GetByID(ctx, vid)
	if err != nil {
		writeBrandErr(w, err)
		return
	}

	memberName := ""
	if brand.ManagerID != nil && *brand.ManagerID != "" {
		if name, nerr := h.uc.ResolveMemberNameByID(ctx, *brand.ManagerID); nerr == nil {
			memberName = name
		}
	}

	_ = json.NewEncoder(w).Encode(toBrandDTO(brand, memberName))
}

func (h *BrandHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var in struct {
		CompanyID            string  `json:"companyId"`
		Name                 string  `json:"name"`
		Description          string  `json:"description"`
		WebsiteURL           string  `json:"websiteUrl"`
		BrandIcon            string  `json:"brandIcon"`
		BrandBackgroundImage string  `json:"brandBackgroundImage"`
		IsActive             *bool   `json:"isActive"`
		ManagerID            *string `json:"managerId"`
		CreatedBy            *string `json:"createdBy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	companyID := in.CompanyID
	if companyID != "" {
		v, err := shared.StrictRequired(companyID, "companyId")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid companyId"})
			return
		}
		companyID = v
	}

	name, err := shared.StrictRequired(in.Name, "name")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "name is required"})
		return
	}

	description := in.Description
	if description != "" {
		if shared.HasOuterWhitespace(description) || shared.HasControlWhitespace(description) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "description must not have leading/trailing whitespace or tab/newline"})
			return
		}
	}

	websiteURL := in.WebsiteURL
	if websiteURL != "" {
		if shared.HasOuterWhitespace(websiteURL) || shared.HasControlWhitespace(websiteURL) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "websiteUrl must not have leading/trailing whitespace or tab/newline"})
			return
		}
	}

	brandIcon := in.BrandIcon
	if brandIcon != "" {
		if shared.HasOuterWhitespace(brandIcon) || shared.HasControlWhitespace(brandIcon) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "brandIcon must not have leading/trailing whitespace or tab/newline"})
			return
		}
	}

	brandBackgroundImage := in.BrandBackgroundImage
	if brandBackgroundImage != "" {
		if shared.HasOuterWhitespace(brandBackgroundImage) || shared.HasControlWhitespace(brandBackgroundImage) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "brandBackgroundImage must not have leading/trailing whitespace or tab/newline"})
			return
		}
	}

	var managerID *string
	if in.ManagerID != nil {
		v, err := shared.StrictRequired(*in.ManagerID, "managerId")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid managerId"})
			return
		}
		managerID = &v
	}

	var createdBy *string
	if in.CreatedBy != nil {
		v, err := shared.StrictRequired(*in.CreatedBy, "createdBy")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid createdBy"})
			return
		}
		createdBy = &v
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	now := time.Now().UTC()
	b, err := branddom.New(
		"",
		companyID,
		name,
		description,
		"",
		websiteURL,
		brandIcon,
		brandBackgroundImage,
		isActive,
		managerID,
		createdBy,
		now,
	)
	if err != nil {
		writeBrandErr(w, err)
		return
	}

	created, err := h.uc.Create(ctx, b)
	if err != nil {
		writeBrandErr(w, err)
		return
	}

	memberName := ""
	if created.ManagerID != nil && *created.ManagerID != "" {
		if name, nerr := h.uc.ResolveMemberNameByID(ctx, *created.ManagerID); nerr == nil {
			memberName = name
		}
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toBrandDTO(created, memberName))
}

func (h *BrandHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	vid, err := shared.StrictID(id, "id")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var in struct {
		Name                 *string `json:"name"`
		Description          *string `json:"description"`
		WebsiteURL           *string `json:"websiteUrl"`
		BrandIcon            *string `json:"brandIcon"`
		BrandBackgroundImage *string `json:"brandBackgroundImage"`
		IsActive             *bool   `json:"isActive"`
		ManagerID            *string `json:"managerId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if in.Name != nil {
		if _, err := shared.StrictRequired(*in.Name, "name"); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "name is required"})
			return
		}
		if shared.HasOuterWhitespace(*in.Name) || shared.HasControlWhitespace(*in.Name) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "name must not have leading/trailing whitespace or tab/newline"})
			return
		}
	}

	if in.Description != nil && *in.Description != "" {
		if shared.HasOuterWhitespace(*in.Description) || shared.HasControlWhitespace(*in.Description) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "description must not have leading/trailing whitespace or tab/newline"})
			return
		}
	}

	if in.WebsiteURL != nil && *in.WebsiteURL != "" {
		if shared.HasOuterWhitespace(*in.WebsiteURL) || shared.HasControlWhitespace(*in.WebsiteURL) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "websiteUrl must not have leading/trailing whitespace or tab/newline"})
			return
		}
	}

	if in.BrandIcon != nil && *in.BrandIcon != "" {
		if shared.HasOuterWhitespace(*in.BrandIcon) || shared.HasControlWhitespace(*in.BrandIcon) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "brandIcon must not have leading/trailing whitespace or tab/newline"})
			return
		}
	}

	if in.BrandBackgroundImage != nil && *in.BrandBackgroundImage != "" {
		if shared.HasOuterWhitespace(*in.BrandBackgroundImage) || shared.HasControlWhitespace(*in.BrandBackgroundImage) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "brandBackgroundImage must not have leading/trailing whitespace or tab/newline"})
			return
		}
	}

	if in.ManagerID != nil {
		v, err := shared.StrictRequired(*in.ManagerID, "managerId")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid managerId"})
			return
		}
		in.ManagerID = &v
	}

	updated, err := h.uc.UpdateBrand(
		ctx,
		vid,
		in.ManagerID,
		in.Name,
		in.Description,
		in.WebsiteURL,
		in.BrandIcon,
		in.BrandBackgroundImage,
		in.IsActive,
	)
	if err != nil {
		writeBrandErr(w, err)
		return
	}

	memberName := ""
	if updated.ManagerID != nil && *updated.ManagerID != "" {
		if name, nerr := h.uc.ResolveMemberNameByID(ctx, *updated.ManagerID); nerr == nil {
			memberName = name
		}
	}

	_ = json.NewEncoder(w).Encode(toBrandDTO(updated, memberName))
}

func (h *BrandHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	var f branddom.Filter

	if raw := q.Get("managerId"); raw != "" {
		v, err := shared.StrictRequired(raw, "managerId")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid managerId"})
			return
		}
		f.ManagerID = &v
	}

	if raw := q.Get("walletAddress"); raw != "" {
		v, err := shared.StrictRequired(raw, "walletAddress")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid walletAddress"})
			return
		}
		f.WalletAddress = &v
	}

	if raw := q.Get("isActive"); raw != "" {
		b, ok, err := shared.StrictBoolParam(raw, "isActive")
		if err != nil || !ok {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid isActive"})
			return
		}
		f.IsActive = &b
	}

	if raw := q.Get("q"); raw != "" {
		v, err := shared.StrictRequired(raw, "q")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid q"})
			return
		}
		f.FilterCommon.SearchQuery = v
	}

	pageNum, err := shared.StrictPositiveIntParam(q.Get("page"), "page", 1)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid page"})
		return
	}

	perPage, err := shared.StrictPositiveIntParam(q.Get("perPage"), "perPage", 50)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid perPage"})
		return
	}

	p := branddom.Page{Number: pageNum, PerPage: perPage}

	result, err := h.uc.List(ctx, f, p)
	if err != nil {
		writeBrandErr(w, err)
		return
	}

	dtoItems := make([]brandDTO, 0, len(result.Items))
	for _, b := range result.Items {
		memberName := ""
		if b.ManagerID != nil && *b.ManagerID != "" {
			if name, nerr := h.uc.ResolveMemberNameByID(ctx, *b.ManagerID); nerr == nil {
				memberName = name
			}
		}
		dtoItems = append(dtoItems, toBrandDTO(b, memberName))
	}

	out := struct {
		Items      []brandDTO `json:"items"`
		TotalCount int        `json:"totalCount"`
		Page       int        `json:"page"`
		PerPage    int        `json:"perPage"`
		TotalPages int        `json:"totalPages"`
	}{
		Items:      dtoItems,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	}

	_ = json.NewEncoder(w).Encode(out)
}

func writeBrandErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case branddom.ErrInvalidID:
		code = http.StatusBadRequest
	case branddom.ErrNotFound:
		code = http.StatusNotFound
	case branddom.ErrConflict:
		code = http.StatusConflict
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
