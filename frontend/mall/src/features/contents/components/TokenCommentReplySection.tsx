// frontend/mall/src/features/contents/components/TokenCommentReplySection.tsx

import { useEffect, useState } from "react";
import { ChevronLeft } from "lucide-react";
import { useNavigate } from "react-router-dom";

import Alert from "../../../components/ui/Alert";
import Chip from "../../../components/ui/Chip";
import IconButton from "../../../components/ui/IconButton";
import MediaIcon from "../../../components/ui/MediaIcon";
import TextState from "../../../components/ui/TextState";
import { formatDateTime } from "../../../components/utils/date";
import { getMyAvatar } from "../../avatar/api/avatarApi";
import ReportModal from "../../report/components/ReportModal";
import { useReport } from "../../report/hooks/useReport";
import { useAuthState } from "../../shared/hooks/useAuthState";
import ReportFlagButton from "../../shared/presentation/components/ReportFlagButton";
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
  editingCommentId: string | null;
  editSaving: boolean;
  deletingCommentId: string | null;
  onBack: () => void;
  onReplyBodyChange: (value: string) => void;
  onSubmitReply: (commentId: string) => Promise<void>;
  onToggleExpanded: (commentId: string) => void;
  onLike: (commentId: string) => void | Promise<void>;
  onDislike: (commentId: string) => void | Promise<void>;
  onStartReply: (commentId: string) => void;
  onStartEdit: (commentId: string) => void;
  onCancelEdit: () => void;
  onDelete: (commentId: string) => void | Promise<void>;
};

type ReplyCommentProps = {
  comment: TokenComment;
  visualDepth?: 0 | 1;
  replyToName?: string;
  tokenBlueprintId: string;
  currentAvatarId: string;
  editingCommentId: string | null;
  deletingCommentId: string | null;
  disabled?: boolean;
  isReplyTarget?: boolean;
  onLike: (commentId: string) => void | Promise<void>;
  onDislike: (commentId: string) => void | Promise<void>;
  onStartReply: (commentId: string) => void;
  onStartEdit: (commentId: string) => void;
  onDelete: (commentId: string) => void | Promise<void>;
  onReport: (commentId: string) => void;
};

