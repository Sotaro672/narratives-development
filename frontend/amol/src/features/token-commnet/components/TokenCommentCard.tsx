// frontend/amol/src/features/token-commnet/components/TokenCommentCard.tsx

import type {
  TokenCommentTreeNode,
} from "../types/tokenCommentTypes";

import TokenCommentForm from "./TokenCommentForm";
import TokenCommentList from "./TokenCommentList";

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

  onCommentBodyChange: (
    value: string,
  ) => void;
  onReplyBodyChange: (
    value: string,
  ) => void;
  onPostComment: () => Promise<void>;
  onToggleExpanded: (
    commentId: string,
  ) => void;
  onLikeComment: (
    commentId: string,
  ) => Promise<void>;
  onDislikeComment: (
    commentId: string,
  ) => Promise<void>;
  onStartReply: (
    commentId: string,
  ) => void;
  onCancelReply: () => void;
  onSubmitReply: (
    parentCommentId: string,
  ) => Promise<void>;
};

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
  onPostComment,
  onToggleExpanded,
  onLikeComment,
  onDislikeComment,
  onStartReply,
  onCancelReply,
  onSubmitReply,
}: TokenCommentCardProps) {
  return (
    <section
      className={[
        "token-comment-card",
        hideCommentForm
          ? "token-comment-card--hide-form"
          : "",
      ]
        .filter(Boolean)
        .join(" ")}
    >
      <div className="token-comment-card__header">
        <div className="token-comment-card__title-wrap">
          <span className="token-comment-card__icon">
            💬
          </span>

          <h2 className="token-comment-card__title">
            コメント
          </h2>
        </div>
      </div>

      {!tokenBlueprintId ? (
        <p className="token-comment-card__message">
          tokenBlueprintId
          未取得のためコメントを表示できません。
        </p>
      ) : (
        <>
          {!hideCommentForm ? (
            <TokenCommentForm
              value={commentBody}
              posting={posting}
              loading={loading}
              rows={1}
              onChange={onCommentBodyChange}
              onSubmit={onPostComment}
            />
          ) : null}

          {commentsError ? (
            <p
              className="token-comment-card__error"
              role="alert"
            >
              {commentsError}
            </p>
          ) : null}

          <TokenCommentList
            commentTree={commentTree}
            commentsLoading={commentsLoading}
            expandedIds={expandedIds}
            replyingCommentId={
              replyingCommentId
            }
            replyBody={replyBody}
            replyPosting={replyPosting}
            onToggleExpanded={
              onToggleExpanded
            }
            onLike={onLikeComment}
            onDislike={onDislikeComment}
            onStartReply={onStartReply}
            onCancelReply={onCancelReply}
            onReplyBodyChange={
              onReplyBodyChange
            }
            onSubmitReply={onSubmitReply}
          />
        </>
      )}
    </section>
  );
}