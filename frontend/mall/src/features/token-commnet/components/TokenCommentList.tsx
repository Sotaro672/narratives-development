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
  onToggleExpanded: (commentId: string) => void;
  onLike: (commentId: string) => void | Promise<void>;
  onDislike: (commentId: string) => void | Promise<void>;
  onStartReply: (commentId: string) => void;
  onReport: (commentId: string) => void;
};

export default function TokenCommentList({
  tokenBlueprintId,
  currentAvatarId,
  commentTree,
  commentsLoading,
  expandedIds,
  onToggleExpanded,
  onLike,
  onDislike,
  onStartReply,
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
          onToggleExpanded={onToggleExpanded}
          onLike={onLike}
          onDislike={onDislike}
          onStartReply={onStartReply}
          onReport={onReport}
        />
      ))}
    </div>
  );
}