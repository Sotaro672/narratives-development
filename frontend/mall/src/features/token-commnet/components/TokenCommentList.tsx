// frontend/mall/src/features/token-commnet/components/TokenCommentList.tsx

import TextState from "../../../components/ui/TextState";
import type { TokenCommentTreeNode } from "../../shared/types/tokenCommentTypes";
import TokenCommentItem from "./TokenCommentItem";

type TokenCommentListProps = {
  tokenBlueprintId: string;
  currentAvatarId: string;
  commentTree: TokenCommentTreeNode[];
  commentsLoading: boolean;
  expandedIds: Set<string>;
  replyingCommentId: string | null;
  replyBody: string;
  replyPosting: boolean;
  editingCommentId: string | null;
  deletingCommentId: string | null;
  useDedicatedReplySection: boolean;
  onToggleExpanded: (commentId: string) => void;
  onLike: (commentId: string) => void | Promise<void>;
  onDislike: (commentId: string) => void | Promise<void>;
  onStartReply: (commentId: string) => void;
  onReplyBodyChange: (value: string) => void;
  onSubmitReply: (parentCommentId: string) => Promise<void>;
  onStartEdit: (commentId: string) => void;
  onDelete: (commentId: string) => void | Promise<void>;
  onReport: (commentId: string) => void;
};

export default function TokenCommentList({
  tokenBlueprintId,
  currentAvatarId,
  commentTree,
  commentsLoading,
  expandedIds,
  replyingCommentId,
  replyBody,
  replyPosting,
  editingCommentId,
  deletingCommentId,
  useDedicatedReplySection,
  onToggleExpanded,
  onLike,
  onDislike,
  onStartReply,
  onReplyBodyChange,
  onSubmitReply,
  onStartEdit,
  onDelete,
  onReport,
}: TokenCommentListProps) {
  if (commentsLoading && commentTree.length === 0) {
    return (
      <div className="token-comment-list">
        <TextState variant="loading">
          コメントを読み込んでいます。
        </TextState>
      </div>
    );
  }

  if (!commentsLoading && commentTree.length === 0) {
    return (
      <div className="token-comment-list">
        <TextState variant="empty">
          コメントはまだありません。
        </TextState>
      </div>
    );
  }

  return (
    <div className="token-comment-list">
      {commentTree.map((node) => (
        <TokenCommentItem
          key={node.comment.commentId}
          tokenBlueprintId={tokenBlueprintId}
          currentAvatarId={currentAvatarId}
          node={node}
          expandedIds={expandedIds}
          replyingCommentId={replyingCommentId}
          replyBody={replyBody}
          replyPosting={replyPosting}
          editingCommentId={editingCommentId}
          deletingCommentId={deletingCommentId}
          useDedicatedReplySection={useDedicatedReplySection}
          onToggleExpanded={onToggleExpanded}
          onLike={onLike}
          onDislike={onDislike}
          onStartReply={onStartReply}
          onReplyBodyChange={onReplyBodyChange}
          onSubmitReply={onSubmitReply}
          onStartEdit={onStartEdit}
          onDelete={onDelete}
          onReport={onReport}
        />
      ))}
    </div>
  );
}