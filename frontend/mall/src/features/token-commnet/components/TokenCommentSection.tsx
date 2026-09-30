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
  replyRows?: number;
  commentTree: TokenCommentTreeNode[];
  commentsLoading: boolean;
  commentsError: string;
  posting: boolean;
  commentBody: string;
  expandedIds: Set<string>;
  replyingCommentId: string | null;
  replyBody: string;
  replyPosting: boolean;
  editingCommentId: string | null;
  editBody: string;
  editSaving: boolean;
  deletingCommentId: string | null;
  onCommentBodyChange: (value: string) => void;
  onReplyBodyChange: (value: string) => void;
  onEditBodyChange: (value: string) => void;
  onPostComment: () => Promise<void>;
  onToggleExpanded: (commentId: string) => void;
  onLikeComment: (commentId: string) => Promise<void>;
  onDislikeComment: (commentId: string) => Promise<void>;
  onStartReply: (commentId: string) => void;
  onCancelReply: () => void;
  onSubmitReply: (parentCommentId: string) => Promise<void>;
  onStartEdit: (commentId: string) => void;
  onCancelEdit: () => void;
  onSubmitEdit: () => Promise<void>;
  onDeleteComment: (commentId: string) => Promise<void>;
};

export default function TokenCommentSection({
  tokenBlueprintId,
  loading = false,
  hideCommentForm = false,
  replyRows = 6,
  commentTree,
  commentsLoading,
  commentsError,
  posting,
  commentBody,
  expandedIds,
  replyingCommentId,
  replyBody,
  replyPosting,
  editingCommentId,
  editBody,
  editSaving,
  deletingCommentId,
  onCommentBodyChange,
  onReplyBodyChange,
  onEditBodyChange,
  onPostComment,
  onToggleExpanded,
  onLikeComment,
  onDislikeComment,
  onStartReply,
  onSubmitReply,
  onStartEdit,
  onCancelEdit,
  onSubmitEdit,
  onDeleteComment,
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
  const normalizedEditingCommentId = editingCommentId?.trim() ?? "";

  const canSubmitEdit = Boolean(
    normalizedEditingCommentId &&
      editBody.trim() &&
      !editSaving &&
      deletingCommentId !== normalizedEditingCommentId,
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

  const handleSubmitEdit = () => {
    if (!canSubmitEdit) {
      return;
    }

    void onSubmitEdit();
  };

  const handleDeleteComment = async (commentId: string) => {
    const normalizedCommentId = commentId.trim();

    if (!normalizedCommentId || deletingCommentId) {
      return;
    }

    const confirmed = window.confirm(
      "このコメントを削除します。よろしいですか？",
    );

    if (!confirmed) {
      return;
    }

    await onDeleteComment(normalizedCommentId);
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
              <Alert variant="error">
                {commentsError}
              </Alert>
            ) : null}

            <TokenCommentList
              tokenBlueprintId={normalizedTokenBlueprintId}
              currentAvatarId={currentAvatarId}
              commentTree={commentTree}
              commentsLoading={commentsLoading}
              expandedIds={expandedIds}
              replyingCommentId={replyingCommentId}
              replyBody={replyBody}
              replyPosting={replyPosting}
              editingCommentId={editingCommentId}
              deletingCommentId={deletingCommentId}
              useDedicatedReplySection={hideCommentForm}
              onToggleExpanded={onToggleExpanded}
              onLike={onLikeComment}
              onDislike={onDislikeComment}
              onStartReply={onStartReply}
              onReplyBodyChange={onReplyBodyChange}
              onSubmitReply={onSubmitReply}
              onStartEdit={onStartEdit}
              onDelete={handleDeleteComment}
              onReport={handleReportComment}
            />
          </>
        )}
      </section>

      {!hideCommentForm ? (
        <ChatComposerModal
          open={Boolean(normalizedEditingCommentId)}
          title="コメントを編集"
          content={editBody}
          placeholder="コメントを編集…"
          submitting={editSaving}
          canSubmit={canSubmitEdit}
          submitLabel="保存"
          submittingLabel="保存中..."
          rows={replyRows}
          onContentChange={onEditBodyChange}
          onCancel={onCancelEdit}
          onSubmit={handleSubmitEdit}
        />
      ) : null}

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