// backend/internal/application/port/company_context.go

package port

import "context"

type CompanyIDResolver func(
	ctx context.Context,
) string
