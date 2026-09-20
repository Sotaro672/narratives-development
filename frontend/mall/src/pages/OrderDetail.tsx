// frontend/mall/src/pages/OrderDetail.tsx

import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import SectionHeader from "../components/ui/SectionHeader";

import OrderDetailItemList from "../features/order/components/OrderDetailItemList";
import OrderDetailSummary from "../features/order/components/OrderDetailSummary";
import OrderPaymentSummary from "../features/order/components/OrderPaymentSummary";
import ReturnRequestModal from "../features/order/components/ReturnRequestModal";
import { useOrderDetail } from "../features/order/hooks/useOrderDetail";
import { useOrderReturn } from "../features/order/hooks/useOrderReturn";
import { useOrderTradeNavigation } from "../features/order/hooks/useOrderTradeNavigation";

import "../styles/page-layout.css";
import "../styles/order-detail-page.css";

export default function OrderDetail() {
  const navigate = useNavigate();

  const {
    order,
    loading,
    cancellingItemIndex,
    returningItemIndex,
    error,
    reload,
    cancelItem,
    returnItem,
  } = useOrderDetail();

  const {
    returnTargetIndex,
    packageState,
    reason,
    setReason,
    setPackageState,
    openReturnModal,
    closeReturnModal,
    submitReturn,
  } = useOrderReturn({
    returningItemIndex,
    returnItem,
  });

  const {
    openingItemIndex,
    error: tradeNavigationError,
    openTrade,
  } = useOrderTradeNavigation();

  const handleBack = () => {
    navigate("/wallet");
  };

  const handleOpenBrand = (brandId?: string) => {
    const id = brandId?.trim() || "";

    if (!id) {
      return;
    }

    navigate(`/brands/${encodeURIComponent(id)}`);
  };

  const showError = !loading && !order && Boolean(error);
  const showDetail = !loading && Boolean(order);

  return (
    <Layout
      title="注文詳細"
      titleClickable={false}
      showBackButton
      onBackButtonClick={handleBack}
      mode="mypage"
      showFooter
    >
      <section className="page-section order-detail-page">
        {loading ? (
          <div className="page-card">
            <p className="page-card__text">読み込み中です...</p>
          </div>
        ) : null}

        {showError ? (
          <div className="page-card">
            <SectionHeader title="注文情報を表示できません" titleAs="h2">
              <p className="page-card__text" role="alert">
                {error}
              </p>
            </SectionHeader>

            <div className="page-actions">
              <button
                type="button"
                className="page-button page-button--secondary"
                onClick={() => void reload()}
              >
                再読み込み
              </button>
            </div>
          </div>
        ) : null}

        {showDetail && order ? (
          <div className="page-stack">
            <OrderDetailSummary
              order={order}
              error={error}
              tradeNavigationError={tradeNavigationError}
            />

            <OrderDetailItemList
              order={order}
              cancellingItemIndex={cancellingItemIndex}
              returningItemIndex={returningItemIndex}
              tradeNavigatingIndex={openingItemIndex}
              onCancelItem={cancelItem}
              onReturnItem={openReturnModal}
              onOpenTrade={openTrade}
              onOpenBrand={handleOpenBrand}
            />

            <OrderPaymentSummary order={order} />
          </div>
        ) : null}
      </section>

      <ReturnRequestModal
        open={returnTargetIndex !== null}
        packageState={packageState}
        reason={reason}
        error={returnTargetIndex !== null ? error : null}
        submitting={
          returnTargetIndex !== null &&
          returningItemIndex === returnTargetIndex
        }
        onPackageStateChange={setPackageState}
        onReasonChange={setReason}
        onCancel={closeReturnModal}
        onSubmit={() => void submitReturn()}
      />
    </Layout>
  );
}