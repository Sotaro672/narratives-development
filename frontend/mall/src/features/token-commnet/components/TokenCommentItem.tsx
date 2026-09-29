// frontend/mall/src/features/token-commnet/components/TokenCommentItem.tsx

import { useNavigate } from "react-router-dom";

import Chip from "../../../components/ui/Chip";
import MediaIcon from "../../../components/ui/MediaIcon";
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

function TokenCommentAuthor({
  comment,
}: {
  comment: TokenComment;
}) {
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
      <MediaIcon
        src={iconUrl}
        alt={displayName}
        fallback="👤"
        size="xs"
        shape="circle"
      />
      <span className="token-comment-author__name">
        {displayName}
      </span>
    </>
  );

  if (!avatarId) {
    return (
      <div className="token-comment-author">
        {content}
      </div>
    );
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
  const hasReplies = comment.childCount > 0 || node.children.length > 0;

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

  const handleOpenReplySection = () => {
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

  return (
    <div className="token-comment-item">
      <article className="token-comment-item__body">
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

        <p className="token-comment-item__text">
          {comment.body}
        </p>

        <div className="token-comment-item__actions">
          <Chip
            size="sm"
            variant="neutral"
            onClick={handleLike}
          >
            👍 {comment.likeCount}
          </Chip>

          <Chip
            size="sm"
            variant="neutral"
            onClick={handleDislike}
          >
            👎 {comment.dislikeCount}
          </Chip>

          <Chip
            size="sm"
            variant="neutral"
            onClick={handleOpenReplySection}
          >
            返信
          </Chip>

          {canReport ? (
            <ReportFlagButton
              label={`${displayName}のコメントを通報`}
              onClick={handleReport}
            />
          ) : null}

          {hasReplies ? (
            <Chip
              size="sm"
              variant="neutral"
              onClick={handleOpenReplySection}
            >
              返信を表示 ({comment.childCount || node.children.length})
            </Chip>
          ) : null}
        </div>
      </article>
    </div>
  );
}