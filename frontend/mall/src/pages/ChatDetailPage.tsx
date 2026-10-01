// frontend/mall/src/pages/ChatDetailPage.tsx

import {
  useEffect,
  useMemo,
} from "react";
import { useParams } from "react-router-dom";

import Alert from "../components/ui/Alert";
import StatePanel from "../components/ui/StatePanel";
import TextState from "../components/ui/TextState";

import InquiryClosePrompt from "../features/inquiry/presentation/components/InquiryClosePrompt";
import InquiryMessageCard from "../features/inquiry/presentation/components/InquiryMessageCard";
import InquiryReplyList from "../features/inquiry/presentation/components/InquiryReplyList";
import { useInquiryDetailPage } from "../features/inquiry/presentation/hooks/useInquiryDetailPage";
import ResaleChatDetail from "../features/resale/presentation/components/ResaleChatDetail";
import ChatInlineComposer from "../features/shared/presentation/components/ChatInlineComposer";
import ChatMessageBubble from "../features/shared/presentation/components/ChatMessageBubble";
import { useChatWorkspace } from "../features/shared/presentation/context/ChatWorkspaceContext";
import type { ChatComposerConfig } from "../features/shared/types/chatComposer";
import { getInquiryTypeLabel } from "../features/shared/types/inquiryTypes";
import TradeChatDetail from "../features/trade/presentation/components/TradeChatDetail";

import "../styles/page-layout.css";
import "../features/shared/styles/chat-detail-page.css";

type ChatDetailRouteParams = {
  inquiryId?: string;
  resaleId?: string;
  tradeId?: string;
};

export default function ChatDetailPage() {
  const { resaleId, tradeId } = useParams<ChatDetailRouteParams>();

  if (tradeId) {
    return <TradeChatDetail tradeId={tradeId} />;
  }

  if (resaleId) {
    return <ResaleChatDetail resaleId={resaleId} />;
  }

  return <InquiryChatDetail />;
}

function InquiryChatDetail() {
  const {
    registerComposer,
    setHeaderTitle,
    clearHeaderTitle,
  } = useChatWorkspace();

  const {
    inquiry,
    sortedReplies,
    loading,
    error,
    replyContent,
    replyFiles,
    replyError,
    postingReply,
    canSubmitReply,
    closingInquiry,
    closeError,
    shouldShowClosePrompt,
    replyActionDisabled,
    setReplyContent,
    addReplyFiles,
    removeReplyFile,
    submitReply,
    handleCloseInquiry,
  } = useInquiryDetailPage();

  const composer = useMemo<ChatComposerConfig>(
    () => ({
      content: replyContent,
      placeholder: "返信内容を入力",
      files: replyFiles,
      error: replyError,
      submitting: postingReply,
      canSubmit: canSubmitReply,
      disabled: replyActionDisabled,
      maxLength: null,
      maxFiles: 10,
      accept: "image/*",
      onContentChange: setReplyContent,
      onFilesAdd: addReplyFiles,
      onRemoveFile: removeReplyFile,
      onSubmit: submitReply,
    }),
    [
      replyContent,
      replyFiles,
      replyError,
      postingReply,
      canSubmitReply,
      replyActionDisabled,
      setReplyContent,
      addReplyFiles,
      removeReplyFile,
      submitReply,
    ],
  );

  useEffect(() => {
    return registerComposer(composer);
  }, [
    composer,
    registerComposer,
  ]);

  useEffect(() => {
    if (!inquiry) {
      setHeaderTitle("問い合わせ");
      return;
    }

    const inquiryLabel = getInquiryTypeLabel(inquiry.inquiryType);
    setHeaderTitle(`${inquiry.productName}/${inquiryLabel}`);
  }, [
    inquiry,
    setHeaderTitle,
  ]);

  useEffect(() => {
    return () => {
      clearHeaderTitle();
    };
  }, [clearHeaderTitle]);

  const hasInitialReturnMessage =
    inquiry?.inquiryType !== "product" &&
    !!inquiry?.content.trim();

  return (
    <div
      className="chat-detail-page-layout chat-detail-page-layout--inquiry"
      data-chat-detail
    >
      <section className="product-detail-page-layout chat-detail-page">
        {error ? (
          <Alert
            variant="error"
            className="chat-detail-page__error"
          >
            {error}
          </Alert>
        ) : null}

        {loading ? (
          <StatePanel
            variant="loading"
            title="読み込み中..."
          />
        ) : null}

        {!loading && !inquiry ? (
          <StatePanel
            variant="empty"
            title="問い合わせが見つかりません。"
          />
        ) : null}

        {!loading && inquiry ? (
          <div className="chat-detail-page__split">
            <div className="chat-detail-page__left">
              <InquiryMessageCard inquiry={inquiry} />
            </div>

            <div className="chat-detail-page__right">
              <div className="chat-detail-page__reply-section">
                {!hasInitialReturnMessage &&
                sortedReplies.length === 0 &&
                !shouldShowClosePrompt ? (
                  <TextState
                    variant="empty"
                    className="chat-detail-page__no-replies"
                  >
                    まだ返信はありません。
                  </TextState>
                ) : (
                  <div className="chat-detail-page__replies">
                    {hasInitialReturnMessage ? (
                      <ChatMessageBubble
                        senderName={inquiry.avatarName}
                        senderIcon={inquiry.avatarIcon}
                        createdAt={inquiry.createdAt}
                        content={inquiry.content}
                        isMine
                      />
                    ) : null}

                    <InquiryReplyList
                      replies={sortedReplies}
                      brandName={inquiry.brandName}
                      brandIcon={inquiry.brandIcon}
                      avatarName={inquiry.avatarName}
                      avatarIcon={inquiry.avatarIcon}
                    />

                    {shouldShowClosePrompt ? (
                      <InquiryClosePrompt
                        error={closeError}
                        closing={closingInquiry}
                        onClose={() => {
                          void handleCloseInquiry();
                        }}
                      />
                    ) : null}
                  </div>
                )}
              </div>

              <ChatInlineComposer {...composer} />
            </div>
          </div>
        ) : null}
      </section>
    </div>
  );
}