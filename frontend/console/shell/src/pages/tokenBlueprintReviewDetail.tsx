// frontend/console/shell/src/pages/tokenBlueprintReviewDetail.tsx

import { useMemo, useRef, useState } from "react";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import TokenContentsCard from "../features/tokenBlueprint/presentation/components/tokenContentsCard";
import LogCard from "../features/log/presentation/LogCard";
import ReviewReportModal from "../features/reviewReport/presentation/components/ReviewReportModal";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";
import { Button } from "../shared/ui/button";

import ReviewAggregateCard from "../features/tokenBlueprintReview/presentation/component/review_aggregate_card";
import ReviewCard from "../features/tokenBlueprintReview/presentation/component/review_card";

import { useTokenBlueprintReviewDetail } from "../features/tokenBlueprintReview/presentation/hook/use_tokenBlueprintReviewDetail";
import type { Comment } from "../shared/types/tokenBlueprintReview";
import type {
  ReviewReportReason,
  ReviewReportResponse,
} from "../shared/types/reviewReport";
import { requiresReviewReportDetail } from "../shared/types/reviewReport";

import "../styles/tokenBlueprintReview.css";

const DEFAULT_REPORT_REASON: ReviewReportReason = "INAPPROPRIATE";

export default function TokenBlueprintReviewDetail() {
  const { vm, handlers } = useTokenBlueprintReviewDetail();
  const [commentBody, setCommentBody] = useState("");

  const [reportTargetCommentId, setReportTargetCommentId] = useState("");
  const [reportReason, setReportReason] = useState<ReviewReportReason>(
    DEFAULT_REPORT_REASON,
  );
  const [reportDetail, setReportDetail] = useState("");
  const [reportSubmitting, setReportSubmitting] = useState(false);
  const [reportErrorMessage, setReportErrorMessage] = useState("");
  const [reportResult, setReportResult] = useState<ReviewReportResponse | null>(
    null,
  );
  const reportSubmittingRef = useRef(false);

  const {
    blueprint,
    assigneeName,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
    tokenContents,
    reviewAggregate,
    comments,
    submitting,
  } = vm;

  const {
    onBack,
    createComment,
    createReply,
    deleteComment,
    reactToComment,
    reportComment,
  } = handlers;

  const tokenBlueprintName = blueprint?.name ?? "";
  const likeCount = reviewAggregate?.likeCount ?? 0;
  const dislikeCount = reviewAggregate?.dislikeCount ?? 0;
  const reviewCount = reviewAggregate?.topLevelCommentCount ?? 0;
  const isReportOpen = Boolean(reportTargetCommentId);

  const reviewList = useMemo<Comment[]>(() => {
    return comments.filter((comment) => comment.depth === 0);
  }, [comments]);

  const repliesByParentId = useMemo(() => {
    const map = new Map<string, Comment[]>();

    for (const comment of comments) {
      if (comment.depth <= 0 || !comment.parentCommentId) {
        continue;
      }

      const current = map.get(comment.parentCommentId) ?? [];
      current.push(comment);
      map.set(comment.parentCommentId, current);
    }

    return map;
  }, [comments]);

  const openReport = (commentId: string) => {
    const normalizedCommentId = commentId.trim();

    if (!normalizedCommentId || reportSubmittingRef.current) {
      return;
    }

    setReportTargetCommentId(normalizedCommentId);
    setReportReason(DEFAULT_REPORT_REASON);
    setReportDetail("");
    setReportErrorMessage("");
    setReportResult(null);
  };

  const closeReport = () => {
    if (reportSubmittingRef.current) {
      return;
    }

    setReportTargetCommentId("");
    setReportReason(DEFAULT_REPORT_REASON);
    setReportDetail("");
    setReportErrorMessage("");
    setReportResult(null);
  };

  const handleReportReasonChange = (reason: ReviewReportReason) => {
    if (reportSubmittingRef.current) {
      return;
    }

    setReportReason(reason);
    setReportErrorMessage("");

    if (!requiresReviewReportDetail(reason)) {
      setReportDetail("");
    }
  };

  const handleReportDetailChange = (detail: string) => {
    if (reportSubmittingRef.current) {
      return;
    }

    setReportDetail(detail);
    setReportErrorMessage("");
  };

  const submitReport = async (): Promise<void> => {
    const normalizedCommentId = reportTargetCommentId.trim();
    const normalizedDetail = reportDetail.trim();

    if (
      reportSubmittingRef.current ||
      !normalizedCommentId ||
      reportResult
    ) {
      return;
    }

    if (
      requiresReviewReportDetail(reportReason) &&
      !normalizedDetail
    ) {
      setReportErrorMessage(
        "「その他」を選択した場合は詳細を入力してください。",
      );
      return;
    }

    reportSubmittingRef.current = true;
    setReportSubmitting(true);
    setReportErrorMessage("");

    try {
      const result = await reportComment(
        normalizedCommentId,
        reportReason,
        normalizedDetail || undefined,
      );

      setReportResult(result);
    } catch (error: unknown) {
      const message =
        error instanceof Error
          ? error.message
          : String(error ?? "コメントの通報に失敗しました。");

      setReportErrorMessage(
        message || "コメントの通報に失敗しました。",
      );
    } finally {
      reportSubmittingRef.current = false;
      setReportSubmitting(false);
    }
  };

  if (!blueprint) {
    return (
      <PageStyle
        layout="single"
        title="トークンレビュー"
        onBack={onBack}
      >
        <p className="p-4 text-sm text-muted-foreground">
          表示可能なトークン設計レビューがありません。
        </p>
      </PageStyle>
    );
  }

  return (
    <>
      <PageStyle
        layout="grid-2"
        title={tokenBlueprintName || "トークンレビュー"}
        onBack={onBack}
      >
        <div>
          <TokenContentsCard
            mode="view"
            contents={tokenContents}
          />

          <div className="tbrd-reviewcard-wrapper">
            <ReviewAggregateCard
              likeCount={likeCount}
              dislikeCount={dislikeCount}
              reviewCount={reviewCount}
            />
          </div>

          <div className="tbrd-section">
            <div className="tbrd-section-title">
              コメントを投稿
            </div>

            <textarea
              value={commentBody}
              onChange={(event) => {
                setCommentBody(event.target.value);
              }}
              placeholder="コメントを入力してください"
              className="w-full min-h-[96px] rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-slate-400"
              disabled={submitting || reportSubmitting}
            />

            <div className="mt-2 flex justify-end">
              <Button
                type="button"
                size="sm"
                disabled={
                  submitting ||
                  reportSubmitting ||
                  commentBody.trim().length === 0
                }
                onClick={async () => {
                  const body = commentBody.trim();

                  if (!body) {
                    return;
                  }

                  await createComment(body);
                  setCommentBody("");
                }}
              >
                {submitting ? "投稿中..." : "投稿"}
              </Button>
            </div>
          </div>

          <div className="tbrd-section">
            <div className="tbrd-section-title">
              Comments ({reviewList.length})
            </div>

            {reviewList.length === 0 ? (
              <div className="tbrd-empty">
                comments はありません
              </div>
            ) : (
              <div className="tbrd-grid">
                {reviewList.map((review) => (
                  <ReviewCard
                    key={review.commentId}
                    item={review}
                    repliesByParentId={repliesByParentId}
                    submitting={submitting || reportSubmitting}
                    onReply={async (parentCommentId, body) => {
                      await createReply(
                        parentCommentId,
                        body,
                      );
                    }}
                    onDelete={async (commentId) => {
                      await deleteComment(commentId);
                    }}
                    onReact={async (commentId, type) => {
                      await reactToComment(
                        commentId,
                        type,
                      );
                    }}
                    onReport={openReport}
                  />
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="space-y-4">
          <AdminCard
            title="管理情報"
            assigneeName={assigneeName}
            createdByName={createdByName}
            createdAt={safeDateTimeLabelJa(createdAt, "-")}
            updatedByName={updatedByName}
            updatedAt={safeDateTimeLabelJa(updatedAt, "-")}
          />

          <LogCard title="更新ログ" />
        </div>
      </PageStyle>

      <ReviewReportModal
        open={isReportOpen}
        targetType="TOKEN_BLUEPRINT_COMMENT"
        reason={reportReason}
        detail={reportDetail}
        submitting={reportSubmitting}
        errorMessage={reportErrorMessage}
        result={reportResult}
        onReasonChange={handleReportReasonChange}
        onDetailChange={handleReportDetailChange}
        onSubmit={submitReport}
        onClose={closeReport}
      />
    </>
  );
}