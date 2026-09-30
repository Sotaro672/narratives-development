// frontend/mall/src/pages/InquiryCreatePage.tsx

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileComposerFooter from "../components/layout/MobileComposerFooter";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import MediaUploader from "../components/ui/MediaUploader";
import Textbox from "../components/ui/Textbox";
import { useInquiryCreatePage } from "../features/inquiry/presentation/hooks/useInquiryCreatePage";

import "../styles/inquiry-page.css";

export default function InquiryCreatePage() {
  const isMobilePortrait = useMobilePortrait();

  const {
    navigate,
    productId,
    content,
    setContent,
    mediaItems,
    currentMediaIndex,
    fileInputRef,
    carouselRef,
    submitting,
    submitted,
    error,
    canSubmit,
    submitInquiry,
    handleFilesAdd,
    handleFilesSelected,
    handleRemoveMediaItem,
    handleRemoveMediaFile,
    handleCarouselScroll,
    handleMoveToSlide,
    handleBackToScanResult,
  } = useInquiryCreatePage();

  const formDisabled = !productId || submitting;
  const shouldShowMobileComposer =
    Boolean(productId) &&
    !submitted;

  const files =
    mediaItems.map(
      (item) => item.file,
    );

  return (
    <>
      <Layout
        title="AMOL"
        mode="mypage"
        showHeader
        showFooter={
          !isMobilePortrait ||
          !shouldShowMobileComposer
        }
        mainClassName="inquiry-page"
      >
        <section className="inquiry-page__container">
          <div className="inquiry-page__header">
            <h1 className="inquiry-page__title">
              商品について問い合わせる
            </h1>
          </div>

          {!productId ? (
            <Alert variant="error" className="inquiry-page__alert">
              <p>商品IDが見つかりませんでした。</p>
              <Button
                variant="secondary"
                size="md"
                className="inquiry-page__alert-action"
                onClick={() => navigate("/scan/result")}
              >
                スキャン結果へ戻る
              </Button>
            </Alert>
          ) : null}

          {submitted ? (
            <Alert variant="success" className="inquiry-page__alert">
              <p>問い合わせを送信しました。</p>
              <p>返信があるまでしばらくお待ちください。</p>
              <Button
                variant="secondary"
                size="md"
                className="inquiry-page__alert-action"
                onClick={handleBackToScanResult}
              >
                スキャン結果へ戻る
              </Button>
            </Alert>
          ) : null}

          {error ? (
            <Alert variant="error" className="inquiry-page__alert">
              {error}
            </Alert>
          ) : null}

          {!submitted ? (
            <form
              className="inquiry-page__form"
              onSubmit={(event) => {
                event.preventDefault();
                void submitInquiry();
              }}
            >
              <input type="hidden" name="productId" value={productId} />
              <input type="hidden" name="inquiryType" value="product" />

              <div className="inquiry-page__desktop-fields">
                <Textbox
                  id="inquiry-content"
                  name="content"
                  label="問い合わせ内容"
                  value={content}
                  placeholder="問い合わせ内容を入力してください"
                  rows={8}
                  maxLength={2000}
                  disabled={formDisabled}
                  counterText={`${content.length.toLocaleString()} / 2,000`}
                  onChange={(event) => setContent(event.target.value)}
                />

                <MediaUploader
                  label="添付画像"
                  hint="商品の状態が分かる画像を添付できます。"
                  emptyText="クリックして写真を追加"
                  selectButtonLabel="写真を追加"
                  selectingButtonLabel="処理中..."
                  accept="image/*"
                  multiple
                  items={mediaItems}
                  currentIndex={currentMediaIndex}
                  disabled={formDisabled}
                  selecting={submitting}
                  selectFromEmptyArea
                  inputRef={fileInputRef}
                  carouselRef={carouselRef}
                  onFilesSelected={handleFilesSelected}
                  onRemoveItem={handleRemoveMediaItem}
                  onCarouselScroll={handleCarouselScroll}
                  onMoveToSlide={handleMoveToSlide}
                />
              </div>

              <div className="inquiry-page__meta">
                <span>商品ID</span>
                <code>{productId || "-"}</code>
              </div>

              <Button
                type="submit"
                size="lg"
                className="inquiry-page__submit"
                disabled={!canSubmit}
              >
                {submitting ? "送信中" : "送信する"}
              </Button>
            </form>
          ) : null}
        </section>
      </Layout>

      {shouldShowMobileComposer ? (
        <MobileComposerFooter
          content={content}
          placeholder="問い合わせ内容を入力"
          files={files}
          error={error}
          submitting={submitting}
          canSubmit={canSubmit}
          disabled={!productId}
          submitLabel="送信"
          submittingLabel="送信中"
          maxLength={2000}
          maxFiles={10}
          accept="image/*"
          onContentChange={setContent}
          onFilesAdd={handleFilesAdd}
          onRemoveFile={handleRemoveMediaFile}
          onSubmit={submitInquiry}
        />
      ) : null}
    </>
  );
}