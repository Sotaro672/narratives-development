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
  editingCommentId: string | null;
  deletingCommentId: string | null;
  useDedicatedReplySection: boolean;
  onToggleExpanded: (commentId: string) => void;
  onLike: (commentId: string) => void | Promise<void>;
  onDislike: (commentId: string) => void | Promise<void>;
  onStartReply: (commentId: string) => void;
  onStartEdit: (commentId: string) => void;
  onDelete: (commentId: string) => void | Promise<void>;
  onReport: (commentId: string) => void;
};

function countDescendantReplies(node: TokenCommentTreeNode): number {
  return node.children.reduce(
    (total, childNode) =>
      total + 1 + countDescendantReplies(childNode),
    0,
  );
}

function getAuthorAvatarId(comment: TokenComment): string {
  if (comment.authorType !== "avatar") {
    return "";
  }

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
  expandedIds,
  editingCommentId,
  deletingCommentId,
  useDedicatedReplySection,
  onToggleExpanded,
  onLike,
  onDislike,
  onStartReply,
  onStartEdit,
  onDelete,
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
  const normalizedEditingCommentId = editingCommentId?.trim() ?? "";
  const normalizedDeletingCommentId = deletingCommentId?.trim() ?? "";
  const totalReplyCount = countDescendantReplies(node);
  const hasReplies = totalReplyCount > 0;
  const isExpanded = Boolean(commentId && expandedIds.has(commentId));

  const isOwnComment = Boolean(
    comment.authorType === "avatar" &&
      normalizedCurrentAvatarId &&
      authorAvatarId &&
      normalizedCurrentAvatarId === authorAvatarId,
  );

  const isEditing = Boolean(
    commentId &&
      normalizedEditingCommentId === commentId,
  );

  const isDeleting = Boolean(
    commentId &&
      normalizedDeletingCommentId === commentId,
  );

  const actionDisabled = isDeleting;

  const canReport = Boolean(
    normalizedTokenBlueprintId &&
      normalizedCurrentAvatarId &&
      commentId &&
      !isOwnComment,
  );

  const handleLike = () => {
    if (!commentId || actionDisabled) {
      return;
    }

    void onLike(commentId);
  };

  const handleDislike = () => {
    if (!commentId || actionDisabled) {
      return;
    }

    void onDislike(commentId);
  };

  const handleStartReply = () => {
    if (!commentId || actionDisabled) {
      return;
    }

    onStartReply(commentId);
  };

  const handleToggleReplies = () => {
    if (!commentId || actionDisabled) {
      return;
    }

    if (useDedicatedReplySection) {
      onStartReply(commentId);
      return;
    }

    onToggleExpanded(commentId);
  };

  const handleStartEdit = () => {
    if (!commentId || !isOwnComment || actionDisabled) {
      return;
    }

    onStartEdit(commentId);
  };

  const handleDelete = () => {
    if (!commentId || !isOwnComment || actionDisabled) {
      return;
    }

    void onDelete(commentId);
  };

  const handleReport = () => {
    if (!canReport || actionDisabled) {
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
            disabled={actionDisabled}
            onClick={handleLike}
          >
            👍 {comment.likeCount}
          </Chip>

          <Chip
            size="sm"
            variant="neutral"
            disabled={actionDisabled}
            onClick={handleDislike}
          >
            👎 {comment.dislikeCount}
          </Chip>

          <Chip
            size="sm"
            variant="neutral"
            disabled={actionDisabled}
            onClick={handleStartReply}
          >
            返信
          </Chip>

          {isOwnComment ? (
            <>
              <Chip
                size="sm"
                variant="neutral"
                selected={isEditing}
                disabled={actionDisabled}
                onClick={handleStartEdit}
              >
                編集
              </Chip>

              <Chip
                size="sm"
                variant="danger"
                disabled={actionDisabled}
                onClick={handleDelete}
              >
                {isDeleting ? "削除中..." : "削除"}
              </Chip>
            </>
          ) : canReport ? (
            <ReportFlagButton
              disabled={actionDisabled}
              label={`${displayName}のコメントを通報`}
              onClick={handleReport}
            />
          ) : null}

          {hasReplies ? (
            <Chip
              size="sm"
              variant="neutral"
              selected={!useDedicatedReplySection && isExpanded}
              disabled={actionDisabled}
              onClick={handleToggleReplies}
            >
              {!useDedicatedReplySection && isExpanded
                ? "返信を非表示"
                : `返信を表示 (${totalReplyCount})`}
            </Chip>
          ) : null}
        </div>
      </article>

      {!useDedicatedReplySection &&
      isExpanded &&
      node.children.length > 0 ? (
        <div className="token-comment-item__children">
          {node.children.map((childNode) => (
            <TokenCommentItem
              key={childNode.comment.commentId}
              tokenBlueprintId={tokenBlueprintId}
              currentAvatarId={currentAvatarId}
              node={childNode}
              expandedIds={expandedIds}
              editingCommentId={editingCommentId}
              deletingCommentId={deletingCommentId}
              useDedicatedReplySection={useDedicatedReplySection}
              onToggleExpanded={onToggleExpanded}
              onLike={onLike}
              onDislike={onDislike}
              onStartReply={onStartReply}
              onStartEdit={onStartEdit}
              onDelete={onDelete}
              onReport={onReport}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}