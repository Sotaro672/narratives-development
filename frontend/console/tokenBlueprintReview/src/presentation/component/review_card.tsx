// frontend/console/tokenBlueprintReview/src/presentation/component/review_card.tsx

import { useMemo, useState } from "react";
import type { Comment, ReactionType } from "../../domain/entity";
import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";
import { Button } from "../../../../shell/src/shared/ui/button";
type ReviewCardProps = {
  item: Comment;
  repliesByParentId?: Map<string, Comment[]>;
  fallbackIndex?: number;
  submitting?: boolean;
  onReply?: (parentCommentId: string, body: string) => Promise<void> | void;
  onReact?: (commentId: string, type: ReactionType) => Promise<void> | void;
};

export default function ReviewCard({
  item,
  repliesByParentId,
  fallbackIndex = 0,
  submitting = false,
  onReply,
  onReact,
}: ReviewCardProps) {
  const [isReplyFormOpen, setIsReplyFormOpen] = useState(false);
  const [isRepliesOpen, setIsRepliesOpen] = useState(false);
  const [replyBody, setReplyBody] = useState("");
  const [isSubmittingReply, setIsSubmittingReply] = useState(false);
  const [isSubmittingReaction, setIsSubmittingReaction] = useState(false);

  const commentId = String(item.commentId ?? `cm_${fallbackIndex}`);
  const body = String(item.body ?? "");

  const authorId = String(item.authorId ?? "");
  const authorType = String(item.authorType ?? "");
  const authorAvatarName = String(item.authorAvatarName ?? "");
  const authorAvatarIcon = String(item.authorAvatarIcon ?? "");
  const brandName = String(item.brandName ?? "");
  const brandIcon = String(item.brandIcon ?? "");

  const likeCount = Number(item.likeCount ?? 0);
  const dislikeCount = Number(item.dislikeCount ?? 0);
  const childCount = Number(item.childCount ?? 0);

  const createdAtRaw = item.createdAt ?? null;
  const createdAt = safeDateTimeLabelJa(createdAtRaw, "-");

  const deleted = Boolean(item.deleted ?? false);
  const isOwnerComment = Boolean(item.isOwnerComment ?? false);

  const authorPrimary =
    authorType === "brand"
      ? brandName || authorId || "-"
      : authorAvatarName || authorId || "-";

  const authorIcon =
    authorType === "brand"
      ? brandIcon
      : authorAvatarIcon;

  const disabled = submitting || isSubmittingReply || isSubmittingReaction;

  const canReply = useMemo(() => {
    return !deleted && commentId !== "";
  }, [deleted, commentId]);

  const canReact = useMemo(() => {
    return !deleted && commentId !== "";
  }, [deleted, commentId]);

  const replies = useMemo(() => {
    if (!repliesByParentId || commentId === "") return [];
    return repliesByParentId.get(commentId) ?? [];
  }, [repliesByParentId, commentId]);

  const sortedReplies = useMemo(() => {
    return [...replies].sort((a, b) => {
      const at = Date.parse(String(a.createdAt ?? ""));
      const bt = Date.parse(String(b.createdAt ?? ""));
      if (Number.isNaN(at) && Number.isNaN(bt)) return 0;
      if (Number.isNaN(at)) return -1;
      if (Number.isNaN(bt)) return 1;
      return at - bt;
    });
  }, [replies]);

  const toggleReplyForm = () => {
    if (!canReply || disabled) return;
    setIsReplyFormOpen((prev) => !prev);
  };

  const toggleRepliesAccordion = () => {
    if (childCount <= 0 && sortedReplies.length <= 0) return;
    setIsRepliesOpen((prev) => !prev);
  };

  const closeReplyForm = () => {
    if (isSubmittingReply) return;
    setIsReplyFormOpen(false);
    setReplyBody("");
  };

  const handleReplySubmit = async () => {
    const trimmed = replyBody.trim();
    if (!trimmed || !onReply || !canReply || disabled) return;

    try {
      setIsSubmittingReply(true);
      await onReply(commentId, trimmed);
      setReplyBody("");
      setIsReplyFormOpen(false);
      setIsRepliesOpen(true);
    } finally {
      setIsSubmittingReply(false);
    }
  };

  const handleReaction = async (type: ReactionType) => {
    if (!onReact || !canReact || disabled) return;

    try {
      setIsSubmittingReaction(true);
      await onReact(commentId, type);
    } finally {
      setIsSubmittingReaction(false);
    }
  };

  return (
    <div
      key={commentId}
      className="bg-white border border-slate-200 rounded-xl shadow-sm tbrd-review-item-card"
    >
      <div className="tbrd-author-row">
        {authorIcon ? (
          <img
            src={authorIcon}
            alt="author icon"
            className="tbrd-author-icon"
          />
        ) : null}

        <span>{authorPrimary}</span>

        {authorType === "brand" && isOwnerComment ? (
          <span className="inline-flex items-center rounded-full border border-slate-300 px-2 py-0.5 text-[11px] text-slate-600">
            投稿者
          </span>
        ) : null}

        <span className="tbrd-created-at">{createdAt}</span>
      </div>

      <div className="tbrd-body">
        {body || <span className="tbrd-body-empty">（本文なし）</span>}
      </div>

      <div className="tbrd-meta-row">
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="tbrd-reaction-button"
          disabled={!canReact || disabled}
          onClick={() => {
            void handleReaction("like");
          }}
        >
          👍 {Number.isFinite(likeCount) ? likeCount : 0}
        </Button>

        <Button
          type="button"
          variant="outline"
          size="sm"
          className="tbrd-reaction-button"
          disabled={!canReact || disabled}
          onClick={() => {
            void handleReaction("dislike");
          }}
        >
          👎 {Number.isFinite(dislikeCount) ? dislikeCount : 0}
        </Button>

        <Button
          type="button"
          variant="outline"
          size="sm"
          className="tbrd-reply-button"
          disabled={!canReply || disabled}
          onClick={toggleReplyForm}
        >
          {isReplyFormOpen ? "返信を閉じる" : "返信"}
        </Button>

        {(childCount > 0 || sortedReplies.length > 0) ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="tbrd-reply-button"
            onClick={toggleRepliesAccordion}
          >
            {isRepliesOpen
              ? `返信を隠す (${Number.isFinite(childCount) ? childCount : sortedReplies.length})`
              : `返信を表示 (${Number.isFinite(childCount) ? childCount : sortedReplies.length})`}
          </Button>
        ) : null}

        {deleted ? <span>削除済み</span> : null}
      </div>

      <div className="tbrd-meta-row">
        <span>返信数: {Number.isFinite(childCount) ? childCount : 0}</span>
      </div>

      {isReplyFormOpen ? (
        <div className="mt-3 border-t border-slate-200 pt-3">
          <textarea
            value={replyBody}
            onChange={(e) => setReplyBody(e.target.value)}
            placeholder="返信を入力してください"
            className="w-full min-h-[96px] rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-slate-400"
            disabled={disabled}
          />

          <div className="mt-2 flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={closeReplyForm}
              disabled={disabled}
            >
              キャンセル
            </Button>
            <Button
              type="button"
              size="sm"
              disabled={replyBody.trim().length === 0 || disabled}
              onClick={() => {
                void handleReplySubmit();
              }}
            >
              {isSubmittingReply ? "送信中..." : "送信"}
            </Button>
          </div>
        </div>
      ) : null}

      {isRepliesOpen ? (
        <div className="mt-3 border-t border-slate-200 pt-3">
          {sortedReplies.length === 0 ? (
            <div className="text-sm text-slate-500">返信はありません</div>
          ) : (
            <div className="flex flex-col gap-3">
              {sortedReplies.map((reply, idx) => (
                <div
                  key={String(reply.commentId ?? `${commentId}_reply_${idx}`)}
                  className="ml-4 rounded-lg border border-slate-200 bg-slate-50 p-3"
                >
                  <ReviewCard
                    item={reply}
                    repliesByParentId={repliesByParentId}
                    fallbackIndex={idx}
                    submitting={submitting}
                    onReply={onReply}
                    onReact={onReact}
                  />
                </div>
              ))}
            </div>
          )}
        </div>
      ) : null}
    </div>
  );
}