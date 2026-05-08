// frontend/amol/src/features/token-commnet/components/TokenCommentCard.tsx

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

type TokenCommentCardProps = {
  tokenBlueprintId: string;
  loading?: boolean;
  hideCommentForm?: boolean;

  commentTree: TokenCommentTreeNode[];
  commentsLoading: boolean;
  commentsError: string;
  posting: boolean;
  commentBody: string;
  expandedIds: Set<string>;
  replyingCommentId: string | null;
  replyBody: string;
  replyPosting: boolean;

  onCommentBodyChange: (value: string) => void;
  onReplyBodyChange: (value: string) => void;
  onRefreshComments: () => Promise<void>;
  onPostComment: () => Promise<void>;
  onToggleExpanded: (commentId: string) => void;
  onLikeComment: (commentId: string) => Promise<void>;
  onDislikeComment: (commentId: string) => Promise<void>;
  onStartReply: (commentId: string) => void;
  onCancelReply: () => void;
  onSubmitReply: (parentCommentId: string) => Promise<void>;
};

type TokenCommentItemProps = {
  node: TokenCommentTreeNode;
  expandedIds: Set<string>;
  replyingCommentId: string | null;
  replyBody: string;
  replyPosting: boolean;
  onToggleExpanded: (commentId: string) => void;
  onLike: (commentId: string) => Promise<void>;
  onDislike: (commentId: string) => Promise<void>;
  onStartReply: (commentId: string) => void;
  onCancelReply: () => void;
  onReplyBodyChange: (value: string) => void;
  onSubmitReply: (parentCommentId: string) => Promise<void>;
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

function TokenCommentItem({
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
              onClick={() => onToggleExpanded(comment.commentId)}
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

export default function TokenCommentCard({
  tokenBlueprintId,
  loading = false,
  hideCommentForm = false,
  commentTree,
  commentsLoading,
  commentsError,
  posting,
  commentBody,
  expandedIds,
  replyingCommentId,
  replyBody,
  replyPosting,
  onCommentBodyChange,
  onReplyBodyChange,
  onRefreshComments,
  onPostComment,
  onToggleExpanded,
  onLikeComment,
  onDislikeComment,
  onStartReply,
  onCancelReply,
  onSubmitReply,
}: TokenCommentCardProps) {
  const handleCommentBodyChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    onCommentBodyChange(event.target.value);
  };

  return (
    <section
      className={[
        "token-comment-card",
        hideCommentForm ? "token-comment-card--hide-form" : "",
      ]
        .filter(Boolean)
        .join(" ")}
    >
      <div className="token-comment-card__header">
        <div className="token-comment-card__title-wrap">
          <span className="token-comment-card__icon">💬</span>
          <h2 className="token-comment-card__title">コメント</h2>
        </div>

        <button
          type="button"
          className="token-comment-card__refresh-button"
          disabled={!tokenBlueprintId || commentsLoading}
          onClick={() => void onRefreshComments()}
        >
          {commentsLoading ? "更新中..." : "更新"}
        </button>
      </div>

      {!tokenBlueprintId ? (
        <p className="token-comment-card__message">
          tokenBlueprintId 未取得のためコメントを表示できません。
        </p>
      ) : (
        <>
          {!hideCommentForm ? (
            <div className="token-comment-form">
              <textarea
                className="token-comment-form__textarea"
                value={commentBody}
                rows={4}
                disabled={posting || loading}
                placeholder="コメントを書く…"
                onChange={handleCommentBodyChange}
              />

              <button
                type="button"
                className="token-comment-form__button"
                disabled={posting || loading || !commentBody.trim()}
                onClick={() => void onPostComment()}
              >
                {posting ? "投稿中..." : "投稿"}
              </button>
            </div>
          ) : null}

          {commentsError ? (
            <p className="token-comment-card__error" role="alert">
              {commentsError}
            </p>
          ) : null}

          <div className="token-comment-list">
            {commentsLoading && commentTree.length === 0 ? (
              <p className="token-comment-card__message">
                コメントを読み込んでいます。
              </p>
            ) : null}

            {!commentsLoading && commentTree.length === 0 ? (
              <p className="token-comment-card__message">
                コメントはまだありません。
              </p>
            ) : null}

            {commentTree.map((node) => (
              <TokenCommentItem
                key={node.comment.commentId}
                node={node}
                expandedIds={expandedIds}
                replyingCommentId={replyingCommentId}
                replyBody={replyBody}
                replyPosting={replyPosting}
                onToggleExpanded={onToggleExpanded}
                onLike={onLikeComment}
                onDislike={onDislikeComment}
                onStartReply={onStartReply}
                onCancelReply={onCancelReply}
                onReplyBodyChange={onReplyBodyChange}
                onSubmitReply={onSubmitReply}
              />
            ))}
          </div>
        </>
      )}
    </section>
  );
}