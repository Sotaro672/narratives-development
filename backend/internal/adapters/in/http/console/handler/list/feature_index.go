// backend/internal/adapters/in/http/console/handler/list/feature_index.go
//
// Responsibility:
// - GET /lists（一覧取得）を担当する。
// - Query があれば read-model で返し、無ければ usecase をフォールバックする。
package list

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	listdom "narratives/internal/domain/list"
)

func (h *ListHandler) listIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	qp := r.URL.Query()

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

	sort := listdom.Sort{} // repo側のデフォルトに任せる

	pageNum := parseIntDefault(qp.Get("page"), 1)
	perPage := parseIntDefault(qp.Get("perPage"), 50)
	page := listdom.Page{Number: pageNum, PerPage: perPage}

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
