// backend/internal/adapters/in/http/console/handler/brand_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"time"

	shared "narratives/internal/adapters/in/http/shared"
	query "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
)

type BrandHandler struct {
	uc              *usecase.BrandUsecase
	managementQuery *query.BrandManagementQuery
	detailQuery     *query.BrandDetailQuery
}

func NewBrandHandler(
	uc *usecase.BrandUsecase,
	managementQuery *query.BrandManagementQuery,
	detailQuery *query.BrandDetailQuery,
) http.Handler {
	return &BrandHandler{
		uc:              uc,
		managementQuery: managementQuery,
		detailQuery:     detailQuery,
	}
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

	case r.Method == http.MethodDelete && len(path) > len("/brands/") && path[:len("/brands/")] == "/brands/":
		id := path[len("/brands/"):]
		h.delete(w, r, id)

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

	result, err := h.detailQuery.GetByID(ctx, vid)
	if err != nil {
		writeBrandErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(toBrandDTO(result.Brand, result.MemberName))
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

	w.WriteHeader(http.StatusCreated)

	result, err := h.detailQuery.GetByID(ctx, created.ID)
	if err == nil {
		_ = json.NewEncoder(w).Encode(toBrandDTO(result.Brand, result.MemberName))
		return
	}

	_ = json.NewEncoder(w).Encode(toBrandDTO(created, ""))
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

	patch := branddom.BrandPatch{
		ManagerID:            in.ManagerID,
		Name:                 in.Name,
		Description:          in.Description,
		URL:                  in.WebsiteURL,
		BrandIcon:            in.BrandIcon,
		BrandBackgroundImage: in.BrandBackgroundImage,
		IsActive:             in.IsActive,
	}

	updated, err := h.uc.Update(ctx, vid, patch)
	if err != nil {
		writeBrandErr(w, err)
		return
	}

	result, err := h.detailQuery.GetByID(ctx, updated.ID)
	if err == nil {
		_ = json.NewEncoder(w).Encode(toBrandDTO(result.Brand, result.MemberName))
		return
	}

	_ = json.NewEncoder(w).Encode(toBrandDTO(updated, ""))
}

func (h *BrandHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	vid, err := shared.StrictID(id, "id")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(r.Context(), vid); err != nil {
		writeBrandErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BrandHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

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

	p := branddom.Page{
		Number:  pageNum,
		PerPage: perPage,
	}

	companyID := usecase.CompanyIDFromContext(ctx)

	result, err := h.managementQuery.ListByCompanyID(ctx, companyID, p)
	if err != nil {
		writeBrandErr(w, err)
		return
	}

	dtoItems := make([]brandDTO, 0, len(result.Items))
	for _, item := range result.Items {
		dtoItems = append(dtoItems, toBrandDTO(item.Brand, item.MemberName))
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
