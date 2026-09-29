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
  editingCommentId: string | null;
  deletingCommentId: string | null;
  onToggleExpanded: (commentId: string) => void;
  onLike: (commentId: string) => void | Promise<void>;
  onDislike: (commentId: string) => void | Promise<void>;
  onStartReply: (commentId: string) => void;
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
  editingCommentId,
  deletingCommentId,
  onToggleExpanded,
  onLike,
  onDislike,
  onStartReply,
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
          editingCommentId={editingCommentId}
          deletingCommentId={deletingCommentId}
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
  );
}