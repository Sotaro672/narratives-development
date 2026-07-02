// frontend/amol/src/pages/ContentsPage.tsx
import "../styles/page-layout.css";
import "../styles/contents-page.css";

import Layout from "../components/layout/Layout";
import ContentsDetailPanel from "../features/contents/components/ContentsDetailPanel";
import ContentsMediaPanel from "../features/contents/components/ContentsMediaPanel";
import { useContentsPage } from "../features/contents/hooks/useContentsPage";

export default function ContentsPage() {
  const page = useContentsPage();

  return (
    <Layout
      title={page.pageTitle}
      titleClickable={false}
      mode="mypage"
      showBackButton
      backTo="/wallet"
      hideHamburgerMenu
      showFooter
      disableFooterPaddingOnDesktop
      footerProps={
        page.isMobilePortrait
          ? {
              variant: "commentAction",
              value: page.commentCard.commentBody,
              placeholder: "コメントを書く…",
              buttonLabel: page.commentCard.posting ? "投稿中" : "投稿",
              disabled:
                page.commentCard.posting ||
                page.loading ||
                !page.contents.tokenBlueprintId ||
                !page.commentCard.commentBody,
              posting: page.commentCard.posting,
              onChange: page.commentCard.setCommentBody,
              onSubmit: page.commentCard.postComment,
            }
          : { variant: "default" }
      }
    >
      <section className="split-page contents-page">
        <div className="split-page-content contents-page-content">
          <ContentsMediaPanel
            loading={page.loading}
            error={page.error}
            metadataUri={page.contents.metadataUri}
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
            loadingAvatarId={page.loadingAvatarId}
            currentAvatarId={page.currentAvatarId}
            isMobilePortrait={page.isMobilePortrait}
            commentCard={page.commentCard}
            onProductNameClick={page.handleProductNameClick}
            onBrandNameClick={page.handleBrandNameClick}
            onResaleClick={page.handleOpenResalePage}
          />
        </div>
      </section>
    </Layout>
  );
}