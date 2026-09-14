// frontend/mall/src/features/landing/components/LandingContactSection.tsx

import type { RefObject } from "react";

import Button from "../../../components/ui/Button";
import ContactForm from "../../contact/components/ContactForm";

type LandingContactSectionProps = {
  contactSectionRef: RefObject<HTMLElement>;
  shouldShowGuestEmailInput: boolean;
  isDesktop: boolean;
  name: string;
  guestEmail: string;
  company: string;
  message: string;
  submitting: boolean;
  attachments: Parameters<typeof ContactForm>[0]["attachments"];
  carouselIndex: number;
  mediaInputRef: Parameters<typeof ContactForm>[0]["mediaInputRef"];
  carouselRef: Parameters<typeof ContactForm>[0]["carouselRef"];
  submitButtonLabel: string;
  onNameChange: Parameters<typeof ContactForm>[0]["onNameChange"];
  onGuestEmailChange: Parameters<typeof ContactForm>[0]["onGuestEmailChange"];
  onCompanyChange: Parameters<typeof ContactForm>[0]["onCompanyChange"];
  onMessageChange: Parameters<typeof ContactForm>[0]["onMessageChange"];
  onFilesSelected: Parameters<typeof ContactForm>[0]["onFilesSelected"];
  onRemoveAttachment: Parameters<typeof ContactForm>[0]["onRemoveAttachment"];
  onCarouselScroll: Parameters<typeof ContactForm>[0]["onCarouselScroll"];
  onMoveToSlide: Parameters<typeof ContactForm>[0]["onMoveToSlide"];
  onSubmit: () => void;
};

export default function LandingContactSection({
  contactSectionRef,
  shouldShowGuestEmailInput,
  isDesktop,
  name,
  guestEmail,
  company,
  message,
  submitting,
  attachments,
  carouselIndex,
  mediaInputRef,
  carouselRef,
  submitButtonLabel,
  onNameChange,
  onGuestEmailChange,
  onCompanyChange,
  onMessageChange,
  onFilesSelected,
  onRemoveAttachment,
  onCarouselScroll,
  onMoveToSlide,
  onSubmit,
}: LandingContactSectionProps) {
  return (
    <section
      ref={contactSectionRef}
      id="contact"
      className="landing-page-section landing-page-section--with-mobile-footer-action"
    >
      <div className="landing-page-section__inner">
        <header className="how-to-use-page__header">
          <p className="how-to-use-page__eyebrow">
            Contact
          </p>

          <h2 className="how-to-use-page__title">
            お問い合わせ
          </h2>
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
            onNameChange={onNameChange}
            onGuestEmailChange={onGuestEmailChange}
            onCompanyChange={onCompanyChange}
            onMessageChange={onMessageChange}
            onFilesSelected={onFilesSelected}
            onRemoveAttachment={onRemoveAttachment}
            onCarouselScroll={onCarouselScroll}
            onMoveToSlide={onMoveToSlide}
          />

          {isDesktop ? (
            <div className="page-actions contact-page__actions">
              <Button
                variant="primary"
                disabled={submitting}
                onClick={onSubmit}
              >
                {submitButtonLabel}
              </Button>
            </div>
          ) : null}
        </div>
      </div>
    </section>
  );
}