type FlattenedReply = {
  comment: TokenComment;
  parentComment: TokenComment;
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

function flattenReplyNodes(
  nodes: TokenCommentTreeNode[],
  parentComment: TokenComment,
): FlattenedReply[] {
  const result: FlattenedReply[] = [];

  for (const node of nodes) {
    result.push({
      comment: node.comment,
      parentComment,
    });

    if (node.children.length > 0) {
      result.push(
        ...flattenReplyNodes(
          node.children,
          node.comment,
        ),
      );
    }
  }

  return result;
}

function getAuthorAvatarId(comment: TokenComment): string {
  if (comment.authorType !== "avatar") {
    return "";
  }

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
  visualDepth = 0,
  replyToName = "",
  tokenBlueprintId,
  currentAvatarId,
  editingCommentId,
  deletingCommentId,
  disabled = false,
  isReplyTarget = false,
  onLike,
  onDislike,
  onStartReply,
  onStartEdit,
  onDelete,
  onReport,
}: ReplyCommentProps) {
  const commentId = comment.commentId?.trim() ?? "";
  const normalizedReplyToName = replyToName.trim();
  const normalizedTokenBlueprintId = tokenBlueprintId.trim();
  const normalizedCurrentAvatarId = currentAvatarId.trim();
  const normalizedEditingCommentId = editingCommentId?.trim() ?? "";
  const normalizedDeletingCommentId = deletingCommentId?.trim() ?? "";
  const authorAvatarId = getAuthorAvatarId(comment);
  const displayName = getTokenCommentDisplayName(comment);

  if (!commentId) {
    return null;
  }

  const indent = visualDepth === 1 ? 24 : 0;

  if (comment.deleted) {
    return (
      <article
        className="token-comment-item token-comment-reply-section__comment token-comment-reply-section__comment--deleted"
        data-token-comment-id={commentId}
        data-reply-depth={visualDepth}
        data-deleted="true"
        style={{ marginLeft: `${indent}px` }}
      >
        <p className="token-comment-reply-section__deleted">
          このコメントは削除されました。
        </p>
      </article>
    );
  }

  const isOwnComment = Boolean(
    comment.authorType === "avatar" &&
      normalizedCurrentAvatarId &&
      authorAvatarId &&
      normalizedCurrentAvatarId === authorAvatarId,
  );

  const isEditing = Boolean(
    normalizedEditingCommentId &&
      normalizedEditingCommentId === commentId,
  );

  const isDeleting = Boolean(
    normalizedDeletingCommentId &&
      normalizedDeletingCommentId === commentId,
  );

  const actionDisabled = disabled || isDeleting;

  const canReport = Boolean(
    normalizedTokenBlueprintId &&
      normalizedCurrentAvatarId &&
      commentId &&
      !isOwnComment,
  );

  const showReplyMention =
    comment.depth >= 2 &&
    Boolean(normalizedReplyToName);

  const handleLike = () => {
    if (actionDisabled) {
      return;
    }

    void onLike(commentId);
  };

  const handleDislike = () => {
    if (actionDisabled) {
      return;
    }

    void onDislike(commentId);
  };

  const handleStartReply = () => {
    if (actionDisabled) {
      return;
    }

    onStartReply(commentId);
  };

  const handleStartEdit = () => {
    if (
      actionDisabled ||
      !isOwnComment
    ) {
      return;
    }

    onStartEdit(commentId);
  };

  const handleDelete = () => {
    if (
      actionDisabled ||
      !isOwnComment
    ) {
      return;
    }

    void onDelete(commentId);
  };

  const handleReport = () => {
    if (
      actionDisabled ||
      !canReport
    ) {
      return;
    }

    onReport(commentId);
  };

  return (
    <article
      className="token-comment-item token-comment-reply-section__comment"
      data-token-comment-id={commentId}
      data-reply-target={isReplyTarget ? "true" : undefined}
      data-reply-depth={visualDepth}
      data-editing={isEditing ? "true" : undefined}
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
        {showReplyMention ? (
          <>
            <span className="token-comment-reply-section__mention">
              @{normalizedReplyToName}
            </span>{" "}
          </>
        ) : null}
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
          selected={isReplyTarget}
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
      </div>
    </article>
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
    editingCommentId,
    editSaving,
    deletingCommentId,
    onBack,
    onLike,
    onDislike,
    onStartReply,
    onStartEdit,
    onCancelEdit,
    onDelete,
  } = props;

  const { authResolved, isLoggedIn } = useAuthState();
  const [currentAvatarId, setCurrentAvatarId] = useState("");

  const {
    target,
    isOpen,
    reason,
    detail,
    submitting,
    error: reportError,
    result,
    canSubmit,
    openTokenBlueprintCommentReport,
    close: closeReport,
    setReason,
    setDetail,
    submit,
  } = useReport();

  const normalizedCommentId = commentId.trim();
  const normalizedReplyingCommentId = replyingCommentId?.trim() ?? "";
  const normalizedEditingCommentId = editingCommentId?.trim() ?? "";
  const normalizedTokenBlueprintId = tokenBlueprintId.trim();

  const targetNode = normalizedCommentId
    ? findCommentNode(
        commentTree,
        normalizedCommentId,
        normalizedTokenBlueprintId,
      )
    : null;

  const replyingTargetNode = normalizedReplyingCommentId
    ? findCommentNode(
        commentTree,
        normalizedReplyingCommentId,
        normalizedTokenBlueprintId,
      )
    : null;

  const replyingTargetComment = replyingTargetNode?.comment ?? null;

  const showReplyingTo =
    Boolean(replyingTargetComment) &&
    (replyingTargetComment?.depth ?? 0) >= 1;

  const replyingToDisplayName =
    showReplyingTo && replyingTargetComment
      ? getTokenCommentDisplayName(replyingTargetComment)
      : "";

  const flattenedReplies = targetNode
    ? flattenReplyNodes(
        targetNode.children,
        targetNode.comment,
      )
    : [];

  const replyCount =
    targetNode?.comment.childCount ??
    targetNode?.children.length ??
    0;

  const interactionBusy =
    replyPosting ||
    editSaving ||
    Boolean(deletingCommentId);

  useEffect(() => {
    let cancelled = false;

    async function loadCurrentAvatar() {
      if (!authResolved || !isLoggedIn) {
        setCurrentAvatarId("");
        return;
      }

      try {
        const avatar = await getMyAvatar();

        if (cancelled) {
          return;
        }

        setCurrentAvatarId(
          avatar?.avatarId?.trim() ?? "",
        );
      } catch {
        if (!cancelled) {
          setCurrentAvatarId("");
        }
      }
    }

    void loadCurrentAvatar();

    return () => {
      cancelled = true;
    };
  }, [
    authResolved,
    isLoggedIn,
  ]);

  const handleReportComment = (targetCommentId: string) => {
    const normalizedTargetCommentId =
      targetCommentId.trim();

    if (
      !isLoggedIn ||
      !currentAvatarId ||
      !normalizedTokenBlueprintId ||
      !normalizedTargetCommentId
    ) {
      return;
    }

    openTokenBlueprintCommentReport({
      tokenBlueprintId: normalizedTokenBlueprintId,
      commentId: normalizedTargetCommentId,
    });
  };

  const handleDeleteComment = async (
    targetCommentId: string,
  ) => {
    const normalizedTargetCommentId =
      targetCommentId.trim();

    if (
      !normalizedTargetCommentId ||
      deletingCommentId
    ) {
      return;
    }

    const confirmed = window.confirm(
      "このコメントを削除します。よろしいですか？",
    );

    if (!confirmed) {
      return;
    }

    await onDelete(
      normalizedTargetCommentId,
    );
  };

  const handleBack = () => {
    if (interactionBusy) {
      return;
    }

    if (normalizedEditingCommentId) {
      onCancelEdit();
    }

    onBack();
  };

  return (
    <>
      <section
        className="token-comment-reply-section"
        aria-label="返信"
        aria-busy={interactionBusy || undefined}
        data-mobile-swipe-dismiss-ignore="true"
      >
        <header className="token-comment-reply-section__header">
          <IconButton
            type="button"
            variant="ghost"
            size="md"
            className="token-comment-reply-section__back"
            aria-label="返信を閉じる"
            disabled={interactionBusy}
            onClick={handleBack}
          >
            <ChevronLeft
              size={24}
              strokeWidth={2.25}
              aria-hidden="true"
            />
          </IconButton>

          <div className="token-comment-reply-section__heading">
            <div className="token-comment-reply-section__heading-main">
              <span className="token-comment-reply-section__title">
                返信
              </span>

              {targetNode ? (
                <span className="token-comment-reply-section__count">
                  {replyCount}件
                </span>
              ) : null}
            </div>

            {normalizedEditingCommentId ? (
              <span className="token-comment-reply-section__replying-to">
                コメントを編集中
              </span>
            ) : replyingToDisplayName ? (
              <span className="token-comment-reply-section__replying-to">
                {replyingToDisplayName}に返信しています
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
              visualDepth={0}
              tokenBlueprintId={normalizedTokenBlueprintId}
              currentAvatarId={currentAvatarId}
              editingCommentId={editingCommentId}
              deletingCommentId={deletingCommentId}
              disabled={replyPosting || editSaving}
              isReplyTarget={
                normalizedReplyingCommentId ===
                targetNode.comment.commentId?.trim()
              }
              onLike={onLike}
              onDislike={onDislike}
              onStartReply={onStartReply}
              onStartEdit={onStartEdit}
              onDelete={handleDeleteComment}
              onReport={handleReportComment}
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
          flattenedReplies.length > 0 ? (
            <div className="token-comment-reply-section__list">
              {flattenedReplies.map(
                (
                  {
                    comment,
                    parentComment,
                  },
                  index,
                ) => {
                  const replyCommentId =
                    comment.commentId?.trim() ?? "";

                  const replyToName =
                    comment.depth >= 2
                      ? getTokenCommentDisplayName(
                          parentComment,
                        )
                      : "";

                  return (
                    <div
                      key={
                        replyCommentId ||
                        `${parentComment.commentId}-${index}`
                      }
                      className="token-comment-reply-section__reply-node"
                    >
                      <ReplyComment
                        comment={comment}
                        visualDepth={1}
                        replyToName={replyToName}
                        tokenBlueprintId={normalizedTokenBlueprintId}
                        currentAvatarId={currentAvatarId}
                        editingCommentId={editingCommentId}
                        deletingCommentId={deletingCommentId}
                        disabled={replyPosting || editSaving}
                        isReplyTarget={
                          normalizedReplyingCommentId ===
                          replyCommentId
                        }
                        onLike={onLike}
                        onDislike={onDislike}
                        onStartReply={onStartReply}
                        onStartEdit={onStartEdit}
                        onDelete={handleDeleteComment}
                        onReport={handleReportComment}
                      />
                    </div>
                  );
                },
              )}
            </div>
          ) : null}

          {targetNode &&
          !commentsLoading &&
          flattenedReplies.length === 0 ? (
            <TextState
              variant="empty"
              className="token-comment-reply-section__empty"
            >
              返信はまだありません。
            </TextState>
          ) : null}

          {commentsLoading &&
          targetNode &&
          flattenedReplies.length === 0 ? (
            <TextState
              variant="loading"
              className="token-comment-reply-section__loading"
            >
              返信を読み込んでいます。
            </TextState>
          ) : null}
        </div>
      </section>

      <ReportModal
        open={isOpen}
        targetType={target?.type}
        reason={reason}
        detail={detail}
        submitting={submitting}
        error={reportError}
        result={result}
        canSubmit={canSubmit}
        onReasonChange={setReason}
        onDetailChange={setDetail}
        onSubmit={submit}
        onClose={closeReport}
      />
    </>
  );
}