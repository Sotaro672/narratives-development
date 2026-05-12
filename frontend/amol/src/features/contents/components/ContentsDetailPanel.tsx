//frontend\amol\src\features\contents\components\ContentsDetailPanel.tsx
import TokenCommentCard from "../../token-commnet/components/TokenCommentCard";
import type {
  ContentsSearchParams,
  TokenCommentCardController,
} from "../types";
import ContentsTokenSummaryCard from "./ContentsTokenSummaryCard";

type ContentsDetailPanelProps = {
  contents: ContentsSearchParams;
  tokenName: string;
  tokenIconUrl: string;
  loading: boolean;
  loadingAvatarId: boolean;
  currentAvatarId: string;
  isMobilePortrait: boolean;
  commentCard: TokenCommentCardController;
  onProductNameClick: () => void;
};

export default function ContentsDetailPanel({
  contents,
  tokenName,
  tokenIconUrl,
  loading,
  loadingAvatarId,
  currentAvatarId,
  isMobilePortrait,
  commentCard,
  onProductNameClick,
}: ContentsDetailPanelProps) {
  return (
    <div className="split-page-right contents-page-detail">
      <ContentsTokenSummaryCard
        contents={contents}
        tokenName={tokenName}
        tokenIconUrl={tokenIconUrl}
        loadingAvatarId={loadingAvatarId}
        currentAvatarId={currentAvatarId}
        onProductNameClick={onProductNameClick}
      />

      <TokenCommentCard
        tokenBlueprintId={contents.tokenBlueprintId}
        loading={loading}
        hideCommentForm={isMobilePortrait}
        commentTree={commentCard.commentTree}
        commentsLoading={commentCard.commentsLoading}
        commentsError={commentCard.commentsError}
        posting={commentCard.posting}
        commentBody={commentCard.commentBody}
        expandedIds={commentCard.expandedIds}
        replyingCommentId={commentCard.replyingCommentId}
        replyBody={commentCard.replyBody}
        replyPosting={commentCard.replyPosting}
        onCommentBodyChange={commentCard.setCommentBody}
        onReplyBodyChange={commentCard.setReplyBody}
        onRefreshComments={commentCard.refreshComments}
        onPostComment={commentCard.postComment}
        onToggleExpanded={commentCard.toggleExpanded}
        onLikeComment={commentCard.likeComment}
        onDislikeComment={commentCard.dislikeComment}
        onStartReply={commentCard.startReply}
        onCancelReply={commentCard.cancelReply}
        onSubmitReply={commentCard.submitReply}
      />
    </div>
  );
}