// frontend/mall/src/pages/LandingPage.tsx

import "../styles/page-layout.css";
import "../styles/landing-page.css";
import "../styles/price-plan-page.css";
import "../styles/company-overview.css";
import "../styles/contact-page.css";

import Layout from "../components/layout/Layout";

import { useContactAttachments } from "../features/contact/hooks/useContactAttachments";
import { useContactSubmit } from "../features/contact/hooks/useContactSubmit";

import AuthenticationSection from "../features/landing/components/AuthenticationSection";
import CompanyOverviewSection from "../features/landing/components/CompanyOverviewSection";
import FleaMarketSection from "../features/landing/components/FleaMarketSection";
import LandingContactFooter from "../features/landing/components/LandingContactFooter";
import LandingContactSection from "../features/landing/components/LandingContactSection";
import LandingFeatureOverview from "../features/landing/components/LandingFeatureOverview";
import LandingHero from "../features/landing/components/LandingHero";
import PricingSection from "../features/landing/components/PricingSection";
import SalesSupportSection from "../features/landing/components/SalesSupportSection";
import { companyOverviewRows } from "../features/landing/components/companyOverview";
import {
  subscriptionPlanColumns,
  subscriptionPlanRows,
} from "../features/landing/components/pricingPlans";

import useContactSectionVisibility from "../features/landing/hooks/useContactSectionVisibility";
import useLandingAuth from "../features/landing/hooks/useLandingAuth";
import useLandingSectionScroll from "../features/landing/hooks/useLandingSectionScroll";
import useLandingViewport from "../features/landing/hooks/useLandingViewport";

export default function LandingPage() {
  const {
    currentUser,
    authResolved,
    isLoggedIn,
  } = useLandingAuth();

  const {
    isMobile,
    isDesktop,
  } = useLandingViewport();

  const {
    authenticationEyebrowRef,
    salesSupportEyebrowRef,
    fleaMarketEyebrowRef,
    scrollToAuthentication,
    scrollToSalesSupport,
    scrollToFleaMarket,
  } = useLandingSectionScroll();

  const {
    contactSectionRef,
    isContactSectionVisible,
  } = useContactSectionVisibility({
    isDesktop,
  });

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
    uploadingAttachments,
    uploadProgress,
    uploadFileProgress,
    uploadFileIndex,
    uploadFileCount,
    handleSubmit,
  } = useContactSubmit({
    currentUser,
    isLoggedIn,
    attachments,
    setAttachments,
    setCarouselIndex,
    revokeAllAttachmentPreviewUrls,
  });

  const shouldShowGuestEmailInput =
    authResolved && !isLoggedIn;

  const submitButtonLabel =
    submitting ? "送信中..." : "問い合わせる";

  const shouldShowFooterNav =
    authResolved && isLoggedIn && isMobile;

  return (
    <Layout title="AMOL" mode="landing">
      <div
        className={[
          "landing-page",
          shouldShowFooterNav ||
          (!isDesktop && isContactSectionVisible)
            ? "landing-page--with-footer-nav"
            : "",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        <LandingHero />

        <LandingFeatureOverview
          onAuthenticationClick={scrollToAuthentication}
          onFleaMarketClick={scrollToFleaMarket}
          onSalesSupportClick={scrollToSalesSupport}
        />

        <AuthenticationSection
          authenticationEyebrowRef={authenticationEyebrowRef}
        />

        <FleaMarketSection
          fleaMarketEyebrowRef={fleaMarketEyebrowRef}
        />

        <SalesSupportSection
          salesSupportEyebrowRef={salesSupportEyebrowRef}
        />

        <PricingSection
          subscriptionPlanColumns={subscriptionPlanColumns}
          subscriptionPlanRows={subscriptionPlanRows}
        />

        <CompanyOverviewSection
          companyOverviewRows={companyOverviewRows}
        />

        <LandingContactSection
          contactSectionRef={contactSectionRef}
          shouldShowGuestEmailInput={shouldShowGuestEmailInput}
          isDesktop={isDesktop}
          name={name}
          guestEmail={guestEmail}
          company={company}
          message={message}
          submitting={submitting}
          attachments={attachments}
          carouselIndex={carouselIndex}
          mediaInputRef={mediaInputRef}
          carouselRef={carouselRef}
          submitButtonLabel={submitButtonLabel}
          onNameChange={setName}
          onGuestEmailChange={setGuestEmail}
          onCompanyChange={setCompany}
          onMessageChange={setMessage}
          onFilesSelected={handleFilesSelected}
          onRemoveAttachment={handleRemoveAttachment}
          onCarouselScroll={handleCarouselScroll}
          onMoveToSlide={handleMoveToSlide}
          onSubmit={handleSubmit}
        />
      </div>

      <LandingContactFooter
        isDesktop={isDesktop}
        isContactSectionVisible={isContactSectionVisible}
        shouldShowFooterNav={shouldShowFooterNav}
        submitButtonLabel={submitButtonLabel}
        submitting={submitting}
        uploadingAttachments={uploadingAttachments}
        uploadProgress={uploadProgress}
        uploadFileProgress={uploadFileProgress}
        uploadFileIndex={uploadFileIndex}
        uploadFileCount={uploadFileCount}
        onSubmit={handleSubmit}
      />
    </Layout>
  );
}