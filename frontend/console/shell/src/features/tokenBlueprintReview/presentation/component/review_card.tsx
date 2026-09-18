// frontend/console/shell/src/features/tokenBlueprintReview/presentation/component/review_card.tsx

import { useMemo, useState } from "react";

import type {
  Comment,
  ReactionType,
} from "../../../../shared/types/tokenBlueprintReview";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import { Badge } from "../../../../shared/ui/badge";
import { Button } from "../../../../shared/ui/button";
import Textarea from "../../../../shared/ui/textarea";

import "./review_card.css";

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
    <div className="token-blueprint-review-card">
      <div className="token-blueprint-review-card__author-row">
        {authorIcon ? (
          <img
            src={authorIcon}
            alt="author icon"
            className="token-blueprint-review-card__author-icon"
          />
        ) : null}

        <span>{authorPrimary}</span>

        {authorType === "brand" && isOwnerComment ? (
          <Badge variant="secondary">
            投稿者
          </Badge>
        ) : null}

        <span>
          {createdAtLabel}
        </span>
      </div>

      <div className="token-blueprint-review-card__body">
        {body || (
          <span className="token-blueprint-review-card__body-empty">
            （本文なし）
          </span>
        )}
      </div>

      <div className="token-blueprint-review-card__meta-row">
        <Button
          type="button"
          variant="outline"
          size="sm"
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
            className="token-blueprint-review-card__danger-button"
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
            className="token-blueprint-review-card__danger-button"
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

      <div className="token-blueprint-review-card__meta-row">
        <span>返信数: {visibleReplyCount}</span>
      </div>

      {isReplyFormOpen ? (
        <div className="token-blueprint-review-card__reply-form">
          <Textarea
            value={replyBody}
            onChange={(event) => {
              setReplyBody(event.target.value);
            }}
            placeholder="返信を入力してください"
            disabled={disabled}
          />

          <div className="token-blueprint-review-card__reply-actions">
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
        <div className="token-blueprint-review-card__replies">
          <div className="token-blueprint-review-card__reply-list">
            {sortedReplies.map((reply) => (
              <div
                key={reply.commentId}
                className="token-blueprint-review-card__reply-item"
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