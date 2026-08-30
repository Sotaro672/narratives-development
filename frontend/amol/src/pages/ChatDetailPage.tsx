// frontend/amol/src/pages/ChatDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";

import InquiryClosePrompt from "../features/inquiry/presentation/components/InquiryClosePrompt";
import InquiryMessageCard from "../features/inquiry/presentation/components/InquiryMessageCard";
import InquiryReplyList from "../features/inquiry/presentation/components/InquiryReplyList";
import InquiryReplyModal from "../features/inquiry/presentation/components/InquiryReplyModal";
import { useInquiryDetailPage } from "../features/inquiry/presentation/hooks/useInquiryDetailPage";

import ResaleChatDetail from "../features/resale/presentation/components/ResaleChatDetail";

import "../styles/page-layout.css";
import "../features/inquiry/presentation/styles/inquiry-detail-page.css";

type ChatDetailRouteParams = {
  inquiryId?: string;
  resaleId?: string;
};

type InquiryChatDetailProps = {
  onBack: () => void;
};

export default function ChatDetailPage() {
  const navigate = useNavigate();
  const { resaleId } = useParams<ChatDetailRouteParams>();

  const handleBack = () => {
    navigate(-1);
  };

  if (resaleId) {
    return (
      <ResaleChatDetail
        resaleId={resaleId}
        onBack={handleBack}
      />
    );
  }

  return <InquiryChatDetail onBack={handleBack} />;
}

function InquiryChatDetail({
  onBack,
}: InquiryChatDetailProps) {
  const {
    title,
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

  return (
    <>
      <Layout
        title={title}
        showBackButton
        onBackButtonClick={onBack}
        showFooter={!isReplyModalOpen}
        mode="mypage"
        mainClassName="chat-detail-page-layout"
        actionButtonLabel="返信"
        onActionButtonClick={openReplyModal}
        actionButtonDisabled={replyActionDisabled}
        footerProps={{
          variant: "default",
          centerActionLabel: "返信",
          centerActionDisabled: replyActionDisabled,
          onCenterActionClick: openReplyModal,
        }}
      >
        <section className="page-section content-page-section chat-detail-page">
          {error ? (
            <div className="chat-detail-page__error" role="alert">
              {error}
            </div>
          ) : null}

          {loading ? (
            <div className="chat-detail-page__state">
              読み込み中...
            </div>
          ) : null}

          {!loading && !inquiry ? (
            <div className="chat-detail-page__empty">
              問い合わせが見つかりません。
            </div>
          ) : null}

          {!loading && inquiry ? (
            <div className="chat-detail-page__thread">
              <InquiryMessageCard inquiry={inquiry} />

              <div className="chat-detail-page__reply-section">
                <h3 className="chat-detail-page__section-title">
                  返信一覧
                </h3>

                {sortedReplies.length === 0 && !shouldShowClosePrompt ? (
                  <div className="chat-detail-page__no-replies">
                    まだ返信はありません。
                  </div>
                ) : (
                  <div className="chat-detail-page__replies">
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
          ) : null}
        </section>
      </Layout>

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