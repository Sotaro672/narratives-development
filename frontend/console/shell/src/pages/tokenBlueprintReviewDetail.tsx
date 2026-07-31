// frontend/console/shell/src/pages/tokenBlueprintReviewDetail.tsx

import { useMemo, useState } from "react";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import TokenContentsCard from "../features/tokenBlueprint/presentation/components/tokenContentsCard";
import LogCard from "../features/log/presentation/LogCard";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";
import { Button } from "../shared/ui/button";

import ReviewAggregateCard from "../features/tokenBlueprintReview/presentation/component/review_aggregate_card";
import ReviewCard from "../features/tokenBlueprintReview/presentation/component/review_card";

import { useTokenBlueprintReviewDetail } from "../features/tokenBlueprintReview/presentation/hook/use_tokenBlueprintReviewDetail";
import type { Comment } from "../shared/types/tokenBlueprintReview";

import "../styles/tokenBlueprintReview.css";

export default function TokenBlueprintReviewDetail() {
  const { vm, handlers } = useTokenBlueprintReviewDetail();
  const [commentBody, setCommentBody] = useState("");

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
  } = handlers;

  const tokenBlueprintName = String(
    blueprint?.name ?? "",
  );

  const likeCount = Number(
    reviewAggregate?.likeCount ?? 0,
  );

  const dislikeCount = Number(
    reviewAggregate?.dislikeCount ?? 0,
  );

  const reviewCount = Number(
    reviewAggregate?.topLevelCommentCount ?? 0,
  );

  const reviewList = useMemo<Comment[]>(() => {
    return comments.filter(
      (comment) => Number(comment.depth ?? 0) === 0,
    );
  }, [comments]);

  const repliesByParentId = useMemo(() => {
    const map = new Map<string, Comment[]>();

    for (const comment of comments) {
      const depth = Number(comment.depth ?? 0);
      const parentId = String(
        comment.parentCommentId ?? "",
      );

      if (depth <= 0 || parentId === "") {
        continue;
      }

      const current = map.get(parentId) ?? [];
      current.push(comment);
      map.set(parentId, current);
    }

    return map;
  }, [comments]);

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
    <PageStyle
      layout="grid-2"
      title={
        tokenBlueprintName ||
        "トークンレビュー"
      }
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
            disabled={submitting}
          />

          <div className="mt-2 flex justify-end">
            <Button
              type="button"
              size="sm"
              disabled={
                submitting ||
                commentBody.trim().length === 0
              }
              onClick={async () => {
                const body = commentBody.trim();

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
              {reviewList.map(
                (review, index) => (
                  <ReviewCard
                    key={String(
                      review.commentId ??
                        `cm_${index}`,
                    )}
                    item={review}
                    repliesByParentId={
                      repliesByParentId
                    }
                    fallbackIndex={index}
                    submitting={submitting}
                    onReply={async (
                      parentCommentId,
                      body,
                    ) => {
                      await createReply(
                        parentCommentId,
                        body,
                      );
                    }}
                    onDelete={async (
                      commentId,
                    ) => {
                      await deleteComment(
                        commentId,
                      );
                    }}
                    onReact={async (
                      commentId,
                      type,
                    ) => {
                      await reactToComment(
                        commentId,
                        type,
                      );
                    }}
                  />
                ),
              )}
            </div>
          )}
        </div>
      </div>

      <div className="space-y-4">
        <AdminCard
          title="管理情報"
          assigneeName={assigneeName}
          createdByName={createdByName}
          createdAt={safeDateTimeLabelJa(
            createdAt,
            createdAt || "-",
          )}
          updatedByName={updatedByName}
          updatedAt={safeDateTimeLabelJa(
            updatedAt,
            updatedAt || "-",
          )}
        />

        <LogCard title="更新ログ" />
      </div>
    </PageStyle>
  );
}