// backend/internal/application/usecase/report_admin_read.go
package usecase

import (
	"context"

	common "narratives/internal/domain/common"
	reportdom "narratives/internal/domain/report"
)

func (u *ReportUsecase) ListReportCases(
	ctx context.Context,
	filter reportdom.CaseFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[reportdom.ReportCase], error) {
	if err := u.ensureReportRepository(); err != nil {
		return common.PageResult[reportdom.ReportCase]{}, err
	}
	return u.reportRepo.ListCases(ctx, filter, sort, page)
}

func (u *ReportUsecase) GetReportCase(
	ctx context.Context,
	caseID reportdom.CaseID,
) (reportdom.ReportCase, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.ReportCase{}, err
	}
	if caseID == "" {
		return reportdom.ReportCase{}, reportdom.ErrInvalidCaseID
	}
	return u.reportRepo.GetCase(ctx, caseID)
}

func (u *ReportUsecase) ListReports(
	ctx context.Context,
	caseID reportdom.CaseID,
	filter reportdom.ReportFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[reportdom.Report], error) {
	if err := u.ensureReportRepository(); err != nil {
		return common.PageResult[reportdom.Report]{}, err
	}
	if caseID == "" {
		return common.PageResult[reportdom.Report]{}, reportdom.ErrInvalidCaseID
	}
	return u.reportRepo.ListReports(ctx, caseID, filter, sort, page)
}
