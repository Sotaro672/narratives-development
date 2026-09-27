// frontend/mall/src/pages/ChatDetailPage.tsx

import { useEffect } from "react";
import { useParams } from "react-router-dom";

import Alert from "../components/ui/Alert";
import StatePanel from "../components/ui/StatePanel";
import TextState from "../components/ui/TextState";

import InquiryClosePrompt from "../features/inquiry/presentation/components/InquiryClosePrompt";
import InquiryMessageCard from "../features/inquiry/presentation/components/InquiryMessageCard";
import InquiryReplyList from "../features/inquiry/presentation/components/InquiryReplyList";
import InquiryReplyModal from "../features/inquiry/presentation/components/InquiryReplyModal";
import { useInquiryDetailPage } from "../features/inquiry/presentation/hooks/useInquiryDetailPage";
import ResaleChatDetail from "../features/resale/presentation/components/ResaleChatDetail";
import ChatMessageBubble from "../features/shared/presentation/components/ChatMessageBubble";
import { useChatWorkspace } from "../features/shared/presentation/context/ChatWorkspaceContext";
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
  const { registerAction } = useChatWorkspace();

  const {
    inquiry,
    sortedReplies,
    loading,
    error,
    isReplyModalOpen,
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
    openReplyModal,
    closeReplyModal,
    handleReplyFilesChange,
    removeReplyFile,
    submitReply,
    handleCloseInquiry,
  } = useInquiryDetailPage();

  useEffect(() => {
    return registerAction({
      label: "返信",
      onClick: openReplyModal,
      disabled: replyActionDisabled,
    });
  }, [
    openReplyModal,
    registerAction,
    replyActionDisabled,
  ]);

  const hasInitialReturnMessage =
    inquiry?.inquiryType !== "product" &&
    !!inquiry?.content.trim();

  return (
    <>
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
              </div>
            </div>
          ) : null}
        </section>
      </div>

      <InquiryReplyModal
        open={isReplyModalOpen}
        content={replyContent}
        files={replyFiles}
        error={replyError}
        submitting={postingReply}
        canSubmit={canSubmitReply}
        onContentChange={setReplyContent}
        onFilesChange={handleReplyFilesChange}
        onRemoveFile={removeReplyFile}
        onCancel={closeReplyModal}
        onSubmit={() => {
          void submitReply();
        }}
      />
    </>
  );
}