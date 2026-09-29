// frontend/mall/src/features/contents/components/TokenCommentReplySection.tsx

import { ChevronLeft } from "lucide-react";
import { useNavigate } from "react-router-dom";

import Alert from "../../../components/ui/Alert";
import Chip from "../../../components/ui/Chip";
import IconButton from "../../../components/ui/IconButton";
import MediaIcon from "../../../components/ui/MediaIcon";
import TextState from "../../../components/ui/TextState";
import { formatDateTime } from "../../../components/utils/date";
import type {
  TokenComment,
  TokenCommentTreeNode,
} from "../../shared/types/tokenCommentTypes";
import {
  getTokenCommentDisplayIconUrl,
  getTokenCommentDisplayName,
} from "../../shared/types/tokenCommentTypes";

import "../../token-commnet/styles/token-comment-reply-section.css";

type TokenCommentReplySectionProps = {
  commentId: string;
  replyingCommentId: string | null;
  tokenBlueprintId: string;
  commentTree: TokenCommentTreeNode[];
  expandedIds: Set<string>;
  replyBody: string;
  replyPosting: boolean;
  commentsLoading: boolean;
  commentsError: string;
  onBack: () => void;
  onReplyBodyChange: (value: string) => void;
  onSubmitReply: (commentId: string) => Promise<void>;
  onToggleExpanded: (commentId: string) => void;
  onLike: (commentId: string) => void | Promise<void>;
  onDislike: (commentId: string) => void | Promise<void>;
  onStartReply: (commentId: string) => void;
};

type ReplyCommentProps = {
  comment: TokenComment;
  depth?: number;
  disabled?: boolean;
  isReplyTarget?: boolean;
  onLike: (commentId: string) => void | Promise<void>;
  onDislike: (commentId: string) => void | Promise<void>;
  onStartReply: (commentId: string) => void;
};

function findCommentNode(
  nodes: TokenCommentTreeNode[],
  commentId: string,
  tokenBlueprintId: string,
): TokenCommentTreeNode | null {
  for (const node of nodes) {
    const nodeCommentId = node.comment.commentId?.trim() ?? "";
    const nodeTokenBlueprintId = node.comment.tokenBlueprintId?.trim() ?? "";

    if (
      nodeCommentId === commentId &&
      (
        !tokenBlueprintId ||
        !nodeTokenBlueprintId ||
        nodeTokenBlueprintId === tokenBlueprintId
      )
    ) {
      return node;
    }

    const nested = findCommentNode(
      node.children,
      commentId,
      tokenBlueprintId,
    );

    if (nested) {
      return nested;
    }
  }

  return null;
}

function getAuthorAvatarId(comment: TokenComment): string {
  return comment.authorId?.trim() ?? "";
}

