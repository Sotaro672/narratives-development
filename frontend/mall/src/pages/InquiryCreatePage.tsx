// frontend/mall/src/pages/InquiryCreatePage.tsx

import Layout from "../components/layout/Layout";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import MediaUploader from "../components/ui/MediaUploader";
import Textbox from "../components/ui/Textbox";
import { useInquiryCreatePage } from "../features/inquiry/presentation/hooks/useInquiryCreatePage";

import "../styles/inquiry-page.css";

export default function InquiryCreatePage() {
  const {
    navigate,
    productId,
    subject,
    setSubject,
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
    handleFilesSelected,
    handleRemoveMediaItem,
    handleCarouselScroll,
    handleMoveToSlide,
    handleBackToScanResult,
  } = useInquiryCreatePage();

  const formDisabled = !productId || submitting;

  return (
    <Layout
      title="AMOL"
      mode="mypage"
      showHeader
      showFooter
      hideHamburgerMenu={false}
      hideSettingsButton={false}
      mainClassName="inquiry-page"
      footerProps={{
        variant: "action",
        buttonLabel: submitting ? "送信中" : submitted ? "送信済み" : "送信する",
        disabled: !canSubmit,
        onButtonClick: submitInquiry,
      }}
    >
      <section className="inquiry-page__container">
        <div className="inquiry-page__header">
          <p className="inquiry-page__eyebrow">CONTACT</p>
          <h1 className="inquiry-page__title">商品について問い合わせる</h1>
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

            <Input
              id="inquiry-subject"
              name="subject"
              label="件名"
              type="text"
              value={subject}
              placeholder="例: 商品の状態について"
              maxLength={120}
              disabled={formDisabled}
              onChange={(event) => setSubject(event.target.value)}
            />

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
              emptyText="画像が登録されていません。"
              selectButtonLabel="画像を選択"
              selectingButtonLabel="処理中..."
              accept="image/*"
              multiple
              items={mediaItems}
              currentIndex={currentMediaIndex}
              disabled={formDisabled}
              selecting={submitting}
              inputRef={fileInputRef}
              carouselRef={carouselRef}
              onFilesSelected={handleFilesSelected}
              onRemoveItem={handleRemoveMediaItem}
              onCarouselScroll={handleCarouselScroll}
              onMoveToSlide={handleMoveToSlide}
            />

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
  );
}