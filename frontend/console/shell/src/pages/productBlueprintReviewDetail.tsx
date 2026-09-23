// frontend/console/shell/src/pages/productBlueprintReviewDetail.tsx

import { useMemo, useState } from "react";
import { useLocation } from "react-router-dom";

import AdminCard from "../features/admin/presentation/components/AdminCard";
import LogCard from "../features/log/presentation/LogCard";
import {
  ratingToStars,
  statusLabelJa,
} from "../features/productBlueprintReview/presentation/component/review";
import { useProductBlueprintReviewDetail } from "../features/productBlueprintReview/presentation/hook/useProductBlueprintReviewDetail";
import ReportModal from "../features/report/presentation/components/ReportModal";
import PageStyle from "../layout/PageStyle/PageStyle";
import AvatarIcon from "../shared/ui/avatarIcon";
import {
  Badge,
  BadgeGroup,
  type BadgeVariant,
} from "../shared/ui/badge";
import { Button } from "../shared/ui/button";
import {
  Card,
  CardContent,
  CardFooter,
} from "../shared/ui/card";
import Empty from "../shared/ui/empty";
import { ErrorMessage } from "../shared/ui/error";
import Pagination from "../shared/ui/pagination";
import RefreshButton from "../shared/ui/refresh";
import Stack from "../shared/ui/stack";
import Text from "../shared/ui/text";
import type { ReviewStatus } from "../shared/types/productBlueprintReview";

import "../styles/productBlueprintReview.css";

type DetailNavState = {
  ProductName?: string;
  AssigneeName?: string;
};

type SortKey = "Rating" | "ReviewedAt" | null;
type SortDir = "asc" | "desc";

function getReviewStatusBadgeVariant(status: ReviewStatus): BadgeVariant {
  switch (status) {
    case "PUBLISHED":
      return "success";
    case "HIDDEN":
      return "warning";
    case "REMOVED":
      return "danger";
    default:
      return "default";
  }
}

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
    ErrorMessage: ErrorMessageText,
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

    setSortDir((current) => current === "desc" ? "asc" : "desc");
  };

  const sortLabel = (key: Exclude<SortKey, null>): string => {
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
      <PageStyle layout="grid-2" title={Title} onBack={OnBack}>
        <div>
          <div className="pbrd-toolbar">
            <div />

            <div className="pbrd-toolbar-right">
              <select
                value={Status}
                onChange={(event) =>
                  SetStatus(event.target.value as ReviewStatus)
                }
                className="pbrd-status-select"
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

          <Stack gap="md">
            {ErrorMessageText ? (
              <ErrorMessage variant="panel">
                {ErrorMessageText}
              </ErrorMessage>
            ) : null}

            {IsLoading ? (
              <Text as="div" size="xs" tone="muted">
                読み込み中...
              </Text>
            ) : SortedItems.length === 0 ? (
              <Empty description="現在登録されているレビューはございません。" />
            ) : (
              <Stack gap="sm">
                {SortedItems.map((review, index) => {
                  const ReviewID = String(review.ID ?? "").trim();
                  const ReviewKey = ReviewID || `rv_${index}`;
                  const Body = String(review.Body ?? "");
                  const AvatarName = String(review.AvatarName ?? "");
                  const AvatarIconUrl = String(review.AvatarIcon ?? "");
                  const AuthorName = AvatarName || "-";
                  const RatingStars = ratingToStars(Number(review.Rating ?? 0));
                  const ReviewedAt = String(review.ReviewedAt ?? "");
                  const StatusLabel = statusLabelJa(review.Status);
                  const CanReport =
                    Boolean(ReviewID) &&
                    review.Status !== "REMOVED";

                  return (
                    <Card key={ReviewKey} largeRadius>
                      <CardContent>
                        <Stack gap="sm">
                          <BadgeGroup>
                            <AvatarIcon
                              src={AvatarIconUrl}
                              alt={`${AuthorName}のアイコン`}
                            />

                            <Text size="xs" weight="semibold">
                              {AuthorName}
                            </Text>

                            <Badge
                              variant={getReviewStatusBadgeVariant(
                                review.Status,
                              )}
                            >
                              {StatusLabel}
                            </Badge>

                            <Badge variant="secondary">
                              {RatingStars}
                            </Badge>
                          </BadgeGroup>

                          <Text
                            as="div"
                            wrap="pre-wrap"
                          >
                            {Body || (
                              <Text tone="muted">
                                （本文なし）
                              </Text>
                            )}
                          </Text>

                          <Text
                            as="div"
                            size="xs"
                            tone="muted"
                          >
                            投稿日時: {ReviewedAt || "-"}
                          </Text>
                        </Stack>
                      </CardContent>

                      {CanReport ? (
                        <CardFooter>
                          <Button
                            type="button"
                            variant="destructive-outline"
                            size="sm"
                            disabled={ReportSubmitting}
                            onClick={() => OpenReport(ReviewID)}
                          >
                            通報
                          </Button>
                        </CardFooter>
                      ) : null}
                    </Card>
                  );
                })}
              </Stack>
            )}

            <Pagination
              currentPage={Page}
              totalPages={TotalPages}
              onPageChange={SetPage}
            />
          </Stack>
        </div>

        <Stack gap="md">
          <AdminCard
            title="管理情報"
            assigneeName={HeaderAssigneeName}
            mode="view"
          />

          <LogCard />
        </Stack>
      </PageStyle>

      <ReportModal
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