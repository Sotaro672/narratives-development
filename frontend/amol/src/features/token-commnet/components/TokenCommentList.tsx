// frontend/amol/src/features/token-commnet/components/TokenCommentList.tsx

import TokenCommentItem from "./TokenCommentItem";
import type { TokenCommentTreeNode } from "../types/tokenCommentTypes";

type TokenCommentListProps = {
  commentTree: TokenCommentTreeNode[];
  commentsLoading: boolean;
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

export default function TokenCommentList({
  commentTree,
  commentsLoading,
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
}: TokenCommentListProps) {
  if (commentsLoading && commentTree.length === 0) {
    return (
      <div className="token-comment-list">
        <p className="token-comment-card__message">
          コメントを読み込んでいます。
        </p>
      </div>
    );
  }

  if (!commentsLoading && commentTree.length === 0) {
    return (
      <div className="token-comment-list">
        <p className="token-comment-card__message">
          コメントはまだありません。
        </p>
      </div>
    );
  }

  return (
    <div className="token-comment-list">
      {commentTree.map((node) => (
        <TokenCommentItem
          key={node.comment.commentId}
          node={node}
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
  );
}