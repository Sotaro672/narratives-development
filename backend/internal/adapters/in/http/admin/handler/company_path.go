// backend/internal/adapters/in/http/admin/handler/company_path.go
package handler

import "strings"

func parseContractDetailCompanyID(
	path string,
) (string, bool) {
	prefix := adminCompaniesPath + "/"
	if !strings.HasPrefix(path, prefix) ||
		!strings.HasSuffix(
			path,
			adminContractDetailPathSuffix,
		) {
		return "", false
	}

	companyID := strings.TrimSuffix(
		strings.TrimPrefix(path, prefix),
		adminContractDetailPathSuffix,
	)
	companyID = strings.TrimSuffix(companyID, "/")
	companyID = strings.TrimSpace(companyID)

	if companyID == "" ||
		strings.Contains(companyID, "/") {
		return "", false
	}

	return companyID, true
}

func parseContractResourcePath(
	path string,
	resource string,
) (
	companyID string,
	resourceID string,
	ok bool,
) {
	prefix := adminCompaniesPath + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	relativePath := strings.TrimPrefix(
		path,
		prefix,
	)
	parts := strings.Split(
		relativePath,
		"/",
	)

	if len(parts) != 3 ||
		parts[1] != resource {
		return "", "", false
	}

	companyID = strings.TrimSpace(parts[0])
	resourceID = strings.TrimSpace(parts[2])

	if companyID == "" ||
		resourceID == "" {
		return "", "", false
	}

	return companyID, resourceID, true
}

func parseContractReviewResourcePath(
	path string,
	resource string,
) (
	companyID string,
	resourceID string,
	ok bool,
) {
	prefix := adminCompaniesPath + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	relativePath := strings.TrimPrefix(
		path,
		prefix,
	)
	parts := strings.Split(
		relativePath,
		"/",
	)

	if len(parts) != 4 ||
		parts[1] != resource ||
		parts[3] != "reviews" {
		return "", "", false
	}

	companyID = strings.TrimSpace(parts[0])
	resourceID = strings.TrimSpace(parts[2])

	if companyID == "" ||
		resourceID == "" {
		return "", "", false
	}

	return companyID, resourceID, true
}
