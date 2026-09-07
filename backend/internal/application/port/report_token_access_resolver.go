// backend/internal/application/port/report_token_access_resolver.go
package port

import "context"

// ReviewReportTokenAccessResolver determines whether an avatar is allowed to
// report a comment under the specified token blueprint.
//
// Authorization details such as purchase / ownership resolution belong to the
// implementation. Callers only depend on this reporting-access decision.
type ReportTokenAccessResolver interface {
	CanReportTokenBlueprintComment(
		ctx context.Context,
		avatarID string,
		tokenBlueprintID string,
	) (bool, error)
}
