// frontend/mall/src/pages/ContentsPage.tsx

import { useEffect, useState } from "react";

import "../styles/page-layout.css";
import "../styles/contents-page.css";

import Layout from "../components/layout/Layout";
import ContentsDetailPanel from "../features/contents/components/ContentsDetailPanel";
import ContentsMediaPanel from "../features/contents/components/ContentsMediaPanel";
import { useContentsPage } from "../features/contents/hooks/useContentsPage";
import ChatComposerModal from "../features/shared/presentation/components/ChatComposerModal";

export default function ContentsPage() {
  const page = useContentsPage();
  const [commentModalOpen, setCommentModalOpen] = useState(false);
  const [commentSubmitRequested, setCommentSubmitRequested] = useState(false);

  const commentActionDisabled =
    page.commentCard.posting ||
    page.loading ||
    !page.contents.tokenBlueprintId;

  const canSubmitComment =
    !commentActionDisabled &&
    page.commentCard.commentBody.trim().length > 0;

  useEffect(() => {
    if (
      !commentSubmitRequested ||
      page.commentCard.posting ||
      page.commentCard.commentBody.trim() !== ""
    ) {
      return;
    }

    setCommentModalOpen(false);
    setCommentSubmitRequested(false);
  }, [
    commentSubmitRequested,
    page.commentCard.commentBody,
    page.commentCard.posting,
  ]);

  const handleOpenCommentModal = () => {
    if (commentActionDisabled) {
      return;
    }

    setCommentSubmitRequested(false);
    setCommentModalOpen(true);
  };

  const handleCloseCommentModal = () => {
    if (page.commentCard.posting) {
      return;
    }

    setCommentModalOpen(false);
    setCommentSubmitRequested(false);
  };

  const handleSubmitComment = () => {
    if (!canSubmitComment) {
      return;
    }

    setCommentSubmitRequested(true);
    void page.commentCard.postComment();
  };

  return (
    <>
      <Layout
        title="AMOL"
        mode="mypage"
        showHeader={!page.isMobilePortrait}
        hideHamburgerMenu
        showFooter={!page.isMobilePortrait || !commentModalOpen}
        disableFooterPaddingOnDesktop
        footerProps={
          page.isMobilePortrait
            ? {
                variant: "default",
                centerActionLabel: "コメント",
                centerActionDisabled: commentActionDisabled,
                onCenterActionClick: handleOpenCommentModal,
              }
            : {
                variant: "default",
              }
        }
      >
        <section className="split-page contents-page">
          <div className="split-page-content contents-page-content">
            <ContentsMediaPanel
              loading={page.loading}
              error={page.error}
              metadataUri={page.contents.metadataUri}
              moderationHidden={page.moderationHidden}
              hasMediaItems={page.hasMediaItems}
              mediaItems={page.mediaItems}
              activeFileIndex={page.activeFileIndex}
              tokenName={page.tokenName}
              onPrevFile={page.handlePrevFile}
              onNextFile={page.handleNextFile}
              onSelectFile={page.setActiveFileIndex}
            />

            <ContentsDetailPanel
              contents={page.contents}
              tokenName={page.tokenName}
              tokenIconUrl={page.tokenIconUrl}
              loading={page.loading}
              isMobilePortrait={page.isMobilePortrait}
              commentCard={page.commentCard}
              resaleDisabled={page.resaleButtonDisabled}
              resaleLabel={page.resaleButtonLabel}
              onProductNameClick={page.handleProductNameClick}
              onBrandNameClick={page.handleBrandNameClick}
              onResaleClick={page.handleOpenResalePage}
            />
          </div>
        </section>
      </Layout>

      <ChatComposerModal
        open={commentModalOpen}
        title="コメントする"
        content={page.commentCard.commentBody}
        placeholder="コメントを書く…"
        error={page.commentCard.commentsError || undefined}
        submitting={page.commentCard.posting}
        canSubmit={canSubmitComment}
        submitLabel="投稿"
        submittingLabel="投稿中..."
        onContentChange={page.commentCard.setCommentBody}
        onCancel={handleCloseCommentModal}
        onSubmit={handleSubmitComment}
      />
    </>
  );
}