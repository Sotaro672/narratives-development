// frontend/src/pages/ContactPage.tsx
import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import Button from "../components/ui/Button";

import { useContactAuth } from "../features/contact/hooks/useContactAuth";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";
import { useContactAttachments } from "../features/contact/hooks/useContactAttachments";
import { useContactSubmit } from "../features/contact/hooks/useContactSubmit";
import ContactForm from "../features/contact/components/ContactForm";

import "../styles/page-layout.css";
import "../styles/landing-page.css";
import "../styles/contact-page.css";

export default function ContactPage() {
  const { currentUser, authResolved, isLoggedIn } = useContactAuth();
  const { isDesktop } = useContactViewport();

  const {
    mediaInputRef,
    carouselRef,
    carouselIndex,
    attachments,
    setAttachments,
    setCarouselIndex,
    handleFilesSelected,
    handleRemoveAttachment,
    handleCarouselScroll,
    handleMoveToSlide,
    revokeAllAttachmentPreviewUrls,
  } = useContactAttachments();

  const {
    name,
    setName,
    guestEmail,
    setGuestEmail,
    company,
    setCompany,
    message,
    setMessage,
    submitting,
    handleSubmit,
  } = useContactSubmit({
    currentUser,
    isLoggedIn,
    attachments,
    setAttachments,
    setCarouselIndex,
    revokeAllAttachmentPreviewUrls,
  });

  const shouldShowGuestEmailInput = authResolved && !isLoggedIn;
  const submitButtonLabel = submitting ? "送信中..." : "問い合わせる";

  return (
    <Layout
      title="AMOL"
      mode="landing"
      hideHamburgerMenu={isLoggedIn}
      hideSettingsButton={isLoggedIn}
    >
      <section className="landing-page-section landing-page-section--with-mobile-footer-action">
        <div className="landing-page-section__inner">
          <header className="how-to-use-page__header">
            <p className="how-to-use-page__eyebrow">Contact</p>
            <h1 className="how-to-use-page__title">お問い合わせ</h1>
          </header>

          <div className="landing-page-card">
            <ContactForm
              shouldShowGuestEmailInput={shouldShowGuestEmailInput}
              name={name}
              guestEmail={guestEmail}
              company={company}
              message={message}
              submitting={submitting}
              attachments={attachments}
              carouselIndex={carouselIndex}
              mediaInputRef={mediaInputRef}
              carouselRef={carouselRef}
              onNameChange={setName}
              onGuestEmailChange={setGuestEmail}
              onCompanyChange={setCompany}
              onMessageChange={setMessage}
              onFilesSelected={handleFilesSelected}
              onRemoveAttachment={handleRemoveAttachment}
              onCarouselScroll={handleCarouselScroll}
              onMoveToSlide={handleMoveToSlide}
            />

            {isDesktop ? (
              <div className="page-actions contact-page__actions">
                <Button
                  variant="primary"
                  disabled={submitting}
                  onClick={handleSubmit}
                >
                  {submitButtonLabel}
                </Button>
              </div>
            ) : null}
          </div>
        </div>
      </section>

      {!isDesktop ? (
        <FooterNav
          variant="action"
          buttonLabel={submitButtonLabel}
          disabled={submitting}
          onButtonClick={handleSubmit}
        />
      ) : null}
    </Layout>
  );
}