// frontend/console/tokenBlueprintReview/src/presentation/pages/tokenBlueprintReviewDetail.tsx
import { useMemo, useState } from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenContentsCard from "../../../../tokenBlueprint/src/presentation/components/tokenContentsCard";
import LogCard from "../../../../log/presentation/LogCard";
import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";
import { Button } from "../../../../shell/src/shared/ui/button";

import ReviewAggregateCard from "../component/review_aggregate_card";
import ReviewCard from "../component/review_card";

import { useTokenBlueprintReviewDetail } from "../hook/use_tokenBlueprintReviewDetail";
import type { Comment } from "../../domain/entity";

import "../../style/tokenBlueprintReview.css";

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
    reactToComment,
  } = handlers;

  const tokenBlueprintName = String(blueprint?.name ?? "");
  const likeCount = Number(reviewAggregate?.likeCount ?? 0);
  const dislikeCount = Number(reviewAggregate?.dislikeCount ?? 0);
  const reviewCount = Number(reviewAggregate?.topLevelCommentCount ?? 0);

  const reviewList: Comment[] = useMemo(() => {
    return comments.filter((c) => Number(c.depth ?? 0) === 0);
  }, [comments]);

  const repliesByParentId = useMemo(() => {
    const map = new Map<string, Comment[]>();

    for (const c of comments) {
      const depth = Number(c.depth ?? 0);
      const parentId = String(c.parentCommentId ?? "");
      if (depth <= 0 || parentId === "") continue;

      const current = map.get(parentId) ?? [];
      current.push(c);
      map.set(parentId, current);
    }

    for (const [parentId, items] of map.entries()) {
      map.set(
        parentId,
        [...items].sort((a, b) => {
          const at = Date.parse(String(a.createdAt ?? ""));
          const bt = Date.parse(String(b.createdAt ?? ""));
          if (Number.isNaN(at) && Number.isNaN(bt)) return 0;
          if (Number.isNaN(at)) return -1;
          if (Number.isNaN(bt)) return 1;
          return at - bt;
        }),
      );
    }

    return map;
  }, [comments]);

  if (!blueprint) {
    return (
      <PageStyle layout="single" title="トークンレビュー" onBack={onBack}>
        <p className="p-4 text-sm text-muted-foreground">
          表示可能なトークン設計レビューがありません。
        </p>
      </PageStyle>
    );
  }

  return (
    <PageStyle
      layout="grid-2"
      title={tokenBlueprintName ? tokenBlueprintName : "トークンレビュー"}
      onBack={onBack}
    >
      <div>
        <TokenContentsCard mode="view" contents={tokenContents} />

        <div className="tbrd-reviewcard-wrapper">
          <ReviewAggregateCard
            likeCount={likeCount}
            dislikeCount={dislikeCount}
            reviewCount={reviewCount}
          />
        </div>

        <div className="tbrd-section">
          <div className="tbrd-section-title">コメントを投稿</div>

          <textarea
            value={commentBody}
            onChange={(e) => setCommentBody(e.target.value)}
            placeholder="コメントを入力してください"
            className="w-full min-h-[96px] rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-slate-400"
            disabled={submitting}
          />

          <div className="mt-2 flex justify-end">
            <Button
              type="button"
              size="sm"
              disabled={submitting || commentBody.trim().length === 0}
              onClick={async () => {
                await createComment(commentBody.trim());
                setCommentBody("");
              }}
            >
              投稿
            </Button>
          </div>
        </div>

        <div className="tbrd-section">
          <div className="tbrd-section-title">Comments ({reviewList.length})</div>

          {reviewList.length === 0 ? (
            <div className="tbrd-empty">comments はありません</div>
          ) : (
            <div className="tbrd-grid">
              {reviewList.map((r: Comment, idx: number) => (
                <ReviewCard
                  key={String(r.commentId ?? `cm_${idx}`)}
                  item={r}
                  repliesByParentId={repliesByParentId}
                  fallbackIndex={idx}
                  submitting={submitting}
                  onReply={async (parentCommentId, body) => {
                    await createReply(parentCommentId, body);
                  }}
                  onReact={async (commentId, type) => {
                    await reactToComment(commentId, type);
                  }}
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
          createdAt={safeDateTimeLabelJa(createdAt, createdAt || "-")}
          updatedByName={updatedByName}
          updatedAt={safeDateTimeLabelJa(updatedAt, updatedAt || "-")}
        />

        <LogCard title="更新ログ" />
      </div>
    </PageStyle>
  );
}