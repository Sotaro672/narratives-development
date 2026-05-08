// frontend/amol/src/features/token-commnet/components/TokenCommentItem.tsx

import type { ChangeEvent } from "react";

import type {
  TokenComment,
  TokenCommentTreeNode,
} from "../types/tokenCommentTypes";
import {
  getTokenCommentDisplayIconUrl,
  getTokenCommentDisplayName,
} from "../types/tokenCommentTypes";
import { hasTokenCommentChildren } from "../utils/commentTree";

type TokenCommentItemProps = {
  node: TokenCommentTreeNode;
  expandedIds: Set<string>;
  replyingCommentId: string | null;
  replyBody: string;
  replyPosting: boolean;
  onToggleExpanded: (commentId: string) => void;
  onLike: (commentId: string) => void | Promise<void>;
  onDislike: (commentId: string) => void | Promise<void>;
  onStartReply: (commentId: string) => void;
  onCancelReply: () => void;
  onReplyBodyChange: (value: string) => void;
  onSubmitReply: (parentCommentId: string) => void | Promise<void>;
};

function formatCommentDate(value: string): string {
  if (!value) {
    return "";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function getFallbackAuthorName(comment: TokenComment): string {
  if (!comment.authorId) {
    return "unknown";
  }

  if (comment.authorId.length <= 10) {
    return comment.authorId;
  }

  return `${comment.authorId.slice(0, 6)}...${comment.authorId.slice(-4)}`;
}

function TokenCommentAuthor({ comment }: { comment: TokenComment }) {
  const displayName =
    getTokenCommentDisplayName(comment) || getFallbackAuthorName(comment);
  const iconUrl = getTokenCommentDisplayIconUrl(comment);

  return (
    <div className="token-comment-author">
      <div className="token-comment-author__icon-wrap">
        {iconUrl ? (
          <img
            src={iconUrl}
            alt={displayName}
            className="token-comment-author__icon"
          />
        ) : (
          <span className="token-comment-author__icon-fallback">👤</span>
        )}
      </div>

      <span className="token-comment-author__name">{displayName}</span>
    </div>
  );
}

export default function TokenCommentItem({
  node,
  expandedIds,
  replyingCommentId,
  replyBody,
  replyPosting,
  onToggleExpanded,
  onLike,
  onDislike,
  onStartReply,
  onCancelReply,
  onReplyBodyChange,
  onSubmitReply,
}: TokenCommentItemProps) {
  const comment = node.comment;
  const isExpanded = expandedIds.has(comment.commentId);
  const isReplying = replyingCommentId === comment.commentId;
  const hasChildren = hasTokenCommentChildren(node);
  const indent = Math.min(Math.max(comment.depth * 16, 0), 48);

  const handleReplyBodyChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    onReplyBodyChange(event.target.value);
  };

  const handleLike = () => {
    if (comment.deleted) {
      return;
    }

    void onLike(comment.commentId);
  };

  const handleDislike = () => {
    if (comment.deleted) {
      return;
    }

    void onDislike(comment.commentId);
  };

  const handleStartReply = () => {
    if (comment.deleted) {
      return;
    }

    onStartReply(comment.commentId);

    if (!isExpanded) {
      onToggleExpanded(comment.commentId);
    }
  };

  const handleToggleExpanded = () => {
    onToggleExpanded(comment.commentId);
  };

  const handleSubmitReply = () => {
    if (!replyBody.trim() || replyPosting) {
      return;
    }

    void onSubmitReply(comment.commentId);
  };

  return (
    <div className="token-comment-item" style={{ marginLeft: `${indent}px` }}>
      <div className="token-comment-item__body">
        <div className="token-comment-item__header">
          <TokenCommentAuthor comment={comment} />

          {comment.createdAt ? (
            <time className="token-comment-item__date">
              {formatCommentDate(comment.createdAt)}
            </time>
          ) : null}
        </div>

        <p className="token-comment-item__text">
          {comment.deleted ? "このコメントは削除されました" : comment.body}
        </p>

        <div className="token-comment-item__actions">
          <button
            type="button"
            className="token-comment-item__action"
            disabled={comment.deleted}
            onClick={handleLike}
          >
            👍 {comment.likeCount}
          </button>

          <button
            type="button"
            className="token-comment-item__action"
            disabled={comment.deleted}
            onClick={handleDislike}
          >
            👎 {comment.dislikeCount}
          </button>

          <button
            type="button"
            className="token-comment-item__action"
            disabled={comment.deleted}
            onClick={handleStartReply}
          >
            返信
          </button>

          {hasChildren ? (
            <button
              type="button"
              className="token-comment-item__action"
              onClick={handleToggleExpanded}
            >
              {isExpanded
                ? "返信を閉じる"
                : `返信を表示 (${comment.childCount})`}
            </button>
          ) : (
            <span className="token-comment-item__reply-count">
              💬 {comment.childCount}
            </span>
          )}
        </div>

        {isReplying ? (
          <div className="token-comment-reply-form">
            <textarea
              className="token-comment-reply-form__textarea"
              value={replyBody}
              rows={3}
              disabled={replyPosting}
              placeholder="返信を書く…"
              onChange={handleReplyBodyChange}
            />

            <div className="token-comment-reply-form__actions">
              <button
                type="button"
                className="token-comment-reply-form__button token-comment-reply-form__button--secondary"
                disabled={replyPosting}
                onClick={onCancelReply}
              >
                キャンセル
              </button>

              <button
                type="button"
                className="token-comment-reply-form__button"
                disabled={replyPosting || !replyBody.trim()}
                onClick={handleSubmitReply}
              >
                {replyPosting ? "投稿中..." : "返信を投稿"}
              </button>
            </div>
          </div>
        ) : null}

        {isExpanded && node.children.length > 0 ? (
          <div className="token-comment-item__children">
            {node.children.map((child) => (
              <TokenCommentItem
                key={child.comment.commentId}
                node={child}
                expandedIds={expandedIds}
                replyingCommentId={replyingCommentId}
                replyBody={replyBody}
                replyPosting={replyPosting}
                onToggleExpanded={onToggleExpanded}
                onLike={onLike}
                onDislike={onDislike}
                onStartReply={onStartReply}
                onCancelReply={onCancelReply}
                onReplyBodyChange={onReplyBodyChange}
                onSubmitReply={onSubmitReply}
              />
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}