function ReplyCommentAuthor({
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

function ReplyComment({
  comment,
  depth = 0,
  disabled = false,
  isReplyTarget = false,
  onLike,
  onDislike,
  onStartReply,
}: ReplyCommentProps) {
  const commentId = comment.commentId?.trim() ?? "";

  if (
    comment.deleted ||
    !commentId
  ) {
    return null;
  }

  const handleLike = () => {
    if (disabled) {
      return;
    }

    void onLike(commentId);
  };

  const handleDislike = () => {
    if (disabled) {
      return;
    }

    void onDislike(commentId);
  };

  const handleStartReply = () => {
    if (disabled) {
      return;
    }

    onStartReply(commentId);
  };

  const indent = Math.min(
    Math.max(depth * 24, 0),
    48,
  );

  return (
    <article
      className="token-comment-item token-comment-reply-section__comment"
      data-token-comment-id={commentId}
      data-reply-target={isReplyTarget ? "true" : undefined}
      data-reply-depth={depth}
      style={{ marginLeft: `${indent}px` }}
    >
      <div className="token-comment-item__header">
        <ReplyCommentAuthor comment={comment} />

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
          disabled={disabled}
          onClick={handleLike}
        >
          👍 {comment.likeCount}
        </Chip>

        <Chip
          size="sm"
          variant="neutral"
          disabled={disabled}
          onClick={handleDislike}
        >
          👎 {comment.dislikeCount}
        </Chip>

        <Chip
          size="sm"
          variant="neutral"
          selected={isReplyTarget}
          disabled={disabled}
          onClick={handleStartReply}
        >
          返信
        </Chip>
      </div>
    </article>
  );
}

function ReplyTree({
  nodes,
  depth,
  disabled,
  replyingCommentId,
  onLike,
  onDislike,
  onStartReply,
}: {
  nodes: TokenCommentTreeNode[];
  depth: number;
  disabled: boolean;
  replyingCommentId: string | null;
  onLike: (commentId: string) => void | Promise<void>;
  onDislike: (commentId: string) => void | Promise<void>;
  onStartReply: (commentId: string) => void;
}) {
  return (
    <>
      {nodes.map((node) => {
        const commentId = node.comment.commentId?.trim() ?? "";

        if (
          node.comment.deleted ||
          !commentId
        ) {
          return null;
        }

        return (
          <div
            key={commentId}
            className="token-comment-reply-section__reply-node"
          >
            <ReplyComment
              comment={node.comment}
              depth={depth}
              disabled={disabled}
              isReplyTarget={replyingCommentId === commentId}
              onLike={onLike}
              onDislike={onDislike}
              onStartReply={onStartReply}
            />

            {node.children.length > 0 ? (
              <ReplyTree
                nodes={node.children}
                depth={depth + 1}
                disabled={disabled}
                replyingCommentId={replyingCommentId}
                onLike={onLike}
                onDislike={onDislike}
                onStartReply={onStartReply}
              />
            ) : null}
          </div>
        );
      })}
    </>
  );
}

export default function TokenCommentReplySection(
  props: TokenCommentReplySectionProps,
) {
  const {
    commentId,
    replyingCommentId,
    tokenBlueprintId,
    commentTree,
    replyPosting,
    commentsLoading,
    commentsError,
    onBack,
    onLike,
    onDislike,
    onStartReply,
  } = props;

  const normalizedCommentId = commentId.trim();
  const normalizedReplyingCommentId = replyingCommentId?.trim() ?? "";
  const normalizedTokenBlueprintId = tokenBlueprintId.trim();

  const targetNode = normalizedCommentId
    ? findCommentNode(
        commentTree,
        normalizedCommentId,
        normalizedTokenBlueprintId,
      )
    : null;

  const replyCount =
    targetNode?.comment.childCount ??
    targetNode?.children.length ??
    0;

  return (
    <section
      className="token-comment-reply-section"
      aria-label="返信"
      aria-busy={replyPosting || undefined}
      data-mobile-swipe-dismiss-ignore="true"
    >
      <header className="token-comment-reply-section__header">
        <IconButton
          type="button"
          variant="ghost"
          size="md"
          className="token-comment-reply-section__back"
          aria-label="返信を閉じる"
          disabled={replyPosting}
          onClick={onBack}
        >
          <ChevronLeft
            size={24}
            strokeWidth={2.25}
            aria-hidden="true"
          />
        </IconButton>

        <div className="token-comment-reply-section__heading">
          <span className="token-comment-reply-section__title">
            返信
          </span>

          {targetNode ? (
            <span className="token-comment-reply-section__count">
              {replyCount}件
            </span>
          ) : null}
        </div>
      </header>

      <div className="token-comment-reply-section__parent">
        {commentsLoading && !targetNode ? (
          <TextState variant="loading">
            コメントを読み込んでいます。
          </TextState>
        ) : null}

        {!commentsLoading && !targetNode ? (
          <TextState variant="error">
            返信対象のコメントが見つかりません。
          </TextState>
        ) : null}

        {targetNode ? (
          <ReplyComment
            comment={targetNode.comment}
            depth={0}
            disabled={replyPosting}
            isReplyTarget={
              normalizedReplyingCommentId ===
              targetNode.comment.commentId?.trim()
            }
            onLike={onLike}
            onDislike={onDislike}
            onStartReply={onStartReply}
          />
        ) : null}
      </div>

      <div className="token-comment-reply-section__scroll">
        {commentsError ? (
          <Alert variant="error">
            {commentsError}
          </Alert>
        ) : null}

        {targetNode &&
        targetNode.children.length > 0 ? (
          <div className="token-comment-reply-section__list">
            <ReplyTree
              nodes={targetNode.children}
              depth={1}
              disabled={replyPosting}
              replyingCommentId={normalizedReplyingCommentId || null}
              onLike={onLike}
              onDislike={onDislike}
              onStartReply={onStartReply}
            />
          </div>
        ) : null}

        {targetNode &&
        !commentsLoading &&
        targetNode.children.length === 0 ? (
          <TextState
            variant="empty"
            className="token-comment-reply-section__empty"
          >
            返信はまだありません。
          </TextState>
        ) : null}

        {commentsLoading &&
        targetNode &&
        targetNode.children.length === 0 ? (
          <TextState
            variant="loading"
            className="token-comment-reply-section__loading"
          >
            返信を読み込んでいます。
          </TextState>
        ) : null}
      </div>
    </section>
  );
}