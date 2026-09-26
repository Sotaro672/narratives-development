// frontend/mall/src/pages/ShippingAddressPage.tsx

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/shipping-address-page.css";

import Layout from "../components/layout/Layout";
import ShippingAddressForm from "../features/shipping-address/components/ShippingAddressForm";
import { useShippingAddressPage } from "../features/shipping-address/hooks/useShippingAddressPage";

export default function ShippingAddressPage() {
  const {
    form,
    isLoading,
    isEditMode,
    isLookingUpAddress,
    zipCodeError,
    actionButtonLabel,
    actionButtonDisabled,
    handleChange,
    handleSubmit,
  } = useShippingAddressPage();

  return (
    <Layout
      title={isEditMode ? "配送先情報編集" : "配送先情報登録"}
      titleClickable={false}
      mode="mypage"
      showFooter
      hideHamburgerMenu
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
          actionButtonLabel={actionButtonLabel}
          actionButtonDisabled={actionButtonDisabled}
          onChange={handleChange}
          onSubmit={handleSubmit}
        />
      </section>
    </Layout>
  );
}