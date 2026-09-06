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
  onReport?: (
    commentId: string,
  ) => void;
};

export default function ReviewCard({
  item,
  repliesByParentId,
  submitting = false,
  onReply,
  onDelete,
  onReact,
  onReport,
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
    createdAt,
    deleted,
    isOwnerComment,
  } = item;

  const normalizedCommentId = commentId.trim();
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

  const canDelete =
    authorType === "brand" &&
    isOwnerComment &&
    onDelete !== undefined;

  const canReport =
    Boolean(normalizedCommentId) &&
    !(authorType === "brand" && isOwnerComment) &&
    onReport !== undefined;

  const replies = useMemo(() => {
    if (!repliesByParentId) {
      return [];
    }

    return repliesByParentId.get(commentId) ?? [];
  }, [repliesByParentId, commentId]);

  const sortedReplies = useMemo(() => {
    return replies
      .filter((reply) => !reply.deleted)
      .sort(
        (a, b) =>
          Date.parse(a.createdAt) -
          Date.parse(b.createdAt),
      );
  }, [replies]);

  const visibleReplyCount = sortedReplies.length;

  const toggleReplyForm = () => {
    if (disabled) {
      return;
    }

    setIsReplyFormOpen((previous) => !previous);
  };

  const toggleRepliesAccordion = () => {
    if (visibleReplyCount <= 0) {
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
      disabled ||
      !normalizedCommentId
    ) {
      return;
    }

    try {
      setIsSubmittingReply(true);
      await onReply(normalizedCommentId, content);
      setReplyBody("");
      setIsReplyFormOpen(false);
      setIsRepliesOpen(true);
    } finally {
      setIsSubmittingReply(false);
    }
  };

  const handleDelete = async () => {
    if (
      !onDelete ||
      !canDelete ||
      disabled ||
      !normalizedCommentId
    ) {
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
      await onDelete(normalizedCommentId);
    } finally {
      setIsSubmittingDelete(false);
    }
  };

  const handleReaction = async (
    type: ReactionType,
  ) => {
    if (
      !onReact ||
      disabled ||
      !normalizedCommentId
    ) {
      return;
    }

    try {
      setIsSubmittingReaction(true);
      await onReact(normalizedCommentId, type);
    } finally {
      setIsSubmittingReaction(false);
    }
  };

  const handleReport = () => {
    if (
      !onReport ||
      !canReport ||
      disabled ||
      !normalizedCommentId
    ) {
      return;
    }

    onReport(normalizedCommentId);
  };

  if (deleted) {
    return null;
  }

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
          disabled={disabled}
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
          disabled={disabled}
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
          disabled={disabled}
          onClick={toggleReplyForm}
        >
          {isReplyFormOpen
            ? "返信を閉じる"
            : "返信"}
        </Button>

        {visibleReplyCount > 0 ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="tbrd-reply-button"
            disabled={disabled}
            onClick={toggleRepliesAccordion}
          >
            {isRepliesOpen
              ? `返信を隠す (${visibleReplyCount})`
              : `返信を表示 (${visibleReplyCount})`}
          </Button>
        ) : null}

        {canReport ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="border-red-200 text-red-600 hover:bg-red-50 hover:text-red-700"
            disabled={disabled}
            onClick={handleReport}
          >
            通報
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
      </div>

      <div className="tbrd-meta-row">
        <span>返信数: {visibleReplyCount}</span>
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

      {isRepliesOpen && visibleReplyCount > 0 ? (
        <div className="mt-3 border-t border-slate-200 pt-3">
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
                  onReport={onReport}
                />
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}