// frontend/mall/src/features/token-commnet/components/TokenCommentItem.tsx

import { useNavigate } from "react-router-dom";

import Card from "../../../components/ui/Card";
import Chip from "../../../components/ui/Chip";
import Media from "../../../components/ui/Media";
import { formatDateTime } from "../../../components/utils/date";
import ReportFlagButton from "../../shared/presentation/components/ReportFlagButton";
import type {
  TokenComment,
  TokenCommentTreeNode,
} from "../../shared/types/tokenCommentTypes";
import {
  getTokenCommentDisplayIconUrl,
  getTokenCommentDisplayName,
} from "../../shared/types/tokenCommentTypes";
import { hasTokenCommentChildren } from "../utils/commentTree";

type TokenCommentItemProps = {
  tokenBlueprintId: string;
  currentAvatarId: string;
  node: TokenCommentTreeNode;
  expandedIds: Set<string>;
  onToggleExpanded: (commentId: string) => void;
  onLike: (commentId: string) => void | Promise<void>;
  onDislike: (commentId: string) => void | Promise<void>;
  onStartReply: (commentId: string) => void;
  onReport: (commentId: string) => void;
};

function getAuthorAvatarId(comment: TokenComment): string {
  return comment.authorId?.trim() || "";
}

function TokenCommentAuthor({ comment }: { comment: TokenComment }) {
  const navigate = useNavigate();
  const displayName = getTokenCommentDisplayName(comment);
  const iconUrl = getTokenCommentDisplayIconUrl(comment);
  const avatarId = getAuthorAvatarId(comment);

  const handleOpenAvatar = () => {
    if (!avatarId) {
      return;
    }

    navigate(`/avatars/${encodeURIComponent(avatarId)}`);
  };

  const content = (
    <>
      <span className="token-comment-author__icon-wrap">
        {iconUrl ? (
          <Media
            src={iconUrl}
            alt={displayName}
            fit="cover"
            className="token-comment-author__icon"
          />
        ) : (
          <span className="token-comment-author__icon-fallback">👤</span>
        )}
      </span>

      <span className="token-comment-author__name">{displayName}</span>
    </>
  );

  if (!avatarId) {
    return <div className="token-comment-author">{content}</div>;
  }

  return (
    <button
      type="button"
      className="token-comment-author token-comment-author--button"
      onClick={handleOpenAvatar}
    >
      {content}
    </button>
  );
}

export default function TokenCommentItem({
  tokenBlueprintId,
  currentAvatarId,
  node,
  expandedIds,
  onToggleExpanded,
  onLike,
  onDislike,
  onStartReply,
  onReport,
}: TokenCommentItemProps) {
  const comment = node.comment;

  if (comment.deleted) {
    return null;
  }

  const commentId = comment.commentId?.trim() || "";
  const authorAvatarId = getAuthorAvatarId(comment);
  const displayName = getTokenCommentDisplayName(comment);
  const normalizedTokenBlueprintId = tokenBlueprintId.trim();
  const normalizedCurrentAvatarId = currentAvatarId.trim();
  const isExpanded = expandedIds.has(commentId);
  const hasChildren = hasTokenCommentChildren(node);
  const isOwnComment = Boolean(
    normalizedCurrentAvatarId &&
      authorAvatarId &&
      normalizedCurrentAvatarId === authorAvatarId,
  );
  const canReport = Boolean(
    normalizedTokenBlueprintId &&
      normalizedCurrentAvatarId &&
      commentId &&
      !isOwnComment,
  );
  const indent = Math.min(Math.max(comment.depth * 16, 0), 48);

  const handleLike = () => {
    if (!commentId) {
      return;
    }

    void onLike(commentId);
  };

  const handleDislike = () => {
    if (!commentId) {
      return;
    }

    void onDislike(commentId);
  };

  const handleStartReply = () => {
    if (!commentId) {
      return;
    }

    onStartReply(commentId);
  };

  const handleReport = () => {
    if (!canReport) {
      return;
    }

    onReport(commentId);
  };

  const handleToggleExpanded = () => {
    if (!commentId) {
      return;
    }

    onToggleExpanded(commentId);
  };

  return (
    <div
      className="token-comment-item"
      style={{ marginLeft: `${indent}px` }}
    >
      <Card padding="sm" className="token-comment-item__body">
        <div className="token-comment-item__header">
          <TokenCommentAuthor comment={comment} />

          {comment.createdAt ? (
            <time
              className="token-comment-item__date"
              dateTime={comment.createdAt}
            >
              {formatDateTime(comment.createdAt)}
            </time>
          ) : null}
        </div>

        <p className="token-comment-item__text">{comment.body}</p>

        <div className="token-comment-item__actions">
          <Chip size="sm" variant="neutral" onClick={handleLike}>
            👍 {comment.likeCount}
          </Chip>

          <Chip size="sm" variant="neutral" onClick={handleDislike}>
            👎 {comment.dislikeCount}
          </Chip>

          <Chip size="sm" variant="neutral" onClick={handleStartReply}>
            返信
          </Chip>

          {canReport ? (
            <ReportFlagButton
              label={`${displayName}のコメントを通報`}
              onClick={handleReport}
            />
          ) : null}

          {hasChildren ? (
            <Chip
              size="sm"
              variant="neutral"
              selected={isExpanded}
              onClick={handleToggleExpanded}
            >
              {isExpanded
                ? "返信を閉じる"
                : `返信を表示 (${comment.childCount})`}
            </Chip>
          ) : (
            <span className="token-comment-item__reply-count">
              💬 {comment.childCount}
            </span>
          )}
        </div>

        {isExpanded && node.children.length > 0 ? (
          <div className="token-comment-item__children">
            {node.children.map((child) => (
              <TokenCommentItem
                key={child.comment.commentId}
                tokenBlueprintId={normalizedTokenBlueprintId}
                currentAvatarId={normalizedCurrentAvatarId}
                node={child}
                expandedIds={expandedIds}
                onToggleExpanded={onToggleExpanded}
                onLike={onLike}
                onDislike={onDislike}
                onStartReply={onStartReply}
                onReport={onReport}
              />
            ))}
          </div>
        ) : null}
      </Card>
    </div>
  );
}