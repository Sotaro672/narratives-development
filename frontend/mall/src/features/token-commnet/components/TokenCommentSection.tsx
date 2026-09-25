// frontend/mall/src/features/token-commnet/components/TokenCommentSection.tsx

import { useEffect, useState } from "react";

import Alert from "../../../components/ui/Alert";
import TextState from "../../../components/ui/TextState";
import { getMyAvatar } from "../../avatar/api/avatarApi";
import ReportModal from "../../report/components/ReportModal";
import { useReport } from "../../report/hooks/useReport";
import { useAuthState } from "../../shared/hooks/useAuthState";
import ChatComposerModal from "../../shared/presentation/components/ChatComposerModal";
import type { TokenCommentTreeNode } from "../../shared/types/tokenCommentTypes";

import TokenCommentForm from "./TokenCommentForm";
import TokenCommentList from "./TokenCommentList";

type TokenCommentSectionProps = {
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
  onCommentBodyChange: (value: string) => void;
  onReplyBodyChange: (value: string) => void;
  onPostComment: () => Promise<void>;
  onToggleExpanded: (commentId: string) => void;
  onLikeComment: (commentId: string) => Promise<void>;
  onDislikeComment: (commentId: string) => Promise<void>;
  onStartReply: (commentId: string) => void;
  onCancelReply: () => void;
  onSubmitReply: (parentCommentId: string) => Promise<void>;
};

export default function TokenCommentSection({
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
}: TokenCommentSectionProps) {
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

  const normalizedTokenBlueprintId = tokenBlueprintId.trim();
  const canSubmitReply = Boolean(
    replyingCommentId && replyBody.trim() && !replyPosting,
  );

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

        setCurrentAvatarId(avatar?.avatarId?.trim() ?? "");
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
  }, [authResolved, isLoggedIn]);

  const handleReportComment = (commentId: string) => {
    const normalizedCommentId = commentId.trim();

    if (
      !isLoggedIn ||
      !currentAvatarId ||
      !normalizedTokenBlueprintId ||
      !normalizedCommentId
    ) {
      return;
    }

    openTokenBlueprintCommentReport({
      tokenBlueprintId: normalizedTokenBlueprintId,
      commentId: normalizedCommentId,
    });
  };

  const handleSubmitReply = () => {
    if (!replyingCommentId || !canSubmitReply) {
      return;
    }

    void onSubmitReply(replyingCommentId);
  };

  return (
    <>
      <section>
        {!normalizedTokenBlueprintId ? (
          <TextState variant="muted">
            tokenBlueprintId 未取得のためコメントを表示できません。
          </TextState>
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
              <Alert variant="error">{commentsError}</Alert>
            ) : null}

            <TokenCommentList
              tokenBlueprintId={normalizedTokenBlueprintId}
              currentAvatarId={currentAvatarId}
              commentTree={commentTree}
              commentsLoading={commentsLoading}
              expandedIds={expandedIds}
              onToggleExpanded={onToggleExpanded}
              onLike={onLikeComment}
              onDislike={onDislikeComment}
              onStartReply={onStartReply}
              onReport={handleReportComment}
            />
          </>
        )}
      </section>

      <ChatComposerModal
        open={Boolean(replyingCommentId)}
        title="返信する"
        content={replyBody}
        placeholder="返信を書く…"
        submitting={replyPosting}
        canSubmit={canSubmitReply}
        submitLabel="返信を投稿"
        submittingLabel="投稿中..."
        onContentChange={onReplyBodyChange}
        onCancel={onCancelReply}
        onSubmit={handleSubmitReply}
      />

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