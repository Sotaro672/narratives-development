// frontend/mall/src/pages/OrderDetail.tsx

import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import SectionHeader from "../components/ui/SectionHeader";

import OrderDetailItemList from "../features/order/components/OrderDetailItemList";
import OrderDetailSummary from "../features/order/components/OrderDetailSummary";
import OrderPaymentSummary from "../features/order/components/OrderPaymentSummary";
import { useOrderDetail } from "../features/order/hooks/useOrderDetail";
import { useOrderTradeNavigation } from "../features/order/hooks/useOrderTradeNavigation";

import "../styles/page-layout.css";
import "../styles/order-detail-page.css";

export default function OrderDetail() {
  const navigate = useNavigate();

  const {
    order,
    loading,
    cancellingItemIndex,
    error,
    reload,
    cancelItem,
  } = useOrderDetail();

  const {
    openingItemIndex,
    error: tradeNavigationError,
    openTrade,
  } = useOrderTradeNavigation();

  const handleOpenBrand = (brandId?: string) => {
    const id = brandId?.trim() || "";

    if (!id) {
      return;
    }

    navigate(`/brands/${encodeURIComponent(id)}`);
  };

  const handleOpenReturn = (itemIndex: number) => {
    if (
      !order?.id ||
      !Number.isInteger(itemIndex) ||
      itemIndex < 0
    ) {
      return;
    }

    navigate(
      `/orders/${encodeURIComponent(order.id)}/items/${itemIndex}/return`,
    );
  };

  const showError = !loading && !order && Boolean(error);
  const showDetail = !loading && Boolean(order);

  return (
    <Layout title="AMOL" mode="mypage" showFooter>
      <section className="page-section order-detail-page">
        {loading ? (
          <section
            className="order-detail-page__section order-detail-page__state"
            aria-label="読み込み状態"
          >
            <p className="order-detail-page__state-text">読み込み中です...</p>
          </section>
        ) : null}

        {showError ? (
          <section className="order-detail-page__section order-detail-page__state">
            <SectionHeader title="注文情報を表示できません" titleAs="h2" />

            <p className="order-detail-page__state-text" role="alert">
              {error}
            </p>

            <div className="page-actions">
              <button
                type="button"
                className="page-button page-button--secondary"
                onClick={() => void reload()}
              >
                再読み込み
              </button>
            </div>
          </section>
        ) : null}

        {showDetail && order ? (
          <div className="page-stack order-detail-page__stack">
            <OrderDetailSummary
              order={order}
              error={error}
              tradeNavigationError={tradeNavigationError}
            />

            <OrderDetailItemList
              order={order}
              cancellingItemIndex={cancellingItemIndex}
              tradeNavigatingIndex={openingItemIndex}
              onCancelItem={cancelItem}
              onReturnItem={handleOpenReturn}
              onOpenTrade={openTrade}
              onOpenBrand={handleOpenBrand}
            />

            <OrderPaymentSummary order={order} />
          </div>
        ) : null}
      </section>
    </Layout>
  );
}