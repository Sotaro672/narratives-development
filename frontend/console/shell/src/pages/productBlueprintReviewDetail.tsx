// frontend/console/shell/src/pages/productBlueprintReviewDetail.tsx

import { useMemo, useState } from "react";
import { useLocation } from "react-router-dom";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import LogCard from "../features/log/presentation/LogCard";
import ReviewReportModal from "../features/reviewReport/presentation/components/ReviewReportModal";

import Pagination from "../shared/ui/pagination";
import RefreshButton from "../shared/ui/refresh";
import { Button } from "../shared/ui/button";

import {
  ratingToStars,
  statusLabelJa,
} from "../features/productBlueprintReview/presentation/component/review";

import type { ReviewStatus } from "../shared/types/productBlueprintReview";

import { useProductBlueprintReviewDetail } from "../features/productBlueprintReview/presentation/hook/useProductBlueprintReviewDetail";

import "../styles/productBlueprintReview.css";

type DetailNavState = {
  ProductName?: string;
  AssigneeName?: string;
};

type SortKey = "Rating" | "ReviewedAt" | null;
type SortDir = "asc" | "desc";

export default function ProductBlueprintReviewDetail() {
  const Location = useLocation();
  const State = (Location.state ?? {}) as DetailNavState;

  const HeaderProductName = String(State.ProductName ?? "");
  const HeaderAssigneeName = String(State.AssigneeName ?? "");

  const {
    ProductBlueprintID,
    Status,
    Page,
    Items,
    TotalPages,
    IsLoading,
    ErrorMessage,
    IsReportOpen,
    ReportReason,
    ReportDetail,
    ReportSubmitting,
    ReportErrorMessage,
    ReportResult,
    OnBack,
    OnReload,
    SetStatus,
    SetPage,
    OpenReport,
    CloseReport,
    SetReportReason,
    SetReportDetail,
    SubmitReport,
  } = useProductBlueprintReviewDetail();

  const Title =
    HeaderProductName ||
    (ProductBlueprintID
      ? `Review: ${ProductBlueprintID}`
      : "Review Detail");

  const [SortBy, setSortBy] = useState<SortKey>(null);
  const [SortDir, setSortDir] = useState<SortDir>("desc");

  const toggleSort = (key: Exclude<SortKey, null>) => {
    if (SortBy !== key) {
      setSortBy(key);
      setSortDir("desc");
      return;
    }

    setSortDir((current) =>
      current === "desc" ? "asc" : "desc",
    );
  };

  const sortLabel = (
    key: Exclude<SortKey, null>,
  ): string => {
    if (SortBy !== key) {
      return "↕";
    }

    return SortDir === "desc" ? "↓" : "↑";
  };

  const SortedItems = useMemo(() => {
    const rows = [...Items];

    if (!SortBy) {
      return rows;
    }

    const direction = SortDir === "asc" ? 1 : -1;

    if (SortBy === "Rating") {
      rows.sort((a, b) => {
        const ratingA = Number(a.Rating ?? 0);
        const ratingB = Number(b.Rating ?? 0);
        return (ratingA - ratingB) * direction;
      });

      return rows;
    }

    rows.sort((a, b) => {
      const reviewedAtA = String(a.ReviewedAt ?? "");
      const reviewedAtB = String(b.ReviewedAt ?? "");

      const timestampA = Date.parse(reviewedAtA);
      const timestampB = Date.parse(reviewedAtB);

      const isValidA = Number.isFinite(timestampA);
      const isValidB = Number.isFinite(timestampB);

      if (isValidA && isValidB) {
        return (timestampA - timestampB) * direction;
      }

      if (isValidA && !isValidB) {
        return -1 * direction;
      }

      if (!isValidA && isValidB) {
        return direction;
      }

      return reviewedAtA.localeCompare(reviewedAtB) * direction;
    });

    return rows;
  }, [Items, SortBy, SortDir]);

  return (
    <>
      <PageStyle
        layout="grid-2"
        title={Title}
        onBack={OnBack}
      >
        <div>
          <div className="pbrd-toolbar">
            <div className="pbrd-toolbar-left" />

            <div className="pbrd-toolbar-right">
              <select
                value={Status}
                onChange={(event) =>
                  SetStatus(
                    event.target.value as ReviewStatus,
                  )
                }
                className="border rounded px-2 py-1"
              >
                <option value="PUBLISHED">
                  {statusLabelJa("PUBLISHED")}
                </option>
                <option value="HIDDEN">
                  {statusLabelJa("HIDDEN")}
                </option>
                <option value="REMOVED">
                  {statusLabelJa("REMOVED")}
                </option>
              </select>

              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => toggleSort("Rating")}
                aria-label="Rating でソート"
                title="Rating でソート"
              >
                評価 {sortLabel("Rating")}
              </Button>

              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => toggleSort("ReviewedAt")}
                aria-label="ReviewedAt でソート"
                title="ReviewedAt でソート"
              >
                投稿日時 {sortLabel("ReviewedAt")}
              </Button>

              <RefreshButton
                onClick={OnReload}
                loading={IsLoading}
                title="リフレッシュ"
                ariaLabel="リフレッシュ"
              />
            </div>
          </div>

          {ErrorMessage ? (
            <div className="mb-3 text-sm text-red-600">
              {ErrorMessage}
            </div>
          ) : null}

          <div className="pbrd-reviewcard-wrapper">
            {IsLoading ? (
              <div className="pbrd-empty">
                読み込み中...
              </div>
            ) : SortedItems.length === 0 ? (
              <div className="pbrd-empty">
                No reviews
              </div>
            ) : (
              <div className="pbrd-grid">
                {SortedItems.map((review, index) => {
                  const ReviewID = String(
                    review.ID ?? "",
                  ).trim();

                  const ReviewKey =
                    ReviewID || `rv_${index}`;

                  const Body = String(
                    review.Body ?? "",
                  );

                  const TitleText = String(
                    review.Title ?? "",
                  );

                  const AvatarName = String(
                    review.AvatarName ?? "",
                  );

                  const AvatarIcon = String(
                    review.AvatarIcon ?? "",
                  );

                  const AuthorName =
                    AvatarName || "-";

                  const RatingStars = ratingToStars(
                    Number(review.Rating ?? 0),
                  );

                  const ReviewedAt = String(
                    review.ReviewedAt ?? "",
                  );

                  const StatusLabel = statusLabelJa(
                    review.Status,
                  );

                  const CanReport =
                    Boolean(ReviewID) &&
                    review.Status !== "REMOVED";

                  return (
                    <div
                      key={ReviewKey}
                      className="pbrd-review-item-card rounded-xl border border-slate-200 bg-white shadow-sm"
                    >
                      <div className="pbrd-author-row">
                        {AvatarIcon ? (
                          <img
                            src={AvatarIcon}
                            alt={`${AuthorName}のアイコン`}
                            className="pbrd-author-icon"
                          />
                        ) : null}

                        <span className="pbrd-author-primary">
                          {AuthorName}
                        </span>

                        <span className="pbrd-pill">
                          {StatusLabel}
                        </span>

                        <span className="pbrd-pill">
                          {RatingStars}
                        </span>
                      </div>

                      <div className="pbrd-title">
                        {TitleText || (
                          <span className="pbrd-body-empty">
                            （タイトルなし）
                          </span>
                        )}
                      </div>

                      <div className="pbrd-body">
                        {Body || (
                          <span className="pbrd-body-empty">
                            （本文なし）
                          </span>
                        )}
                      </div>

                      <div className="pbrd-datetime">
                        投稿日時: {ReviewedAt || "-"}
                      </div>

                      {CanReport ? (
                        <div className="mt-3 flex justify-end">
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            className="border-red-200 text-red-600 hover:bg-red-50 hover:text-red-700"
                            disabled={ReportSubmitting}
                            onClick={() => OpenReport(ReviewID)}
                          >
                            通報
                          </Button>
                        </div>
                      ) : null}
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          <Pagination
            currentPage={Page}
            totalPages={TotalPages}
            onPageChange={SetPage}
          />
        </div>

        <div>
          <AdminCard
            title="管理情報"
            assigneeName={HeaderAssigneeName}
            mode="view"
          />

          <div className="section-gap">
            <LogCard />
          </div>
        </div>
      </PageStyle>

      <ReviewReportModal
        open={IsReportOpen}
        targetType="PRODUCT_BLUEPRINT_REVIEW"
        reason={ReportReason}
        detail={ReportDetail}
        submitting={ReportSubmitting}
        errorMessage={ReportErrorMessage}
        result={ReportResult}
        onReasonChange={SetReportReason}
        onDetailChange={SetReportDetail}
        onSubmit={SubmitReport}
        onClose={CloseReport}
      />
    </>
  );
}