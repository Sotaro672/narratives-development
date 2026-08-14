// frontend/console/shell/src/features/tokenBlueprintReview/presentation/component/review_card.tsx

import { useMemo, useState } from "react";

import type {
  Comment,
  ReactionType,
} from "../../../../shared/types/tokenBlueprintReview";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import { Button } from "../../../../shared/ui/button";

type ReviewCardProps = {
  item: Comment;
  repliesByParentId?: Map<string, Comment[]>;
  submitting?: boolean;
  onReply?: (
    parentCommentId: string,
    body: string,
  ) => Promise<void> | void;
  onDelete?: (
    commentId: string,
  ) => Promise<void> | void;
  onReact?: (
    commentId: string,
    type: ReactionType,
  ) => Promise<void> | void;
};

export default function ReviewCard({
  item,
  repliesByParentId,
  submitting = false,
  onReply,
  onDelete,
  onReact,
}: ReviewCardProps) {
  const [isReplyFormOpen, setIsReplyFormOpen] = useState(false);
  const [isRepliesOpen, setIsRepliesOpen] = useState(false);
  const [replyBody, setReplyBody] = useState("");
  const [isSubmittingReply, setIsSubmittingReply] = useState(false);
  const [isSubmittingDelete, setIsSubmittingDelete] = useState(false);
  const [isSubmittingReaction, setIsSubmittingReaction] = useState(false);

  const {
    commentId,
    body,
    authorId,
    authorType,
    authorAvatarName,
    authorAvatarIcon,
    brandName,
    brandIcon,
    likeCount,
    dislikeCount,
    childCount,
    createdAt,
    deleted,
    isOwnerComment,
  } = item;

  const createdAtLabel = safeDateTimeLabelJa(createdAt, "-");

  const authorPrimary =
    authorType === "brand"
      ? brandName || authorId || "-"
      : authorAvatarName || authorId || "-";

  const authorIcon =
    authorType === "brand"
      ? brandIcon
      : authorAvatarIcon;

  const disabled =
    submitting ||
    isSubmittingReply ||
    isSubmittingDelete ||
    isSubmittingReaction;

  const canInteract = !deleted;

  const canDelete =
    canInteract &&
    authorType === "brand" &&
    isOwnerComment &&
    onDelete !== undefined;

  const replies = useMemo(() => {
    if (!repliesByParentId) {
      return [];
    }

    return repliesByParentId.get(commentId) ?? [];
  }, [repliesByParentId, commentId]);

  const sortedReplies = useMemo(() => {
    return [...replies].sort(
      (a, b) =>
        Date.parse(a.createdAt) -
        Date.parse(b.createdAt),
    );
  }, [replies]);

  const toggleReplyForm = () => {
    if (!canInteract || disabled) {
      return;
    }

    setIsReplyFormOpen((previous) => !previous);
  };

  const toggleRepliesAccordion = () => {
    if (
      childCount <= 0 &&
      sortedReplies.length <= 0
    ) {
      return;
    }

    setIsRepliesOpen((previous) => !previous);
  };

  const closeReplyForm = () => {
    if (isSubmittingReply) {
      return;
    }

    setIsReplyFormOpen(false);
    setReplyBody("");
  };

  const handleReplySubmit = async () => {
    const content = replyBody.trim();

    if (
      !content ||
      !onReply ||
      !canInteract ||
      disabled
    ) {
      return;
    }

    try {
      setIsSubmittingReply(true);
      await onReply(commentId, content);
      setReplyBody("");
      setIsReplyFormOpen(false);
      setIsRepliesOpen(true);
    } finally {
      setIsSubmittingReply(false);
    }
  };

  const handleDelete = async () => {
    if (!onDelete || !canDelete || disabled) {
      return;
    }

    const confirmed = window.confirm(
      "このコメントを削除しますか？",
    );

    if (!confirmed) {
      return;
    }

    try {
      setIsSubmittingDelete(true);
      await onDelete(commentId);
    } finally {
      setIsSubmittingDelete(false);
    }
  };

  const handleReaction = async (
    type: ReactionType,
  ) => {
    if (
      !onReact ||
      !canInteract ||
      disabled
    ) {
      return;
    }

    try {
      setIsSubmittingReaction(true);
      await onReact(commentId, type);
    } finally {
      setIsSubmittingReaction(false);
    }
  };

  return (
    <div className="bg-white border border-slate-200 rounded-xl shadow-sm tbrd-review-item-card">
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

        <span className="tbrd-created-at">
          {createdAtLabel}
        </span>
      </div>

      <div className="tbrd-body">
        {body || (
          <span className="tbrd-body-empty">
            （本文なし）
          </span>
        )}
      </div>

      <div className="tbrd-meta-row">
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="tbrd-reaction-button"
          disabled={!canInteract || disabled}
          onClick={() => {
            void handleReaction("like");
          }}
        >
          👍 {likeCount}
        </Button>

        <Button
          type="button"
          variant="outline"
          size="sm"
          className="tbrd-reaction-button"
          disabled={!canInteract || disabled}
          onClick={() => {
            void handleReaction("dislike");
          }}
        >
          👎 {dislikeCount}
        </Button>

        <Button
          type="button"
          variant="outline"
          size="sm"
          className="tbrd-reply-button"
          disabled={!canInteract || disabled}
          onClick={toggleReplyForm}
        >
          {isReplyFormOpen
            ? "返信を閉じる"
            : "返信"}
        </Button>

        {childCount > 0 || sortedReplies.length > 0 ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="tbrd-reply-button"
            disabled={disabled}
            onClick={toggleRepliesAccordion}
          >
            {isRepliesOpen
              ? `返信を隠す (${childCount})`
              : `返信を表示 (${childCount})`}
          </Button>
        ) : null}

        {canDelete ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="border-red-200 text-red-600 hover:bg-red-50 hover:text-red-700"
            disabled={disabled}
            onClick={() => {
              void handleDelete();
            }}
          >
            {isSubmittingDelete
              ? "削除中..."
              : "削除"}
          </Button>
        ) : null}

        {deleted ? <span>削除済み</span> : null}
      </div>

      <div className="tbrd-meta-row">
        <span>返信数: {childCount}</span>
      </div>

      {isReplyFormOpen ? (
        <div className="mt-3 border-t border-slate-200 pt-3">
          <textarea
            value={replyBody}
            onChange={(event) => {
              setReplyBody(event.target.value);
            }}
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
              disabled={
                replyBody.trim().length === 0 ||
                disabled
              }
              onClick={() => {
                void handleReplySubmit();
              }}
            >
              {isSubmittingReply
                ? "送信中..."
                : "送信"}
            </Button>
          </div>
        </div>
      ) : null}

      {isRepliesOpen ? (
        <div className="mt-3 border-t border-slate-200 pt-3">
          {sortedReplies.length === 0 ? (
            <div className="text-sm text-slate-500">
              返信はありません
            </div>
          ) : (
            <div className="flex flex-col gap-3">
              {sortedReplies.map((reply) => (
                <div
                  key={reply.commentId}
                  className="ml-4 rounded-lg border border-slate-200 bg-slate-50 p-3"
                >
                  <ReviewCard
                    item={reply}
                    repliesByParentId={repliesByParentId}
                    submitting={submitting}
                    onReply={onReply}
                    onDelete={onDelete}
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