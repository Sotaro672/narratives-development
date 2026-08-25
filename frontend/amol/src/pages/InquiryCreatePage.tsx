// frontend/amol/src/pages/InquiryCreatePage.tsx 
import Layout from "../components/layout/Layout"; 
import MediaUploader from "../components/ui/MediaUploader"; 
import { useInquiryCreatePage } from "../features/inquiry/presentation/hooks/useInquiryCreatePage"; 
import "../styles/inquiry-page.css"; 
 
export default function InquiryCreatePage() { 
  const { 
    navigate, 
    productId, 
    orderId, 
    orderItemIndex, 
    inquiryType, 
    hasReturnContext, 
    hasValidReturnContext, 
    backTo, 
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
 
  const isReturnOpened = 
    inquiryType === "return_opened"; 
 
  const formDisabled = 
    !productId || 
    submitting || 
    ( 
      hasReturnContext && 
      !hasValidReturnContext 
    ); 
 
  return ( 
    <Layout 
      title="AMOL" 
      mode="mypage" 
      showHeader 
      showBackButton 
      showFooter 
      backTo={backTo} 
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
          <h1 className="inquiry-page__title"> 
            {isReturnOpened 
              ? "開封後の返品について" 
              : "商品について問い合わせる"} 
          </h1> 
        </div> 
 
        {!productId ? ( 
          <div className="inquiry-page__notice inquiry-page__notice--error"> 
            <p>商品IDが見つかりませんでした。</p> 
            <button 
              type="button" 
              className="inquiry-page__secondary-button" 
              onClick={() => navigate("/scan/result")} 
            > 
              スキャン結果へ戻る 
            </button> 
          </div> 
        ) : null} 
 
        { 
          productId && 
          hasReturnContext && 
          !hasValidReturnContext 
        ? ( 
          <div className="inquiry-page__notice inquiry-page__notice--error"> 
            <p>返品対象の注文情報を確認できませんでした。</p> 
            <p>スキャン結果からもう一度お進みください。</p> 
            <button 
              type="button" 
              className="inquiry-page__secondary-button" 
              onClick={handleBackToScanResult} 
            > 
              スキャン結果へ戻る 
            </button> 
          </div> 
        ) : null} 
 
        {submitted ? ( 
          <div className="inquiry-page__notice inquiry-page__notice--success"> 
            <p> 
              {isReturnOpened 
                ? "開封後の返品について送信しました。" 
                : "問い合わせを送信しました。"} 
            </p> 
            <p>返信があるまでしばらくお待ちください。</p> 
            <button 
              type="button" 
              className="inquiry-page__secondary-button" 
              onClick={handleBackToScanResult} 
            > 
              スキャン結果へ戻る 
            </button> 
          </div> 
        ) : null} 
 
        {error ? ( 
          <div className="inquiry-page__notice inquiry-page__notice--error"> 
            {error} 
          </div> 
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
 
            {isReturnOpened ? ( 
              <> 
                <input type="hidden" name="orderId" value={orderId} /> 
                <input 
                  type="hidden" 
                  name="orderItemIndex" 
                  value={ 
                    orderItemIndex !== null 
                      ? String(orderItemIndex) 
                      : "" 
                  } 
                /> 
                <input 
                  type="hidden" 
                  name="inquiryType" 
                  value="return_opened" 
                /> 
              </> 
            ) : ( 
              <input 
                type="hidden" 
                name="inquiryType" 
                value="product" 
              /> 
            )} 
 
            <div className="inquiry-page__field"> 
              <label className="inquiry-page__label" htmlFor="inquiry-subject"> 
                件名 
              </label> 
              <input 
                id="inquiry-subject" 
                className="inquiry-page__input" 
                type="text" 
                value={subject} 
                placeholder={ 
                  isReturnOpened 
                    ? "例: 開封後の返品について" 
                    : "例: 商品の状態について" 
                } 
                maxLength={120} 
                disabled={formDisabled} 
                onChange={(event) => setSubject(event.target.value)} 
              /> 
            </div> 
 
            <div className="inquiry-page__field"> 
              <label className="inquiry-page__label" htmlFor="inquiry-content"> 
                問い合わせ内容 
              </label> 
              <textarea 
                id="inquiry-content" 
                className="inquiry-page__textarea" 
                value={content} 
                placeholder={ 
                  isReturnOpened 
                    ? "開封後の返品について入力してください" 
                    : "問い合わせ内容を入力してください" 
                } 
                rows={8} 
                maxLength={2000} 
                disabled={formDisabled} 
                onChange={(event) => setContent(event.target.value)} 
              /> 
              <div className="inquiry-page__counter"> 
                {content.length.toLocaleString()} / 2,000 
              </div> 
            </div> 
 
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
 
            {isReturnOpened ? ( 
              <> 
                <div className="inquiry-page__meta"> 
                  <span>注文ID</span> 
                  <code>{orderId || "-"}</code> 
                </div> 
 
                <div className="inquiry-page__meta"> 
                  <span>注文商品</span> 
                  <code> 
                    {orderItemIndex !== null 
                      ? orderItemIndex + 1 
                      : "-"} 
                  </code> 
                </div> 
              </> 
            ) : null} 
 
            <button 
              type="submit" 
              className="inquiry-page__submit-button" 
              disabled={!canSubmit} 
            > 
              {submitting ? "送信中" : "送信する"} 
            </button> 
          </form> 
        ) : null} 
      </section> 
    </Layout> 
  ); 
}