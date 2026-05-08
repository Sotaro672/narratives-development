// frontend/amol/src/pages/ShippingAddressPage.tsx
import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/shipping-address-page.css";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";
import ShippingAddressForm from "../features/shipping-address/components/ShippingAddressForm";
import { useShippingAddressPage } from "../features/shipping-address/hooks/useShippingAddressPage";

export default function ShippingAddressPage() {
  const { isDesktop } = useContactViewport();

  const {
    form,
    isLoading,
    isEditMode,
    isLookingUpAddress,
    zipCodeError,
    actionButtonLabel,
    actionButtonDisabled,
    handleChange,
    handleSave,
    handleSubmit,
  } = useShippingAddressPage();

  return (
    <Layout
      title={isEditMode ? "配送先情報編集" : "配送先情報登録"}
      showBackButton
      mode="default"
      backTo="/lists"
      hideHamburgerMenu
      hideSettingsButton
      actionButtonLabel={isDesktop ? actionButtonLabel : undefined}
      onActionButtonClick={isDesktop ? handleSave : undefined}
      actionButtonDisabled={actionButtonDisabled}
    >
      <section className="page-section content-page-section settings-page shipping-address-page">
        <p className="content-page-description shipping-address-page__description">
          {isEditMode
            ? "登録済みの配送先情報を編集できます。"
            : "商品のお届け先として使用する氏名・フリガナ・住所を登録してください。"}
        </p>

        <ShippingAddressForm
          form={form}
          isLoading={isLoading}
          isLookingUpAddress={isLookingUpAddress}
          zipCodeError={zipCodeError}
          onChange={handleChange}
          onSubmit={handleSubmit}
        />
      </section>

      {!isDesktop ? (
        <FooterNav
          variant="action"
          buttonLabel={actionButtonLabel}
          disabled={actionButtonDisabled}
          onButtonClick={handleSave}
        />
      ) : null}
    </Layout>
  );
}